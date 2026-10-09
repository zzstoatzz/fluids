//! 2d sph fluid sim, a port of go/simulation (FluidSim) to zig.
//!
//! behavior matches the go version step for step (see `zig build parity`),
//! with two deliberate structural changes that don't change results:
//!   - particles are stored struct-of-arrays instead of []Particle
//!   - everything runs single-threaded, in index order. the go version's
//!     parallelFor only changes *when* things run, except for the n-body
//!     reduction whose float summation order is map-iteration-random in go
//!     anyway, and the boundary jitter whose draw order is goroutine-random.
const std = @import("std");
const kernel = @import("kernel.zig");
const Grid = @import("Grid.zig");

const Sim = @This();
const Allocator = std.mem.Allocator;

pub const Params = struct {
    /// kernel radius h; also the grid cell size at construction time
    interaction_radius: f64 = 10.0,
    smoothing_factor: f64 = 0.10,
    dampening_factor: f64 = 0.2,
    drag_enabled: bool = true,
    attraction_factor: f64 = -50000.0,
    rho0: f64 = 1000.0,
    nu: f64 = 0.8,
    pressure_multiplier: f64 = 10.0,
    dt: f64 = 0.0008,
    gravity: f64 = 0,
    mouse_force: f64 = 1000.0,
    mouse_force_radius: f64 = 100.0,
};

pub const BoundaryType = enum { reflective, periodic };

pub const Domain = struct { x: f64, y: f64 };

/// where boundary-jitter randomness comes from. `tape` replays draws recorded
/// from the go oracle so parity runs are deterministic.
pub const JitterSource = union(enum) {
    prng: std.Random.DefaultPrng,
    tape: struct { values: []const f64, pos: usize = 0 },

    pub fn next(self: *JitterSource) f64 {
        switch (self.*) {
            .prng => |*p| return p.random().float(f64),
            .tape => |*t| {
                const v = t.values[t.pos % t.values.len];
                t.pos += 1;
                return v;
            },
        }
    }
};

pub const Stats = struct { mean_pressure: f64, std_pressure: f64 };

// boundary constants, from go/spatial/boundaries.go
const boundary_epsilon = 0.001;
const boundary_dampening = 0.8;
const randomization_factor = 0.05;

allocator: Allocator,
n: usize,

// particle state (soa)
x: []f64,
y: []f64,
vx: []f64,
vy: []f64,
density: []f64,
pressure: []f64,
fx: []f64,
fy: []f64,
mass: []f64,
radius: []f64,

// tunables that can change at runtime (mirrors the exported FluidSim fields)
dt: f64,
rho0: f64,
nu: f64,
domain: Domain,
smoothing_factor: f64,
dampening_factor: f64,
drag_enabled: bool,
attraction_factor: f64,
interaction_radius: f64,
left_boundary: BoundaryType = .reflective,
top_boundary: BoundaryType = .reflective,

grid: Grid,
/// csr neighbor lists: neighbors of i are nbr_idx[nbr_start[i]..nbr_start[i+1]]
nbr_start: []u32,
nbr_idx: std.ArrayList(u32) = .empty,
/// scratch for the n-body pass
ax: []f64,
ay: []f64,

jitter: JitterSource,

pub fn init(allocator: Allocator, n: usize, domain: Domain, params: Params, seed: u64) !Sim {
    var sim = try initEmpty(allocator, n, domain, params, .{ .prng = std.Random.DefaultPrng.init(seed ^ 0x9e3779b97f4a7c15) });
    var prng = std.Random.DefaultPrng.init(seed);
    sim.randomize(prng.random());
    return sim;
}

/// allocate a sim with zeroed particles; callers fill x/y/vx/vy/radius/mass.
pub fn initEmpty(allocator: Allocator, n: usize, domain: Domain, params: Params, jitter: JitterSource) !Sim {
    const sim: Sim = .{
        .allocator = allocator,
        .n = n,
        .x = try allocator.alloc(f64, n),
        .y = try allocator.alloc(f64, n),
        .vx = try allocator.alloc(f64, n),
        .vy = try allocator.alloc(f64, n),
        .density = try allocator.alloc(f64, n),
        .pressure = try allocator.alloc(f64, n),
        .fx = try allocator.alloc(f64, n),
        .fy = try allocator.alloc(f64, n),
        .mass = try allocator.alloc(f64, n),
        .radius = try allocator.alloc(f64, n),
        .ax = try allocator.alloc(f64, n),
        .ay = try allocator.alloc(f64, n),
        .nbr_start = try allocator.alloc(u32, n + 1),
        .dt = params.dt,
        .rho0 = params.rho0,
        .nu = params.nu,
        .domain = domain,
        .smoothing_factor = params.smoothing_factor,
        .dampening_factor = params.dampening_factor,
        .drag_enabled = params.drag_enabled,
        .attraction_factor = params.attraction_factor,
        .interaction_radius = params.interaction_radius,
        .grid = Grid.init(params.interaction_radius),
        .jitter = jitter,
    };
    for ([_][]f64{ sim.x, sim.y, sim.vx, sim.vy, sim.pressure, sim.fx, sim.fy, sim.mass, sim.radius }) |s| @memset(s, 0);
    @memset(sim.density, params.rho0);
    @memset(sim.nbr_start, 0);
    return sim;
}

pub fn deinit(sim: *Sim) void {
    const a = sim.allocator;
    for ([_][]f64{ sim.x, sim.y, sim.vx, sim.vy, sim.density, sim.pressure, sim.fx, sim.fy, sim.mass, sim.radius, sim.ax, sim.ay }) |s| a.free(s);
    a.free(sim.nbr_start);
    sim.nbr_idx.deinit(a);
    sim.grid.deinit(a);
}

/// RandomStillInitialCondition + the radius/mass setup from NewFluidSim.
pub fn randomize(sim: *Sim, r: std.Random) void {
    for (0..sim.n) |i| {
        sim.x[i] = r.float(f64) * sim.domain.x;
        sim.y[i] = r.float(f64) * sim.domain.y;
        sim.vx[i] = (r.float(f64) * 0.2) - 0.1;
        sim.vy[i] = (r.float(f64) * 0.2) - 0.1;
        sim.density[i] = sim.rho0;
        sim.pressure[i] = 0;
        sim.radius[i] = 2.0 * (0.8 + 0.4 * r.float(f64));
        sim.mass[i] = std.math.pi * sim.radius[i] * sim.radius[i];
    }
}

/// advance one step. same phase order as FluidSim.Step.
pub fn step(sim: *Sim, gravity: f64, pressure_multiplier: f64, dt: f64) !Stats {
    @memset(sim.fx, 0);
    @memset(sim.fy, 0);
    sim.predictPositions(dt);
    try sim.grid.update(sim.allocator, sim.x, sim.y);
    try sim.findNeighbors();
    sim.updateDensities();
    sim.updatePressure(pressure_multiplier);
    sim.applyGravity(gravity);
    sim.standardForces(pressure_multiplier);
    sim.attractionForces();
    sim.applyDrag(dt);
    sim.integrate(dt);
    return sim.pressureStats();
}

fn predictPositions(sim: *Sim, dt: f64) void {
    for (sim.x, sim.vx) |*x, vx| x.* += vx * dt;
    for (sim.y, sim.vy) |*y, vy| y.* += vy * dt;
}

fn findNeighbors(sim: *Sim) !void {
    const r2 = sim.interaction_radius * sim.interaction_radius;
    sim.nbr_idx.clearRetainingCapacity();
    for (0..sim.n) |i| {
        sim.nbr_start[i] = @intCast(sim.nbr_idx.items.len);
        const cx = sim.grid.cell_x[i];
        const cy = sim.grid.cell_y[i];
        var dx: i64 = -1;
        while (dx <= 1) : (dx += 1) {
            var dy: i64 = -1;
            while (dy <= 1) : (dy += 1) {
                for (sim.grid.cell(cx +% dx, cy +% dy)) |j| {
                    if (j == i) continue;
                    const ddx = sim.x[i] - sim.x[j];
                    const ddy = sim.y[i] - sim.y[j];
                    if (ddx * ddx + ddy * ddy < r2) try sim.nbr_idx.append(sim.allocator, j);
                }
            }
        }
    }
    sim.nbr_start[sim.n] = @intCast(sim.nbr_idx.items.len);
}

pub inline fn neighbors(sim: *const Sim, i: usize) []const u32 {
    return sim.nbr_idx.items[sim.nbr_start[i]..sim.nbr_start[i + 1]];
}

fn updateDensities(sim: *Sim) void {
    const h = sim.interaction_radius;
    const h2 = h * h;
    const self_term = kernel.smoothing(0, h);
    for (0..sim.n) |i| {
        var d: f64 = 0;
        for (sim.neighbors(i)) |j| {
            const dx = sim.x[i] - sim.x[j];
            const dy = sim.y[i] - sim.y[j];
            const d2 = dx * dx + dy * dy;
            if (d2 < h2) d += kernel.smoothing(@sqrt(d2), h);
        }
        sim.density[i] = d + self_term;
    }
}

fn updatePressure(sim: *Sim, pm: f64) void {
    for (sim.pressure, sim.density) |*p, d| p.* = pm * (d - sim.rho0);
}

fn applyGravity(sim: *Sim, gravity: f64) void {
    for (sim.fy, sim.mass) |*fy, m| {
        if (m > 0) fy.* += gravity * m;
    }
}

/// pressure + viscosity + repulsion, summed per particle exactly like
/// CalculateStandardForces (three separate accumulators, then added).
fn standardForces(sim: *Sim, pm: f64) void {
    const h = sim.interaction_radius;
    const h2 = h * h;
    const min_dist = sim.smoothing_factor * h;
    for (0..sim.n) |i| {
        var px: f64 = 0;
        var py: f64 = 0;
        var visx: f64 = 0;
        var visy: f64 = 0;
        var rx: f64 = 0;
        var ry: f64 = 0;
        const nbrs = sim.neighbors(i);

        // pressure
        for (nbrs) |j| {
            const dx = sim.x[j] - sim.x[i];
            const dy = sim.y[j] - sim.y[i];
            const d2 = dx * dx + dy * dy;
            if (d2 < kernel.epsilon or d2 >= h2) continue;
            const sd = kernel.goMax(@sqrt(d2), min_dist);
            const inv = 1.0 / sd;
            const avg_p = (sim.pressure[i] + sim.pressure[j]) * 0.5;
            const mag = avg_p * kernel.derivative(sd, h) * pm;
            px += dx * inv * mag;
            py += dy * inv * mag;
        }
        // viscosity
        for (nbrs) |j| {
            const dvx = sim.vx[j] - sim.vx[i];
            const dvy = sim.vy[j] - sim.vy[i];
            const dx = sim.x[j] - sim.x[i];
            const dy = sim.y[j] - sim.y[i];
            const d2 = dx * dx + dy * dy;
            if (d2 < kernel.epsilon or d2 >= h2) continue;
            const sd = kernel.goMax(@sqrt(d2), min_dist);
            const f = kernel.derivative(sd, h) * sim.nu;
            visx += dvx * f;
            visy += dvy * f;
        }
        // density-gradient repulsion
        for (nbrs) |j| {
            const dd = sim.density[j] - sim.density[i];
            if (dd <= 0) continue;
            const dx = sim.x[i] - sim.x[j];
            const dy = sim.y[i] - sim.y[j];
            const d2 = dx * dx + dy * dy;
            if (d2 < kernel.epsilon or d2 >= h2) continue;
            const dist = @sqrt(d2);
            const s = kernel.pow1_5(dd) * 0.1 * pm;
            rx += dx / dist * s;
            ry += dy / dist * s;
        }
        sim.fx[i] += px + visx + rx;
        sim.fy[i] += py + visy + ry;
    }
}

/// pairwise "gravity" between particles in neighboring cells (each pair once).
fn attractionForces(sim: *Sim) void {
    if (@abs(sim.attraction_factor) < 1e-6 or sim.interaction_radius <= 0) return;
    @memset(sim.ax, 0);
    @memset(sim.ay, 0);
    const r2 = sim.interaction_radius * sim.interaction_radius;
    const min_dist = sim.smoothing_factor * sim.interaction_radius;
    const scale = sim.attraction_factor;
    for (sim.grid.cells.items) |c| {
        const here = sim.grid.sorted.items[c.start..][0..c.len];
        var ox: i64 = -1;
        while (ox <= 1) : (ox += 1) {
            var oy: i64 = -1;
            while (oy <= 1) : (oy += 1) {
                const there = sim.grid.cell(c.x +% ox, c.y +% oy);
                for (here) |a| {
                    for (there) |b| {
                        if (a >= b) continue;
                        const dx = sim.x[b] - sim.x[a];
                        const dy = sim.y[b] - sim.y[a];
                        const d2 = dx * dx + dy * dy;
                        if (d2 >= r2 or d2 < kernel.epsilon) continue;
                        const dist = @sqrt(d2);
                        const sd = kernel.goMax(dist, min_dist);
                        if (sd < kernel.epsilon) continue;
                        const mag = scale * (sim.mass[a] * sim.mass[b]) / (sd * sd);
                        const g = mag / dist;
                        const fx = g * dx;
                        const fy = g * dy;
                        if (std.math.isNan(fx) or std.math.isNan(fy)) continue;
                        sim.ax[a] += fx;
                        sim.ay[a] += fy;
                        sim.ax[b] -= fx;
                        sim.ay[b] -= fy;
                    }
                }
            }
        }
    }
    for (sim.fx, sim.ax) |*f, a| f.* += a;
    for (sim.fy, sim.ay) |*f, a| f.* += a;
}

fn applyDrag(sim: *Sim, dt: f64) void {
    if (!sim.drag_enabled) return;
    const k = 1.0 - (sim.dampening_factor * dt * 60.0);
    for (sim.vx, sim.vy) |*vx, *vy| {
        if (vx.* * vx.* + vy.* * vy.* < 1e-6) continue;
        vx.* *= k;
        vy.* *= k;
    }
}

fn integrate(sim: *Sim, dt: f64) void {
    for (0..sim.n) |i| {
        const m = sim.mass[i];
        if (m > 0) {
            sim.vx[i] += (sim.fx[i] / m) * dt;
            sim.vy[i] += (sim.fy[i] / m) * dt;
        } else {
            sim.vx[i] += sim.fx[i] * dt;
            sim.vy[i] += sim.fy[i] * dt;
        }
        sim.x[i] += sim.vx[i] * dt;
        sim.y[i] += sim.vy[i] * dt;
        sim.handleBoundary(&sim.x[i], &sim.vx[i], sim.domain.x, sim.left_boundary);
        sim.handleBoundary(&sim.y[i], &sim.vy[i], sim.domain.y, sim.top_boundary);
    }
}

fn handleBoundary(sim: *Sim, pos: *f64, vel: *f64, limit: f64, kind: BoundaryType) void {
    const push_out = boundary_epsilon * 2.0;
    if (pos.* >= limit) {
        switch (kind) {
            .reflective => {
                pos.* = limit - push_out;
                vel.* *= -boundary_dampening;
                vel.* += sim.randomJitter(vel.*) * 0.1;
            },
            .periodic => pos.* = 0,
        }
    } else if (pos.* <= 0) {
        switch (kind) {
            .reflective => {
                pos.* = push_out;
                vel.* *= -boundary_dampening;
                vel.* += sim.randomJitter(vel.*) * 0.1;
            },
            .periodic => pos.* = limit,
        }
    }
}

fn randomJitter(sim: *Sim, v: f64) f64 {
    return (sim.jitter.next() * 2 - 1.0) * randomization_factor * @abs(v);
}

/// mean/std of pressure. the go version sums int64(p*1000) for the mean, so
/// we do the same truncation (wrapping on overflow, like go).
fn pressureStats(sim: *Sim) Stats {
    if (sim.n == 0) return .{ .mean_pressure = 0, .std_pressure = 0 };
    var total: i64 = 0;
    for (sim.pressure) |p| total +%= goInt64(p * 1000);
    const nf: f64 = @floatFromInt(sim.n);
    const mean = @as(f64, @floatFromInt(total)) / (nf * 1000.0);
    var var_sum: f64 = 0;
    for (sim.pressure) |p| var_sum += (p - mean) * (p - mean);
    return .{ .mean_pressure = mean, .std_pressure = @sqrt(var_sum / nf) };
}

/// go's int64(f) on amd64: truncate; NaN/out-of-range yields minInt.
pub fn goInt64(f: f64) i64 {
    if (!(f > -9.223372036854775808e18 and f < 9.223372036854775808e18)) return std.math.minInt(i64);
    return @intFromFloat(@trunc(f));
}

/// ApplyMouseForceToParticles, taking the click already in sim coordinates.
/// uses the grid from the last step, like the go version.
pub fn applyMouseForce(sim: *Sim, mx: f64, my: f64, force: f64, radius: f64) void {
    const r2 = radius * radius;
    const cs = sim.grid.cell_size;
    const mcx: i64 = goInt64(mx / cs);
    const mcy: i64 = goInt64(my / cs);
    const reach: i64 = goInt64(@ceil(radius / cs));
    var dx: i64 = -reach;
    while (dx <= reach) : (dx += 1) {
        var dy: i64 = -reach;
        while (dy <= reach) : (dy += 1) {
            for (sim.grid.cell(mcx +% dx, mcy +% dy)) |i| {
                const px = sim.x[i] - mx;
                const py = sim.y[i] - my;
                const d2 = px * px + py * py;
                if (d2 > r2 or d2 < 1e-6) continue;
                const d = @sqrt(d2);
                const f = (1.0 - d / radius) * force;
                sim.vx[i] += px / d * f;
                sim.vy[i] += py / d * f;
            }
        }
    }
}

test "sim runs and stays in the box" {
    var sim = try Sim.init(std.testing.allocator, 500, .{ .x = 300, .y = 200 }, .{}, 42);
    defer sim.deinit();
    for (0..200) |_| _ = try sim.step(0, 10, sim.dt);
    for (sim.x, sim.y) |x, y| {
        try std.testing.expect(x > 0 and x < 300);
        try std.testing.expect(y > 0 and y < 200);
    }
}
