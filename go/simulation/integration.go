package simulation

import (
	"fluids/spatial"
	// "math" // Not directly needed by these specific functions
)

// PredictPositions updates particle positions based on current velocities
// This is part of a semi-implicit integration scheme
func (sim *FluidSim) PredictPositions(dt float64) {
	parallelFor(0, len(sim.Particles), func(i int) {
		p := &sim.Particles[i]
		p.X += p.Vx * dt
		p.Y += p.Vy * dt
	})
}

// Integrate updates velocities and positions based on forces
func (sim *FluidSim) Integrate(dt float64) {
	parallelFor(0, len(sim.Particles), func(i int) {
		p := &sim.Particles[i]
		if p.Mass > 0 {
			p.Vx += (p.Force.X / p.Mass) * dt
			p.Vy += (p.Force.Y / p.Mass) * dt
		} else {
			p.Vx += p.Force.X * dt
			p.Vy += p.Force.Y * dt
		}
		p.X += p.Vx * dt
		p.Y += p.Vy * dt
		spatial.HandleBoundary(&p.X, &p.Vx, sim.Domain.X, sim.LeftBoundary)
		spatial.HandleBoundary(&p.Y, &p.Vy, sim.Domain.Y, sim.TopBoundary)
	})
}
