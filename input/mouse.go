package input

import (
	"fluids/simulation"
	"math"
)

func ApplyMouseForceToParticles(sim *simulation.FluidSim, mouseX, mouseY, windowWidth, windowHeight int32) {
	const forceRadius = 10.0
	const forceMagnitude = 30.0
	for i := range sim.Particles {
		x_norm := float64(mouseX) / float64(windowWidth) * sim.Domain.X
		y_norm := float64(mouseY) / float64(windowHeight) * sim.Domain.Y

		dx := sim.Particles[i].X - x_norm
		dy := sim.Particles[i].Y - y_norm

		distanceSquared := dx*dx + dy*dy
		if distanceSquared > forceRadius*forceRadius {
			continue
		}

		length := math.Sqrt(distanceSquared)
		if length == 0 {
			continue
		}
		dx /= length
		dy /= length

		sim.Particles[i].Vx += dx * forceMagnitude
		sim.Particles[i].Vy += dy * forceMagnitude
	}
}
