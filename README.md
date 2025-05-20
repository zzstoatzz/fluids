# fluid simulation

a real-time fluid simulation using Smoothed Particle Hydrodynamics (SPH) implemented in Go.

## install

```console
git clone https://github.com/zzstoatzz/fluids.git
cd fluids
go run main.go
```

## features

This implementation includes numerous optimizations and features for high-performance fluid simulation:

- Spatial partitioning with integer-based grid cells for efficient neighbor searching.
- Multi-threaded physics calculations leveraging Go's concurrency.
- Tunable smoothing factors for stable and realistic fluid behavior.
- Centralized `SimParameters` struct for streamlined configuration and default management.
- Modular code structure (simulation logic refactored into multiple focused files).
- Addressed `rand.Seed` deprecation using localized random number generators.
- Batched rendering for smoother display.
- Key bug fixes including race condition in neighbor finding and incorrect particle coordinate updates.


## command-line flags

### basic parameters
- `n`: Number of particles (default: 1000)
- `radius`: Visual radius of particles (default: 2.4, range: 1.0-5.0)
- `fps`: Target frame rate (default: 120, limited by hardware)
- `g`: Gravity strength (default: 0 = disabled)
- `dt`: Physics time step (default: 0.0008, smaller = more accurate but slower)
- `boom`: Magnitude of mouse explosion force (default: 5000.0)
- `mouseRadius`: Radius of mouse explosion force (default: 100.0)
- `pressure`: Pressure multiplier affecting fluid stiffness (default: 1.0)
- `nu`: Viscosity of the fluid (default: 0.5, higher = more viscous)

### performance and behavior parameters
- `smooth`: Smoothing factor for forces (default: 0.05, smaller = more accurate but can be unstable)
- `drag`: Drag/dampening factor (default: 0.3, higher = more damping)
- `workers`: Number of worker threads for physics calculations (default: auto-detected based on CPU cores)

### example commands

For a large, fast-moving, water-like simulation (adjust parameters as per new defaults):
```console
go run main.go -n 2000 -radius 2 -pressure 100 -dt 0.0008 -boom 1000 -drag 0.05 -mouseRadius 75
```

For a smaller, more viscous, honey-like fluid (adjust parameters as per new defaults):
```console
go run main.go -n 500 -radius 3 -pressure 50 -nu 3.0 -dt 0.001 -boom 500 -drag 0.2 -mouseRadius 50
```

For maximum performance on a high-end machine:
```console
go run main.go -n 5000 -radius 1.5 -workers 16
```
(Note: Optimal parameters depend heavily on the interaction between `pressure`, `smooth`, `dt`, and N-body `AttractionFactor` if used).

## in-simulation controls

### basic controls
- **left click**: create an explosion force pushing particles away from cursor
- **press g**: toggle gravity on/off
- **press d**: toggle debug visualization mode (shows grid, connections, and velocities)
- **press b**: bigger explosion (5x normal force)
- **space**: pause/resume simulation
- **press r**: reset simulation

### interactive parameter adjustment
- **press a**: decrease pressure stiffness
- **press s**: increase pressure stiffness
- **press z**: decrease drag/dampening
- **press x**: increase drag/dampening
- **press c**: decrease smoothing factor
- **press v**: increase smoothing factor
- **press q**: decrease n-body attraction (more negative = stronger repulsion)
- **press w**: increase n-body attraction (more positive = stronger attraction)
- **press e**: decrease interaction radius
- **press t**: increase interaction radius
- **press [**: decrease mouse explosion force
- **press ]**: increase mouse explosion force
- **press ,**: decrease mouse explosion radius
- **press .**: increase mouse explosion radius

The current parameter values are displayed in the console when adjusted (and in the debug overlay if active).

## physics implementation

The simulation uses Smoothed Particle Hydrodynamics (SPH), a mesh-free Lagrangian method for simulating fluid flows. Key components include:

- Pressure forces calculated from particle density (Tait's equation of state).
- Viscosity forces for fluid resistance.
- Repulsion forces for more natural clustering behavior (part of standard SPH forces).
- Optional N-body attraction/repulsion force, independently tunable.
- Spatial partitioning for efficient neighbor search.
- Semi-implicit integration scheme for stability.


<details>

### recent improvements
- **Parameter Centralization**: Simulation parameters are now managed in a dedicated `SimParameters` struct (`simulation/config.go`), making defaults and configurations easier to handle.
- **Code Modularity**: The core simulation logic in `simulation/sph.go` has been refactored into several more focused files (`fluidsim.go`, `forces.go`, `integration.go`, etc.) for better maintainability.
- **Concurrency Safety**: Resolved a critical race condition in the neighbor finding algorithm.
- **Correctness Fixes**: Addressed bugs related to particle data updates in the spatial grid and ensured simulation parameters (like interaction radius) are correctly used by kernel functions.
- **Modernized Randomness**: Updated random number generation to align with Go 1.20+ best practices, removing deprecated `rand.Seed` calls.

</details>
