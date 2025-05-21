# AGENTS.md: Project Onboarding Guide

This document provides a high-level overview of the Go SPH fluid simulation project, focusing on how to run it, its design principles, and development patterns. It's intended to help a new developer get up to speed quickly.

## Running the Simulation

1.  **Clone the repository:**
    ```console
    git clone https://github.com/zzstoatzz/fluids.git
    cd fluids
    ```
2.  **Run the simulation:**
    ```console
    go run main.go [flags]
    ```
    *   Common flags include `-n` (number of particles), `-pressure` (pressure multiplier), `-dt` (time step).
    *   Refer to `README.md` or run `go run main.go -h` for a full list of command-line flags and their default values.
    *   In-simulation controls allow for real-time adjustment of many parameters (see `README.md`).

## Project Overview & Goals

This project implements a 2D fluid simulation using the Smoothed Particle Hydrodynamics (SPH) method. The primary goals are:
*   **Real-time performance:** Capable of simulating thousands of particles at interactive frame rates.
*   **Visual appeal and interactivity:** Creating a visually engaging simulation that users can interact with (e.g., mouse explosions, parameter tuning).
*   **Exploration of SPH techniques:** Implementing and optimizing core SPH algorithms.

## Core Architectural Components

*   **SPH Engine (`simulation/` package):**
    *   `fluidsim.go`: Contains the main `FluidSim` struct and the `Step` function orchestrating the simulation loop.
    *   `forces.go`: Implements SPH forces (pressure, viscosity, repulsion) and the N-body attraction/repulsion force.
    *   `density_pressure.go`: Handles density and pressure calculations.
    *   `integration.go`: Contains the integration scheme.
    *   `neighbors.go`: Manages finding neighboring particles.
    *   `initialization.go`: Particle initialization logic.
    *   `config.go`: Defines `SimParameters` and default values, providing centralized configuration.
*   **Spatial Grid (`spatial/` package):**
    *   `grid.go`: Implements a spatial partitioning grid to accelerate neighbor searches.
    *   `kernel.go`: Contains SPH smoothing kernel functions.
*   **Main Application (`main.go`):**
    *   Handles command-line flag parsing.
    *   Initializes and runs the simulation loop.
    *   Manages SDL for windowing, rendering, and input.
    *   Provides an HTTP server for `pprof` profiling.
*   **Visualization (`viz/` package):**
    *   `render.go`: Handles all SDL rendering.
*   **Input Handling (`input/` package):**
    *   `mouse.go`: Logic for mouse interaction forces.

## Key Design Principles & Development Patterns

Throughout the development of this project, several key principles and patterns have been emphasized:

1.  **Performance and Optimization:**
    *   **Concurrency:** Physics calculations are parallelized using a custom `parallelFor` utility.
    *   **Algorithmic Efficiency:** The spatial grid is crucial for efficient neighbor finding.
    *   **Memory Management:** Efforts to reuse memory and minimize allocations in hot paths.
    *   **Profiling:** Integrated `pprof` for identifying and addressing performance bottlenecks.

2.  **Modularity and Configuration:**
    *   **Code Separation:** The simulation logic is broken down into multiple files within the `simulation` package.
    *   **Centralized Parameters:** `SimParameters` struct (`simulation/config.go`) for managing tunable parameters and defaults.

3.  **Tunability and Interactivity:**
    *   **Extensive Controls:** Many parameters are adjustable via command-line flags and real-time keyboard shortcuts.
    *   **Visual Feedback:** Debug overlay and console outputs provide insight into simulation state.

4.  **Iterative Development and Debugging:**
    *   **Bottleneck-Driven Optimization:** A cycle of profiling, identifying issues, implementing solutions, and testing.
    *   **Debugging Aids:** Use of diagnostic prints and careful state examination to resolve bugs.
    *   **Addressing Edge Cases:** Focus on numerical stability (e.g., `KERNEL_EPSILON`, `SmoothingFactor`).

5.  **Correctness and Robustness:**
    *   **Race Condition Resolution:** Identified and fixed concurrency issues.
    *   **Logical Bug Fixes:** Addressed errors in physics updates and data handling.
    *   **Modern Go Practices:** Updated code to align with current Go idioms (e.g., `rand.Seed` replacement).

## Getting Started with Development

1.  **Understand the Main Loop:** Start with `main.go` (`RunSimulation` function).
2.  **Explore `FluidSim`:** Review `simulation/fluidsim.go` (`FluidSim` struct and `Step()` method).
3.  **Review `SimParameters`:** Check `simulation/config.go`.
4.  **Dive into Specific Systems:** `simulation/forces.go`, `spatial/grid.go`, `viz/render.go`.
5.  **Use `pprof`:** Access `http://localhost:6060/debug/pprof/` during simulation.
6.  **Experiment:** Modify parameters, add temporary logging, and observe effects.

This project is a continuous work in progress, with ongoing efforts to improve performance, realism, and features. 