package simulation

import (
	"fluids/core"
	"fluids/spatial"
	"math"
	"math/rand"
	"runtime"
)

type Domain struct {
	X, Y float64
}

// FluidSim represents a fluid simulation using Smoothed Particle Hydrodynamics
type FluidSim struct {
	Particles         []core.Particle
	N                 int
	Dt                float64
	Rho0, Nu          float64
	Domain            Domain
	Grid              *spatial.Grid
	LeftBoundary      spatial.BoundaryType
	TopBoundary       spatial.BoundaryType
	SmoothingFactor   float64
	DampeningFactor   float64
	DragEnabled       bool
	AttractionFactor  float64
	InteractionRadius float64
	tempVectors       []core.Vector // Pre-allocated vectors for reuse
	localAtomics      []int64       // Atomic counter for thread-local sums (used by CalculatePressureStats)
	statsBuffers      []float64     // Buffers for statistical calculations (used by CalculatePressureStats)
}

// NewFluidSim creates a new fluid simulation using parameters from SimParameters.
func NewFluidSim(n int, domain Domain, params SimParameters, rng *rand.Rand) *FluidSim {
	particles := make([]core.Particle, n)

	for i := 0; i < n; i++ {
		particles[i].X, particles[i].Y, particles[i].Vx, particles[i].Vy = RandomStillInitialCondition(i, domain, rng)
		// Rho0 is set from params, so particle density can be initialized to it.
		particles[i].Density = params.Rho0
		particles[i].NeighborIndices = make([]int, 0, 32)
		baseRadius := 2.0 // This is particle visual radius, not interaction radius
		particles[i].Radius = baseRadius * (0.8 + 0.4*rng.Float64())
		particles[i].Mass = math.Pi * particles[i].Radius * particles[i].Radius
	}

	grid := spatial.NewGrid(params.InteractionRadius, int(domain.X), int(domain.Y))

	sim := &FluidSim{
		Particles:         particles,
		N:                 n,
		Dt:                params.Dt,
		Domain:            domain,
		Rho0:              params.Rho0,
		Nu:                params.Nu,
		Grid:              grid,
		SmoothingFactor:   params.SmoothingFactor,
		DampeningFactor:   params.DampeningFactor,
		DragEnabled:       params.DragEnabled,
		AttractionFactor:  params.AttractionFactor,
		InteractionRadius: params.InteractionRadius,
		LeftBoundary:      spatial.Reflective,
		TopBoundary:       spatial.Reflective,
		tempVectors:       make([]core.Vector, n),
		localAtomics:      make([]int64, runtime.NumCPU()),
		statsBuffers:      make([]float64, runtime.NumCPU()*2),
	}
	return sim
}

// Step advances the simulation by one time step
func (sim *FluidSim) Step(gravity, pressureMultiplier, dt float64) (float64, float64) {
	// 0. Initialize forces for all particles
	parallelFor(0, len(sim.Particles), func(i int) {
		sim.Particles[i].Force = core.Vector{X: 0, Y: 0}
	})

	// 1. Predictor step
	sim.PredictPositions(dt)

	// 2. Update spatial grid (Grid itself is in spatial package)
	sim.Grid.Update(sim.Particles)

	// 3. Find neighbors
	sim.FindNeighbors()

	// 4. Calculate densities
	sim.UpdateDensities()

	// 5. Calculate pressures
	sim.UpdatePressure(pressureMultiplier)

	// --- Force Calculation Phase ---
	sim.ApplyGravityForces(gravity)
	sim.CalculateStandardForces(pressureMultiplier)
	sim.CalculateAttractionForces()
	sim.ApplyDrag(dt)
	// --- End Force Calculation Phase ---

	// Integrator step
	sim.Integrate(dt)

	// Calculate statistics for visualization/debugging
	meanPressure, stdPressure := sim.CalculatePressureStats()

	return meanPressure, stdPressure
}
