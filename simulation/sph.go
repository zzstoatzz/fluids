package simulation

import (
	"fluids/core"
	"fluids/spatial"
	"math/rand"
)

const K = 1000.0
const DAMPENING_FACTOR = 0.5
const NEIGHBOR_RADIUS = 10.0
const Gravity = -9.81 // Acceleration due to gravity in m/s^2
const PressureMultiplier = 1.0

type InitialConditionFunc func(i, n int) (x, y, vx, vy float64)

type Vector struct {
	X, Y float64
}

type Domain struct {
	X, Y float64
}

type FluidSim struct {
	Particles []core.Particle
	N         int     // Number of particles
	Dt        float64 // Time step
	Rho0, Nu  float64 // Reference density and viscosity
	Domain    Domain  // Domain of the simulation
	Grid      *spatial.Grid
}

func RandomInitialCondition(i, n int) (float64, float64, float64, float64) {
	x := rand.Float64() * 100.0
	y := rand.Float64() * 100.0
	vx := (rand.Float64() * 2.0) - 1.0
	vy := (rand.Float64() * 2.0) - 1.0
	return x, y, vx, vy
}

func NewFluidSim(n int, dt, domainX, domainY, rho0, nu float64) *FluidSim {
	particles := make([]core.Particle, n)
	for i := 0; i < n; i++ {
		particles[i].X, particles[i].Y, particles[i].Vx, particles[i].Vy = RandomInitialCondition(i, n)
		particles[i].Density = rho0
	}

	grid := spatial.NewGrid(5.0, int(domainX), int(domainY))

	return &FluidSim{
		Particles: particles,
		N:         n,
		Dt:        dt,
		Domain:    Domain{X: domainX, Y: domainY},
		Rho0:      rho0,
		Nu:        nu,
		Grid:      grid,
	}
}

func (sim *FluidSim) CalculatePressureForce(p core.Particle) core.Vector {
	var force core.Vector
	for _, neighbor := range p.Neighbors {
		dx := neighbor.X - p.X
		dy := neighbor.Y - p.Y
		gradW := spatial.SmoothingKernelGradient(p)
		sharedPressure := (p.Pressure + neighbor.Pressure) / (2 * neighbor.Density)
		scaledPressure := sharedPressure * PressureMultiplier

		force.X += gradW.X * scaledPressure * dx
		force.Y += gradW.Y * scaledPressure * dy
	}
	return force
}

func (sim *FluidSim) CalculateViscosityForce(p core.Particle) core.Vector {
	var force core.Vector
	for _, neighbor := range p.Neighbors {
		dx := neighbor.X - p.X
		dy := neighbor.Y - p.Y

		lapW := spatial.SmoothingKernelLaplacian(p)
		viscosityTerm := (neighbor.Vx - p.Vx + neighbor.Vy - p.Vy) / neighbor.Density

		force.X += sim.Nu * lapW * viscosityTerm * dx
		force.Y += sim.Nu * lapW * viscosityTerm * dy
	}
	return force
}

func (sim *FluidSim) FindNeighbors() {
	parallelFor(0, len(sim.Particles), func(i int) {
		// Clear existing neighbors
		sim.Particles[i].Neighbors = nil

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

						if distanceSquared < NEIGHBOR_RADIUS*NEIGHBOR_RADIUS {
							sim.Particles[i].Neighbors = append(sim.Particles[i].Neighbors, sim.Particles[neighborIdx])
						}
					}
				}
			}
		}
	})
}

func (sim *FluidSim) UpdateDensity() {
	mass := 1.0
	for i, p := range sim.Particles {
		density := 0.0
		for _, neighbor := range p.Neighbors {
			distance := core.CalculateDistance(p, neighbor)
			influence := spatial.SmoothingKernel(NEIGHBOR_RADIUS, distance)
			density += mass * influence
		}
		sim.Particles[i].Density = density
	}
}

// Pressure Calculation
func (sim *FluidSim) UpdatePressure() {
	for i, p := range sim.Particles {
		sim.Particles[i].Pressure = K * (p.Density - sim.Rho0)
	}
}

func (sim *FluidSim) UpdateForces(gravity float64) {
	// Reset forces for all particles
	for i := range sim.Particles {
		sim.Particles[i].Force = core.Vector{X: 0, Y: 0}
	}

	// Add gravitational force (multiplied by density to simulate effect of mass)
	for i := range sim.Particles {
		sim.Particles[i].Force.Y -= sim.Particles[i].Density * gravity
	}

	for i := range sim.Particles {
		p1 := sim.Particles[i]
		pressureForce := sim.CalculatePressureForce(p1)
		viscosityForce := sim.CalculateViscosityForce(p1)

		p1.Force.X += pressureForce.X + viscosityForce.X
		p1.Force.Y += pressureForce.Y + viscosityForce.Y
	}
}

func (sim *FluidSim) Integrate() {
	for i := range sim.Particles {
		// update velocities
		sim.Particles[i].Vx += sim.Particles[i].Force.X * sim.Dt
		sim.Particles[i].Vy += sim.Particles[i].Force.Y * sim.Dt

		// update positions
		sim.Particles[i].X += sim.Particles[i].Vx * sim.Dt
		sim.Particles[i].Y += sim.Particles[i].Vy * sim.Dt

		// -- boundary conditions --

		// bounce off top and bottom
		if sim.Particles[i].Y > sim.Domain.Y {
			sim.Particles[i].Y = sim.Domain.Y
			sim.Particles[i].Vy *= -DAMPENING_FACTOR
		} else if sim.Particles[i].Y < 0 {
			sim.Particles[i].Y = 0
			sim.Particles[i].Vy *= -DAMPENING_FACTOR
		}

		// Loop-around from left to right (periodic boundary)
		if sim.Particles[i].X < 0 {
			sim.Particles[i].X += sim.Domain.X // Move particle to right boundary
		} else if sim.Particles[i].X > sim.Domain.X {
			sim.Particles[i].X -= sim.Domain.X // Move particle to left boundary
		}
	}
}

// ####################################################################################################

func (sim *FluidSim) Step(gravity float64) {
	sim.Grid.Update(sim.Particles)
	sim.FindNeighbors()
	sim.UpdateDensity()
	sim.UpdatePressure()
	sim.UpdateForces(gravity)
	sim.Integrate()
}
