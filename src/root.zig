//! fluids: 2d sph fluid sim in zig.
pub const Sim = @import("Sim.zig");
pub const Grid = @import("Grid.zig");
pub const kernel = @import("kernel.zig");
pub const color = @import("color.zig");

test {
    @import("std").testing.refAllDecls(@This());
}
