# AGENTS.md

2d sph fluid sim. zig is the implementation; `go/` is the original go version, kept as the parity reference.

## commands

```console
zig build test                         # unit tests
zig build bench                        # native perf
zig build wasm                         # web/fluids.wasm
./web/bundle.sh                        # web/dist/index.html with the wasm inlined
cd go && go run ./cmd/oracle -cases 300 -seed 1 -portable-pow > /tmp/parity.json
zig build parity -Doptimize=ReleaseSafe -- /tmp/parity.json
```

zig 0.17. no dependencies.

## rules of the road

- behavior changes to `src/` must keep `zig build parity` green, or come with the matching change in `go/` and a note on why. physics bug fixes are fine, they just need to be deliberate.
- `src/Sim.zig` runs single-threaded in index order on purpose; it's what makes wasm and native produce identical bits.
- float constants in `src/kernel.zig` are typed `f64`. comptime_float math rounds differently from go's runtime f64 math and breaks bit-exactness.
- don't swap `pow1_5`, `goExp` or `goLog` for `@exp`/`@log`/`std.math.pow`; see the comments there.
- the go side has two package vars (`spatial.JitterFloat64`, `simulation.Pow`) that exist only so the oracle can record and pin randomness. default behavior is unchanged.
