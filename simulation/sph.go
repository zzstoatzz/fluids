package simulation

import (
	"fluids/core"
	"fluids/spatial"
	"fmt"
	"math"
	"math/rand"
	"sync"
)

type InitialConditionFunc func(i, n int) (x, y, vx, vy float64)

type Domain struct {
	X, Y float64
}

func RandomStillInitialCondition(i int, domain Domain) (float64, float64, float64, float64) {
	x := rand.Float64() * domain.X
	y := rand.Float64() * domain.Y
	// vx := (rand.Float64() * 2.0) - 1.0
	// vy := (rand.Float64() * 2.0) - 1.0
	return x, y, 0, 0
}

func RandomMotionInitialCondition(i int, domain Domain) (float64, float64, float64, float64) {
	x := rand.Float64() * domain.X
	y := rand.Float64() * domain.Y
	vx := (rand.Float64() * 2.0) - 1.0
	vy := (rand.Float64() * 2.0) - 1.0
	return x, y, vx, vy
}

type FluidSim struct {
	Particles    []core.Particle
	N            int     // Number of particles
	Dt           float64 // Time step
	Rho0, Nu     float64 // Reference density and viscosity
	Domain       Domain  // Domain of the simulation
	Grid         *spatial.Grid
	LeftBoundary spatial.BoundaryType
	TopBoundary  spatial.BoundaryType
}

func NewFluidSim(n int, domain Domain, dt, rho0, nu float64) *FluidSim {
	particles := make([]core.Particle, n)
	for i := 0; i < n; i++ {
		particles[i].X, particles[i].Y, particles[i].Vx, particles[i].Vy = RandomStillInitialCondition(i, domain)
		particles[i].Density = rho0
	}

	grid := spatial.NewGrid(spatial.SMOOTHING_RADIUS, int(domain.X), int(domain.Y))
	return &FluidSim{
		Particles: particles,
		N:         n,
		Dt:        dt,
		Domain:    domain,
		Rho0:      rho0,
		Nu:        nu,
		Grid:      grid,
	}
}

func (sim *FluidSim) PredictPositions(dt float64) {
	for i := range sim.Particles {
		p := &sim.Particles[i]
		p.X += p.Vx * dt
		p.Y += p.Vy * dt
	}
}

func (sim *FluidSim) FindNeighbors() {
	for i := range sim.Particles {
		sim.Particles[i].Neighbors = []core.Particle{}
		cellX, cellY := int(sim.Particles[i].X/sim.Grid.CellSize), int(sim.Particles[i].Y/sim.Grid.CellSize)

		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				neighborCellX, neighborCellY := cellX+dx, cellY+dy
				key := fmt.Sprintf("%d-%d", neighborCellX, neighborCellY) // make cell key

				if neighborIndices, found := sim.Grid.CellMap[key]; found {
					for _, neighborIdx := range neighborIndices {
						dx := sim.Particles[i].X - sim.Particles[neighborIdx].X
						dy := sim.Particles[i].Y - sim.Particles[neighborIdx].Y
						distanceSquared := dx*dx + dy*dy

						if distanceSquared < spatial.SMOOTHING_RADIUS*spatial.SMOOTHING_RADIUS {
							sim.Particles[i].Neighbors = append(sim.Particles[i].Neighbors, sim.Particles[neighborIdx])
						}
					}
				}
			}
		}
	}
}

func (sim *FluidSim) UpdateDensities() {
	parallelFor(0, len(sim.Particles), func(i int) {
		sim.Particles[i].Density = spatial.CalculateDensity(sim.Particles[i])
	})
}

// update pressure based on density
func (sim *FluidSim) UpdatePressure(pressureMultiplier float64) {
	parallelFor(0, len(sim.Particles), func(i int) {
		sim.Particles[i].Pressure = pressureMultiplier * (sim.Particles[i].Density - sim.Rho0)
	})
}

func (sim *FluidSim) CalculatePressureForce(p *core.Particle, pressureMultiplier float64) *core.Vector {
	var force core.Vector

	for _, neighbor := range p.Neighbors {
		dx := neighbor.X - p.X
		dy := neighbor.Y - p.Y
		r2 := dx*dx + dy*dy + spatial.EPSILON

		gradW := spatial.SmoothingKernelGradient(neighbor)

		forceContribution := &gradW
		forceContribution.MultiplyByScalar((p.Pressure + neighbor.Pressure) / (2 * r2))
		forceContribution.MultiplyByScalar(-1 * pressureMultiplier)
		force.Add(forceContribution)
		neighbor.Force.Subtract(forceContribution) // Newton's 3rd Law
	}
	return &force
}

func (sim *FluidSim) CalculateViscosityForce(p *core.Particle) *core.Vector {
	var force core.Vector

	for _, neighbor := range p.Neighbors {
		dx := neighbor.X - p.X
		dy := neighbor.Y - p.Y
		velocityDiff := (neighbor.Vx - p.Vx) + (neighbor.Vy - p.Vy)
		lapW := spatial.SmoothingKernelLaplacian(*p)

		forceContribution := &core.Vector{X: dx, Y: dy}
		forceContribution.MultiplyByScalar(lapW * sim.Nu * velocityDiff)
		forceContribution.MultiplyByScalar(-1)
		force.Add(forceContribution)
		neighbor.Force.Subtract(forceContribution) // Newton's 3rd Law
	}
	return &force
}

func (sim *FluidSim) CalculateRepulsionForce(p *core.Particle, pressureMultiplier float64) *core.Vector {
	repulsionForce := &core.Vector{X: 0, Y: 0}
	for _, neighbor := range p.Neighbors {
		if neighbor.Density < p.Density { // Move away from higher density
			dx := p.X - neighbor.X
			dy := p.Y - neighbor.Y
			distance := math.Sqrt(dx*dx + dy*dy)
			if distance > 0 {
				repulsionForce.X += (dx / distance) * pressureMultiplier
				repulsionForce.Y += (dy / distance) * pressureMultiplier
			}
		}
	}
	return repulsionForce
}

func (sim *FluidSim) UpdateForces(gravity, pressureMultiplier float64) {
	for i := range sim.Particles {
		// Step 1: Reset forces and apply gravitational force
		sim.Particles[i].Force = core.Vector{X: 0, Y: -sim.Particles[i].Density * gravity}

		// Step 2: Calculate and apply pressure and viscosity forces
		p1 := &sim.Particles[i]
		pressureForce := sim.CalculatePressureForce(p1, pressureMultiplier)
		viscosityForce := sim.CalculateViscosityForce(p1)
		repulsionForce := sim.CalculateRepulsionForce(p1, pressureMultiplier)

		// Step 3: Aggregate all forces
		sim.Particles[i].Force.Add(pressureForce)
		sim.Particles[i].Force.Add(viscosityForce)
		sim.Particles[i].Force.Add(repulsionForce)
	}
}

func (sim *FluidSim) Integrate() {
	parallelFor(0, len(sim.Particles), func(i int) {
		p := &sim.Particles[i]

		// Update velocities
		p.Vx += p.Force.X * sim.Dt
		p.Vy += p.Force.Y * sim.Dt

		// Update positions
		p.X += p.Vx * sim.Dt
		p.Y += p.Vy * sim.Dt

		// Handle boundaries
		spatial.HandleBoundary(&p.X, &p.Vx, sim.Domain.X, sim.LeftBoundary)
		spatial.HandleBoundary(&p.Y, &p.Vy, sim.Domain.Y, sim.TopBoundary)
	})
}

func (sim *FluidSim) CalculatePressureStats() (float64, float64) {
	var meanPressure, stdPressure float64
	var meanSum, stdSum float64
	n := len(sim.Particles)
	meanMux := &sync.Mutex{}
	stdMux := &sync.Mutex{}

	parallelFor(0, n, func(i int) {
		meanMux.Lock()
		meanSum += sim.Particles[i].Pressure
		meanMux.Unlock()
	})

	meanPressure = meanSum / float64(n)

	parallelFor(0, n, func(i int) {
		d := sim.Particles[i].Pressure - meanPressure
		stdMux.Lock()
		stdSum += d * d
		stdMux.Unlock()
	})

	stdPressure = math.Sqrt(stdSum / float64(n))

	return meanPressure, stdPressure
}

// ####################################################################################################

func (sim *FluidSim) Step(gravity, pressureMultiplier, dt float64) (float64, float64) {
	sim.PredictPositions(dt)
	sim.Grid.Update(sim.Particles)
	sim.FindNeighbors()
	sim.UpdateDensities()
	sim.UpdatePressure(pressureMultiplier)
	sim.UpdateForces(gravity, pressureMultiplier)
	sim.Integrate()

	meanPressure, stdPressure := sim.CalculatePressureStats()
	return meanPressure, stdPressure
}
