# fluid simulation

a real-time fluid simulation using Smoothed Particle Hydrodynamics (SPH) implemented in Go.

## install

```console
git clone https://github.com/zzstoatzz/fluids.git
cd fluids
go run main.go
```

## features

This implementation includes numerous optimizations for high-performance fluid simulation:

- Spatial partitioning with integer-based grid cells
- Multi-threaded physics calculations using Go's concurrency features
- Smoothing factors for more stable and realistic fluid behavior
- Efficient memory usage with pre-allocated buffers
- Batched rendering for smoother display
- Smart particle neighbor searching

## command-line flags

### basic parameters
- `n`: Number of particles (default: 1000)
- `radius`: Radius of particles (default: 2.4, range: 1.0-5.0)
- `fps`: Target frame rate (default: 120, limited by hardware)
- `g`: Gravity strength (default: 0 = disabled)
- `dt`: Physics time step (default: 0.0008, smaller = more accurate but slower)
- `boom`: Magnitude of mouse explosion force (default: 500)
- `pressure`: Pressure multiplier affecting fluid stiffness (default: 20000)
- `nu`: Viscosity of the fluid (default: 0.5, higher = more viscous)

### performance parameters
- `smooth`: Smoothing factor for forces (default: 0.15, smaller = more accurate but can be unstable)
- `drag`: Drag/dampening factor (default: 0.1, higher = more damping)
- `workers`: Number of worker threads for physics calculations (default: auto-detected)

### example commands

For a large, fast-moving, water-like simulation:
```console
go run main.go -n 2000 -radius 2 -pressure 15000 -fps 120 -dt 0.0008 -boom 500 -drag 0.05
```

For a smaller, more viscous, honey-like fluid:
```console
go run main.go -n 500 -radius 3 -pressure 25000 -fps 60 -dt 0.001 -boom 200 -nu 3.0 -drag 0.15
```

For maximum performance on a high-end machine:
```console
go run main.go -n 5000 -radius 1.5 -workers 16
```

## in-simulation controls

### basic controls
- **left click**: create an explosion force pushing particles away from cursor
- **press g**: toggle gravity on/off
- **press d**: toggle debug visualization mode (shows grid, connections, and velocities)
- **press b**: bigger explosion (5x normal force)
- **space**: pause/resume simulation
- **press r**: reset simulation

### interactive parameter adjustment
- **press a**: decrease pressure stiffness (softer fluid, less responsive to compression)
- **press s**: increase pressure stiffness (stiffer fluid, more responsive to compression)
- **press z**: decrease drag/dampening (more bouncy, less stable)
- **press x**: increase drag/dampening (more fluid-like, more stable)
- **press c**: decrease smoothing factor (more accurate physics but potentially unstable)
- **press v**: increase smoothing factor (more stable but less accurate physics)
- **press q**: decrease n-body attraction (negative values = particles repel like electrostatic force)
- **press w**: increase n-body attraction (positive values = particles attract like gravitational force)
- **press e**: decrease interaction radius (particles interact in smaller range)
- **press t**: increase interaction radius (particles interact in larger range)
- **press [**: decrease mouse explosion force
- **press ]**: increase mouse explosion force

The current parameter values are displayed in the console when adjusted.

## physics implementation

The simulation uses Smoothed Particle Hydrodynamics (SPH), a mesh-free Lagrangian method for simulating fluid flows. Key components include:

- Pressure forces calculated from particle density
- Viscosity forces for fluid resistance
- Repulsion forces for more natural clustering behavior
- Spatial partitioning for efficient neighbor search
- Semi-implicit integration scheme for stability