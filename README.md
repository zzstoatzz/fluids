# fluids

a real-time 2d fluid sim using smoothed particle hydrodynamics (sph), written in zig and compiled to a ~24kb wasm module.

it's a port of the original go + sdl version, which still lives in [`go/`](go/) as the reference implementation. the zig sim matches it step for step (see [parity](#parity)).

## run it

needs zig 0.17.

```console
zig build test            # unit tests
zig build bench           # native ms/step for 1000 particles
./web/bundle.sh           # wasm + inline it into web/dist/index.html, open that in a browser
```

in the browser: tap or click for an explosion, two-finger tap for a 5x one, `space` pauses, `g` toggles gravity, `r` resets. the tune drawer has sliders for the parameters the go version bound to keys.

the go version still runs the same way it always did:

```console
cd go && go run . -n 1000
```

## layout

| path | what |
|---|---|
| `src/Sim.zig` | the sph step: predict, grid, neighbors, density, pressure, forces, n-body, drag, integrate |
| `src/Grid.zig` | uniform spatial hash, counting-sorted so each cell is one contiguous slice |
| `src/kernel.zig` | smoothing kernels, plus go's portable `exp`/`log` for `Pow(x, 1.5)` |
| `src/color.zig` | pressure → color palette, same as the go renderer |
| `src/wasm.zig` | wasm exports the page talks to |
| `src/parity.zig` | replays go oracle output and checks it |
| `web/` | the page |
| `go/` | the original go implementation + `cmd/oracle` |

## parity

`go/cmd/oracle` generates random sim states (uniform, clustered, wall-hugging; random params, gravity, pressure, mouse pokes, a runtime-changed interaction radius), runs them through the go sim, and records the boundary-jitter random draws so zig can replay them. `zig build parity` replays every case and checks:

1. **bit-exact**: with the n-body term off and the oracle run with `-portable-pow`, every step's positions, velocities, forces, density, pressure and mean pressure are identical to go, bit for bit
2. **within 1e-9 relative** for everything else

```console
cd go && go run ./cmd/oracle -cases 1000 -seed 1 -portable-pow > /tmp/parity.json && cd ..
zig build parity -Doptimize=ReleaseSafe -- /tmp/parity.json
```

why not bit-exact everywhere: the go sim sums the n-body forces in map-iteration order, and stock `math.Pow` goes through an amd64 asm `exp` whose bits depend on whether the cpu has fma. so go doesn't agree with itself past ~1e-12 either. the zig port uses go's portable `exp`/`log`, which makes it deterministic on every target, wasm included.

## physics

- density from a custom quadratic kernel, pressure from `k * (rho - rho0)`
- pressure, viscosity and density-gradient repulsion forces between neighbors
- optional n-body attraction/repulsion between particles in neighboring cells
- velocity drag, reflective walls with a little random jitter, semi-implicit euler

known quirks inherited from the go version (kept for parity, not fixed yet): see the PR that introduced the port.
