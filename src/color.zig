//! pressure → color mapping, ported from go/viz/render.go (initColorCache,
//! getColor and the sigmoid normalization in RenderFrame).
const std = @import("std");

pub const Rgb = extern struct { r: u8, g: u8, b: u8 };

pub const palette: [256]Rgb = blk: {
    @setEvalBranchQuota(10_000);
    var out: [256]Rgb = undefined;
    for (0..256) |i| {
        const np: f64 = @as(f64, @floatFromInt(i)) / 255.0;
        if (np < 0.5) {
            const t = np * 2.0;
            out[i] = .{ .r = u8t(10 + 70 * t), .g = u8t(120 * t), .b = u8t(180 + 50 * t) };
        } else {
            const t = (np - 0.5) * 2.0;
            out[i] = .{ .r = u8t(80 + 175 * t), .g = u8t(120 + 135 * t), .b = u8t(230 + 25 * t) };
        }
    }
    break :blk out;
};

fn u8t(f: f64) u8 {
    return @intFromFloat(@trunc(f));
}

/// palette index for each particle, same normalization as RenderFrame.
pub fn indices(pressure: []const f64, mean: f64, stddev: f64, out: []u8) void {
    var scale = @max(stddev, 1.0) * 3.0;
    if (scale < 1000.0) scale = 1000.0;
    for (pressure, out) |p, *o| {
        var np = 1.0 / (1.0 + @exp(-((p - mean) / scale * 2.0)));
        if (np < 0) np = 0 else if (np > 1) np = 1;
        if (std.math.isNan(np)) np = 0;
        o.* = @intFromFloat(np * 255);
    }
}

test "palette endpoints" {
    try std.testing.expectEqual(Rgb{ .r = 10, .g = 0, .b = 180 }, palette[0]);
    try std.testing.expectEqual(Rgb{ .r = 255, .g = 255, .b = 255 }, palette[255]);
}
