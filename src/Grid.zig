//! uniform spatial hash, a port of go/spatial/grid.go.
//!
//! go keeps `map[CellIndex][]int`; we counting-sort particle indices into one
//! flat array so each cell is a contiguous slice. within a cell, indices stay
//! ascending (same order go appends them), so neighbor iteration order matches.
const std = @import("std");
const Allocator = std.mem.Allocator;

const Grid = @This();

pub const Cell = struct { x: i64, y: i64, start: u32, len: u32 };

cell_size: f64,
/// per-particle cell coords from the last update (go: Particle.CellX/CellY)
cell_x: []i64 = &.{},
cell_y: []i64 = &.{},
cells: std.ArrayList(Cell) = .empty,
sorted: std.ArrayList(u32) = .empty,
lookup: std.AutoHashMapUnmanaged(i64, u32) = .empty,
slot_of: []u32 = &.{},

pub fn init(cell_size: f64) Grid {
    return .{ .cell_size = cell_size };
}

pub fn deinit(g: *Grid, a: Allocator) void {
    a.free(g.cell_x);
    a.free(g.cell_y);
    a.free(g.slot_of);
    g.cells.deinit(a);
    g.sorted.deinit(a);
    g.lookup.deinit(a);
}

/// go's MakeCellIndex: x in the high 32 bits, low 32 bits of y in the low half.
pub inline fn key(x: i64, y: i64) i64 {
    return @bitCast((@as(u64, @bitCast(x)) << 32) | (@as(u64, @bitCast(y)) & 0xFFFFFFFF));
}

fn goInt(f: f64) i64 {
    if (!(f > -9.223372036854775808e18 and f < 9.223372036854775808e18)) return std.math.minInt(i64);
    return @intFromFloat(@trunc(f));
}

pub fn update(g: *Grid, a: Allocator, xs: []const f64, ys: []const f64) !void {
    const n = xs.len;
    if (g.cell_x.len != n) {
        a.free(g.cell_x);
        a.free(g.cell_y);
        a.free(g.slot_of);
        g.cell_x = try a.alloc(i64, n);
        g.cell_y = try a.alloc(i64, n);
        g.slot_of = try a.alloc(u32, n);
    }
    g.cells.clearRetainingCapacity();
    g.lookup.clearRetainingCapacity();

    // pass 1: assign cells, count
    for (0..n) |i| {
        const cx = goInt(xs[i] / g.cell_size);
        const cy = goInt(ys[i] / g.cell_size);
        g.cell_x[i] = cx;
        g.cell_y[i] = cy;
        const gop = try g.lookup.getOrPut(a, key(cx, cy));
        if (!gop.found_existing) {
            gop.value_ptr.* = @intCast(g.cells.items.len);
            try g.cells.append(a, .{ .x = cx, .y = cy, .start = 0, .len = 0 });
        }
        g.slot_of[i] = gop.value_ptr.*;
        g.cells.items[gop.value_ptr.*].len += 1;
    }
    // pass 2: prefix sums
    var off: u32 = 0;
    for (g.cells.items) |*c| {
        c.start = off;
        off += c.len;
        c.len = 0;
    }
    // pass 3: stable scatter
    try g.sorted.resize(a, n);
    for (0..n) |i| {
        const c = &g.cells.items[g.slot_of[i]];
        g.sorted.items[c.start + c.len] = @intCast(i);
        c.len += 1;
    }
}

pub inline fn cell(g: *const Grid, x: i64, y: i64) []const u32 {
    const slot = g.lookup.get(key(x, y)) orelse return &.{};
    const c = g.cells.items[slot];
    return g.sorted.items[c.start..][0..c.len];
}

test "cells keep ascending index order" {
    var g = Grid.init(10);
    defer g.deinit(std.testing.allocator);
    const xs = [_]f64{ 1, 25, 2, 3, 26 };
    const ys = [_]f64{ 1, 1, 1, 1, 1 };
    try g.update(std.testing.allocator, &xs, &ys);
    try std.testing.expectEqualSlices(u32, &.{ 0, 2, 3 }, g.cell(0, 0));
    try std.testing.expectEqualSlices(u32, &.{ 1, 4 }, g.cell(2, 0));
    try std.testing.expectEqual(@as(usize, 0), g.cell(1, 0).len);
}
