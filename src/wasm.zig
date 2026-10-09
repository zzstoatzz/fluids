//! wasm entry point. js owns the canvas + input; zig owns the sim and hands
//! back pointers into linear memory for positions and palette indices.
const std = @import("std");
const Sim = @import("Sim.zig");
const color = @import("color.zig");

const allocator = std.heap.wasm_allocator;

var sim: ?Sim = null;
var params: Sim.Params = .{};
var colors: []u8 = &.{};
var stats: Sim.Stats = .{ .mean_pressure = 0, .std_pressure = 0 };
// positions as f32 for cheap drawing on the js side
var pos: []f32 = &.{};

export fn init(n: u32, domain_x: f64, domain_y: f64, seed: u32) bool {
    if (sim) |*s| s.deinit();
    allocator.free(colors);
    allocator.free(pos);
    sim = Sim.init(allocator, n, .{ .x = domain_x, .y = domain_y }, params, seed) catch return false;
    colors = allocator.alloc(u8, n) catch return false;
    pos = allocator.alloc(f32, n * 2) catch return false;
    @memset(colors, 128);
    syncPositions();
    return true;
}

export fn step(steps: u32, gravity: f64, pressure_multiplier: f64) bool {
    const s = &(sim orelse return false);
    for (0..steps) |_| stats = s.step(gravity, pressure_multiplier, s.dt) catch return false;
    color.indices(s.pressure, stats.mean_pressure, stats.std_pressure, colors);
    syncPositions();
    return true;
}

fn syncPositions() void {
    const s = &(sim orelse return);
    for (0..s.n) |i| {
        pos[2 * i] = @floatCast(s.x[i]);
        pos[2 * i + 1] = @floatCast(s.y[i]);
    }
}

export fn poke(x: f64, y: f64, force: f64, radius: f64) void {
    const s = &(sim orelse return);
    s.applyMouseForce(x, y, force, radius);
}

export fn positionsPtr() [*]f32 {
    return pos.ptr;
}
export fn colorsPtr() [*]u8 {
    return colors.ptr;
}
export fn palettePtr() [*]const u8 {
    return @ptrCast(&color.palette);
}
export fn meanPressure() f64 {
    return stats.mean_pressure;
}
export fn stdPressure() f64 {
    return stats.std_pressure;
}

// runtime tunables (the go version's keyboard controls)
export fn setParam(which: u32, value: f64) void {
    const s = &(sim orelse return);
    switch (which) {
        0 => s.dampening_factor = value,
        1 => s.smoothing_factor = value,
        2 => s.attraction_factor = value,
        3 => s.interaction_radius = value,
        4 => s.nu = value,
        5 => s.dt = value,
        6 => s.drag_enabled = value != 0,
        else => {},
    }
}
export fn getParam(which: u32) f64 {
    const s = &(sim orelse return 0);
    return switch (which) {
        0 => s.dampening_factor,
        1 => s.smoothing_factor,
        2 => s.attraction_factor,
        3 => s.interaction_radius,
        4 => s.nu,
        5 => s.dt,
        6 => if (s.drag_enabled) 1 else 0,
        7 => s.grid.cell_size,
        else => 0,
    };
}
