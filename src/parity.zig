//! replays cases recorded by the go oracle (go/cmd/oracle) through the zig sim
//! and checks two properties:
//!   1. with the n-body term off and the oracle run with -portable-pow, every
//!      step (positions, velocities, forces, density, pressure, mean pressure)
//!      is bit-identical to go
//!   2. otherwise results agree to 1e-9 relative. go reduces the n-body forces
//!      in map-iteration order, and stock math.Pow goes through an amd64 asm
//!      exp whose bits depend on the cpu, so go doesn't agree with itself
//!      beyond that either.
//! std pressure is always compared with tolerance: go sums it in buckets keyed
//! by runtime.NumCPU(), so its bits depend on the machine.
const std = @import("std");
const Sim = @import("Sim.zig");

const GoParams = struct {
    InteractionRadius: f64,
    SmoothingFactor: f64,
    DampeningFactor: f64,
    DragEnabled: bool,
    AttractionFactor: f64,
    Rho0: f64,
    Nu: f64,
    PressureMultiplier: f64,
    Dt: f64,
    Gravity: f64,
    MouseForce: f64,
    MouseForceRadius: f64,
};

const State = struct {
    X: []f64,
    Y: []f64,
    Vx: []f64,
    Vy: []f64,
    Density: []f64,
    Pressure: []f64,
    Fx: ?[]f64 = null,
    Fy: ?[]f64 = null,
};

const Poke = struct { Before: usize, X: f64, Y: f64, Force: f64, Radius: f64 };

const Case = struct {
    Seed: i64,
    PortablePow: bool = false,
    N: usize,
    DomainX: f64,
    DomainY: f64,
    Params: GoParams,
    InteractionRadius: f64,
    Gravity: f64,
    Pressure: f64,
    Radius: []f64,
    Mass: []f64,
    Init: State,
    Pokes: ?[]Poke,
    Tape: ?[]f64,
    Steps: []State,
    Mean: []f64,
    Std: []f64,
};

const tol = 1e-9;

pub fn main(init: std.process.Init) !void {
    const gpa = init.gpa;
    const args = try init.minimal.args.toSlice(init.arena.allocator());
    const path = if (args.len > 1) args[1] else "parity.json";
    const raw = try std.Io.Dir.cwd().readFileAlloc(init.io, path, gpa, .limited(1 << 30));
    defer gpa.free(raw);
    const parsed = try std.json.parseFromSlice([]Case, gpa, raw, .{ .ignore_unknown_fields = true });
    defer parsed.deinit();

    var exact_cases: usize = 0;
    var exact_ok: usize = 0;
    var tol_cases: usize = 0;
    var tol_ok: usize = 0;
    var worst: f64 = 0;
    var worst_seed: i64 = 0;
    var steps_total: usize = 0;

    for (parsed.value) |c| {
        const p = c.Params;
        var sim = try Sim.initEmpty(gpa, c.N, .{ .x = c.DomainX, .y = c.DomainY }, .{
            .interaction_radius = p.InteractionRadius,
            .smoothing_factor = p.SmoothingFactor,
            .dampening_factor = p.DampeningFactor,
            .drag_enabled = p.DragEnabled,
            .attraction_factor = p.AttractionFactor,
            .rho0 = p.Rho0,
            .nu = p.Nu,
            .pressure_multiplier = p.PressureMultiplier,
            .dt = p.Dt,
        }, .{ .tape = .{ .values = c.Tape orelse &.{0.5} } });
        defer sim.deinit();
        sim.interaction_radius = c.InteractionRadius;
        @memcpy(sim.x, c.Init.X);
        @memcpy(sim.y, c.Init.Y);
        @memcpy(sim.vx, c.Init.Vx);
        @memcpy(sim.vy, c.Init.Vy);
        @memcpy(sim.density, c.Init.Density);
        @memcpy(sim.radius, c.Radius);
        @memcpy(sim.mass, c.Mass);

        const exact = p.AttractionFactor == 0 and c.PortablePow;
        var case_ok = true;
        var case_err: f64 = 0;
        const pokes = c.Pokes orelse &.{};
        var pi: usize = 0;
        for (c.Steps, 0..) |want, k| {
            while (pi < pokes.len and pokes[pi].Before == k) : (pi += 1) {
                sim.applyMouseForce(pokes[pi].X, pokes[pi].Y, pokes[pi].Force, pokes[pi].Radius);
            }
            const st = try sim.step(c.Gravity, c.Pressure, sim.dt);
            steps_total += 1;
            const fields = [_][2][]const f64{
                .{ sim.density, want.Density }, .{ sim.pressure, want.Pressure },
                .{ sim.fx, want.Fx.? },         .{ sim.fy, want.Fy.? },
                .{ sim.x, want.X },             .{ sim.y, want.Y },
                .{ sim.vx, want.Vx },           .{ sim.vy, want.Vy },
            };
            for (fields) |f| {
                const e = compare(f[0], f[1], exact);
                case_err = @max(case_err, e);
                if (e > @as(f64, if (exact) 0 else tol)) case_ok = false;
            }
            const em = relErr(st.mean_pressure, c.Mean[k], @abs(c.Mean[k]) + 1);
            if (em > @as(f64, if (exact) 0 else tol)) case_ok = false;
            if (relErr(st.std_pressure, c.Std[k], @abs(c.Std[k]) + 1) > tol) case_ok = false;
            case_err = @max(case_err, em);
        }
        if (exact) {
            exact_cases += 1;
            if (case_ok) exact_ok += 1;
        } else {
            tol_cases += 1;
            if (case_ok) tol_ok += 1;
        }
        if (!case_ok) std.debug.print("FAIL seed={d} n={d} attraction={d} err={e}\n", .{ c.Seed, c.N, p.AttractionFactor, case_err });
        if (case_err > worst) {
            worst = case_err;
            worst_seed = c.Seed;
        }
    }
    std.debug.print(
        \\parity: {d} cases, {d} steps
        \\  bit-exact cases:      {d}/{d}
        \\  within {e} cases: {d}/{d}
        \\  worst relative error: {e} (seed {d})
        \\
    , .{ parsed.value.len, steps_total, exact_ok, exact_cases, tol, tol_ok, tol_cases, worst, worst_seed });
    if (exact_ok != exact_cases or tol_ok != tol_cases) std.process.exit(1);
}

/// max error over a field. exact mode counts any bit difference as error 1.
/// tolerance mode scales by the field's largest magnitude, since n-body
/// summation-order noise is relative to the biggest terms, not each element.
fn compare(got: []const f64, want: []const f64, exact: bool) f64 {
    var scale: f64 = 1;
    for (want) |w| scale = @max(scale, @abs(w));
    var worst: f64 = 0;
    for (got, want) |g, w| {
        if (exact) {
            if (@as(u64, @bitCast(g)) != @as(u64, @bitCast(w))) worst = @max(worst, @max(relErr(g, w, scale), 1e-300));
        } else worst = @max(worst, relErr(g, w, scale));
    }
    return worst;
}

fn relErr(a: f64, b: f64, scale: f64) f64 {
    if (std.math.isNan(a) or std.math.isNan(b)) return if (std.math.isNan(a) and std.math.isNan(b)) 0 else std.math.inf(f64);
    return @abs(a - b) / scale;
}
