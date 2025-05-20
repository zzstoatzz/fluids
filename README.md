# Go Fluid Simulation

A real-time fluid simulation using Smoothed Particle Hydrodynamics (SPH) implemented in Go.

![Fluid Simulation Demo](https://github.com/zzstoatzz/fluids/raw/main/demo.gif)

## Getting Started

```console
git clone https://github.com/zzstoatzz/fluids.git
cd fluids
go run main.go
```

## Performance-Optimized Features

This implementation includes numerous optimizations for high-performance fluid simulation:

- Spatial partitioning with integer-based grid cells
- Multi-threaded physics calculations using Go's concurrency features
- Smoothing factors for more stable and realistic fluid behavior
- Efficient memory usage with pre-allocated buffers
- Batched rendering for smoother display
- Smart particle neighbor searching

## Command-Line Flags

### Basic Parameters
- `n`: Number of particles (default: 1000)
- `radius`: Radius of particles (default: 2.4, range: 1.0-5.0)
- `fps`: Target frame rate (default: 120, limited by hardware)
- `g`: Gravity strength (default: 0 = disabled)
- `dt`: Physics time step (default: 0.0008, smaller = more accurate but slower)
- `boom`: Magnitude of mouse explosion force (default: 500)
- `pressure`: Pressure multiplier affecting fluid stiffness (default: 20000)
- `nu`: Viscosity of the fluid (default: 0.5, higher = more viscous)

### Performance Parameters
- `smooth`: Smoothing factor for forces (default: 0.15, smaller = more accurate but can be unstable)
- `drag`: Drag/dampening factor (default: 0.1, higher = more damping)
- `workers`: Number of worker threads for physics calculations (default: auto-detected)

### Example Commands

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

## In-Simulation Controls

### Basic Controls
- **Left Click**: Create an explosion force pushing particles away from cursor
- **Press G**: Toggle gravity on/off
- **Press D**: Toggle debug visualization mode (shows grid, connections, and velocities)
- **Press B**: Bigger explosion (5x normal force)
- **Space**: Pause/resume simulation
- **Press R**: Reset simulation

### Interactive Parameter Adjustment
- **Press A**: Decrease pressure stiffness (softer fluid, less responsive to compression)
- **Press S**: Increase pressure stiffness (stiffer fluid, more responsive to compression)
- **Press Z**: Decrease drag/dampening (more bouncy, less stable)
- **Press X**: Increase drag/dampening (more fluid-like, more stable)
- **Press C**: Decrease smoothing factor (more accurate physics but potentially unstable)
- **Press V**: Increase smoothing factor (more stable but less accurate physics)
- **Press Q**: Decrease N-body attraction (negative values = particles repel like electrostatic force)
- **Press W**: Increase N-body attraction (positive values = particles attract like gravitational force)
- **Press E**: Decrease interaction radius (particles interact in smaller range)
- **Press T**: Increase interaction radius (particles interact in larger range)
- **Press [**: Decrease mouse explosion force
- **Press ]**: Increase mouse explosion force

The current parameter values are displayed in the console when adjusted.

## Physics Implementation

The simulation uses Smoothed Particle Hydrodynamics (SPH), a mesh-free Lagrangian method for simulating fluid flows. Key components include:

- Pressure forces calculated from particle density
- Viscosity forces for fluid resistance
- Repulsion forces for more natural clustering behavior
- Spatial partitioning for efficient neighbor search
- Semi-implicit integration scheme for stability