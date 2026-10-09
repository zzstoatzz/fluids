//! sph smoothing kernels, ported 1:1 from go/spatial/kernel.go.
//!
//! the go code uses `math.Pow(h, 4)` and `math.Pow(x, 1.5)`. go's Pow tracks the
//! exponent separately and multiplies mantissas; for integer powers that rounds
//! identically to `(h*h)*(h*h)`, so we spell it out to stay bit-exact.

const std = @import("std");

pub const epsilon: f64 = 1e-6;

// typed f64 on purpose: comptime_float math would round differently from go
const pi: f64 = std.math.pi;
const pi2: f64 = pi * pi;
/// 12 / (pi^4 * pi)
pub const derivative_scale: f64 = 12.0 / ((pi2 * pi2) * pi);

pub inline fn pow4(h: f64) f64 {
    const h2 = h * h;
    return h2 * h2;
}

/// go's math.Pow(x, 1.5): exp(0.5*log(x)) times the frexp mantissa of x, then
/// ldexp. go's own exp is asm on amd64/arm64 and its bits depend on whether the
/// cpu has fma, so go doesn't agree with itself across machines here. we use
/// go's portable exp/log (below) so every zig target, wasm included, gets the
/// same bits; vs go on amd64 this term differs by an ulp in ~15% of calls.
pub fn pow1_5(x: f64) f64 {
    if (x == 0) return 0;
    if (std.math.isInf(x) or std.math.isNan(x)) return x;
    const fr = std.math.frexp(x);
    return std.math.ldexp(goExp(0.5 * goLog(x)) * fr.significand, fr.exponent);
}

const ln2_hi = 6.93147180369123816490e-01;
const ln2_lo = 1.90821492927058770002e-10;

/// port of go's portable math.exp (from FreeBSD's e_exp.c)
pub fn goExp(x: f64) f64 {
    const log2e = 1.44269504088896338700e+00;
    const overflow = 7.09782712893383973096e+02;
    const underflow = -7.45133219101941108420e+02;
    const near_zero = 1.0 / @as(f64, 1 << 28);
    if (std.math.isNan(x) or std.math.isPositiveInf(x)) return x;
    if (std.math.isNegativeInf(x)) return 0;
    if (x > overflow) return std.math.inf(f64);
    if (x < underflow) return 0;
    if (-near_zero < x and x < near_zero) return 1 + x;
    var k: i32 = 0;
    if (x < 0) k = @intFromFloat(@trunc(log2e * x - 0.5)) else if (x > 0) k = @intFromFloat(@trunc(log2e * x + 0.5));
    const kf: f64 = @floatFromInt(k);
    const hi = x - kf * ln2_hi;
    const lo = kf * ln2_lo;
    const P1 = 1.66666666666666657415e-01;
    const P2 = -2.77777777770155933842e-03;
    const P3 = 6.61375632143793436117e-05;
    const P4 = -1.65339022054652515390e-06;
    const P5 = 4.13813679705723846039e-08;
    const r = hi - lo;
    const t = r * r;
    const c = r - t * (P1 + t * (P2 + t * (P3 + t * (P4 + t * P5))));
    const y = 1 - ((lo - (r * c) / (2 - c)) - hi);
    return std.math.ldexp(y, k);
}

/// port of go's portable math.log (from FreeBSD's e_log.c)
pub fn goLog(x: f64) f64 {
    const L1 = 6.666666666666735130e-01;
    const L2 = 3.999999999940941908e-01;
    const L3 = 2.857142874366239149e-01;
    const L4 = 2.222219843214978396e-01;
    const L5 = 1.818357216161805012e-01;
    const L6 = 1.531383769920937332e-01;
    const L7 = 1.479819860511658591e-01;
    if (std.math.isNan(x) or std.math.isPositiveInf(x)) return x;
    if (x < 0) return std.math.nan(f64);
    if (x == 0) return -std.math.inf(f64);
    const fr = std.math.frexp(x);
    var f1 = fr.significand;
    var ki = fr.exponent;
    if (f1 < std.math.sqrt2 / 2.0) {
        f1 *= 2;
        ki -= 1;
    }
    const f = f1 - 1;
    const k: f64 = @floatFromInt(ki);
    const s = f / (2 + f);
    const s2 = s * s;
    const s4 = s2 * s2;
    const t1 = s2 * (L1 + s4 * (L3 + s4 * (L5 + s4 * L7)));
    const t2 = s4 * (L2 + s4 * (L4 + s4 * L6));
    const R = t1 + t2;
    const hfsq = 0.5 * f * f;
    return k * ln2_hi - ((hfsq - (s * (hfsq + R) + k * ln2_lo)) - f);
}

/// note: the "volume" is (pi + h^4) / 6, not pi*h^4/6. kept as-is for parity.
pub inline fn smoothing(distance: f64, h: f64) f64 {
    if (distance >= h) return 0;
    const volume = (pi + pow4(h)) / 6.0;
    if (volume == 0) return 0;
    return (h - distance) * (h - distance) / volume;
}

pub inline fn derivative(distance: f64, h: f64) f64 {
    if (distance >= h) return 0;
    return (distance - h) * derivative_scale;
}

/// go's math.Max: NaN-propagating, otherwise plain max (we never see -0 here).
pub inline fn goMax(a: f64, b: f64) f64 {
    if (std.math.isNan(a) or std.math.isNan(b)) return std.math.nan(f64);
    return if (a > b) a else b;
}

test "pow1_5 tracks x^1.5" {
    const xs = [_]f64{ 1e-9, 0.3, 1, 2, 17.25, 1234.5, 9.9e7 };
    for (xs) |x| try std.testing.expectApproxEqRel(@sqrt(x) * x, pow1_5(x), 1e-14);
}

test "kernel is zero outside h" {
    try std.testing.expectEqual(@as(f64, 0), smoothing(10, 10));
    try std.testing.expectEqual(@as(f64, 0), derivative(11, 10));
    try std.testing.expect(smoothing(0, 10) > 0);
    try std.testing.expect(derivative(0, 10) < 0);
}
