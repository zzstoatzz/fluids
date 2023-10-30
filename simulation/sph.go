package simulation

import (
	"fluids/core"
	"fluids/spatial"
	"math"
	"math/rand"
	"sync"
)

const K = 1000.0
const EPSILON = 0.01
const DAMPENING_FACTOR = 0.3

type InitialConditionFunc func(i, n int) (x, y, vx, vy float64)

type Vector struct {
	X, Y float64
}

type Domain struct {
	X, Y float64
}

type BoundaryType int

const (
	Reflective BoundaryType = iota
	Periodic
)

func handleBoundary(position *float64, velocity *float64, limit float64, boundaryType BoundaryType) {
	if *position >= limit {
		if boundaryType == Reflective {
			*position = limit - EPSILON
			*velocity *= -DAMPENING_FACTOR
		}
	} else if *position <= 0 {
		if boundaryType == Reflective {
			*position = EPSILON
			*velocity *= -DAMPENING_FACTOR
		}
	}
}

type FluidSim struct {
	Particles      []core.Particle
	N              int     // Number of particles
	Dt             float64 // Time step
	Rho0, Nu       float64 // Reference density and viscosity
	Domain         Domain  // Domain of the simulation
	Grid           *spatial.Grid
	LeftBoundary   BoundaryType
	RightBoundary  BoundaryType
	TopBoundary    BoundaryType
	BottomBoundary BoundaryType
}

func RandomInitialCondition(i int, domain Domain) (float64, float64, float64, float64) {
	x := rand.Float64() * domain.X
	y := rand.Float64() * domain.Y
	vx := (rand.Float64() * 2.0) - 1.0
	vy := (rand.Float64() * 2.0) - 1.0
	return x, y, vx, vy
}

func NewFluidSim(n int, domain Domain, dt, rho0, nu float64) *FluidSim {
	particles := make([]core.Particle, n)
	for i := 0; i < n; i++ {
		particles[i].X, particles[i].Y, particles[i].Vx, particles[i].Vy = RandomInitialCondition(i, domain)
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

func (sim *FluidSim) CalculatePressureForce(p *core.Particle, pressureMultiplier float64) *core.Vector {
	var force core.Vector

	for _, neighbor := range p.Neighbors {
		dx := neighbor.X - p.X
		dy := neighbor.Y - p.Y
		r2 := dx*dx + dy*dy + EPSILON

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

func (sim *FluidSim) FindNeighbors() {
	MaxNeighbors := sim.N / 10

	for i := range sim.Particles {
		// Preallocate slice for neighbors
		sim.Particles[i].Neighbors = make([]core.Particle, 0, MaxNeighbors)

		// Get grid cell of the particle
		cellX, cellY := int(sim.Particles[i].X/sim.Grid.CellSize), int(sim.Particles[i].Y/sim.Grid.CellSize)

		// Loop over neighboring cells
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				neighborCellX, neighborCellY := cellX+dx, cellY+dy
				if neighborCellX >= 0 && neighborCellX < sim.Grid.NumCellsX && neighborCellY >= 0 && neighborCellY < sim.Grid.NumCellsY {
					for _, neighborIdx := range sim.Grid.Cells[neighborCellX][neighborCellY].Particles {
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
func (sim *FluidSim) UpdatePressure() {
	parallelFor(0, len(sim.Particles), func(i int) {
		sim.Particles[i].Pressure = K * (sim.Particles[i].Density - sim.Rho0)
	})
}

func (sim *FluidSim) UpdateForces(gravity, pressureMultiplier float64) {
	for i := range sim.Particles {
		// Step 1: Reset forces and apply gravitational force
		sim.Particles[i].Force = core.Vector{X: 0, Y: -sim.Particles[i].Density * gravity}

		// Step 2: Calculate and apply pressure and viscosity forces
		p1 := &sim.Particles[i]
		pressureForce := sim.CalculatePressureForce(p1, pressureMultiplier)
		viscosityForce := sim.CalculateViscosityForce(p1)

		sim.Particles[i].Force.Add(pressureForce)
		sim.Particles[i].Force.Add(viscosityForce)
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
		handleBoundary(&p.X, &p.Vx, sim.Domain.X, sim.LeftBoundary)
		handleBoundary(&p.Y, &p.Vy, sim.Domain.Y, sim.TopBoundary)
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

func (sim *FluidSim) Step(gravity, pressureMultiplier float64) (float64, float64) {
	sim.Grid.Update(sim.Particles)
	sim.FindNeighbors()
	sim.UpdateDensities()
	sim.UpdatePressure()
	sim.UpdateForces(gravity, pressureMultiplier)
	sim.Integrate()

	meanPressure, stdPressure := sim.CalculatePressureStats()
	return meanPressure, stdPressure
}
