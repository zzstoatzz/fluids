const std = @import("std");
const Sim = @import("Sim.zig");

pub fn main(init: std.process.Init) !void {
    const n = 1000;
    var sim = try Sim.init(init.gpa, n, .{ .x = 1024, .y = 768 }, .{}, 1);
    defer sim.deinit();
    const steps = 2000;
    const t0 = std.Io.Timestamp.now(init.io, .awake);
    for (0..steps) |_| _ = try sim.step(0, 10, sim.dt);
    const ns: u64 = @intCast(t0.untilNow(init.io, .awake).nanoseconds);
    std.debug.print("{d} particles, {d} steps: {d:.3} ms/step\n", .{ n, steps, @as(f64, @floatFromInt(ns)) / steps / 1e6 });
}
