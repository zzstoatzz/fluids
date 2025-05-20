package input

import (
	"fluids/simulation"
	"fluids/spatial"
	"math"
)

// ApplyMouseForceToParticles applies an explosion force to particles near the mouse position
// This creates an interactive explosion effect when the user clicks
func ApplyMouseForceToParticles(
	sim *simulation.FluidSim,
	mouseX, mouseY, windowWidth, windowHeight int32,
	mouseForce float64,
	forceRadius float64,
) {
	// Convert screen coordinates to simulation coordinates
	x_norm := float64(mouseX) / float64(windowWidth) * sim.Domain.X
	y_norm := float64(mouseY) / float64(windowHeight) * sim.Domain.Y

	// Configure explosion radius and falloff
	forceRadiusSq := forceRadius * forceRadius

	// Use spatial grid for more efficient explosion calculation
	// Only check particles in cells near the mouse position
	cellSize := sim.Grid.CellSize

	// Determine grid cell containing the mouse position
	mouseCellX := int(x_norm / cellSize)
	mouseCellY := int(y_norm / cellSize)

	// Calculate check radius in cells
	checkRadiusCells := int(math.Ceil(forceRadius / cellSize))

	// Track particles we've already affected to avoid duplicates
	checkedParticles := make(map[int]bool)

	// Iterate through cells in radius around mouse position
	for dx := -checkRadiusCells; dx <= checkRadiusCells; dx++ {
		for dy := -checkRadiusCells; dy <= checkRadiusCells; dy++ {
			cellX := mouseCellX + dx
			cellY := mouseCellY + dy

			// Get cell index
			cellIdx := spatial.MakeCellIndex(cellX, cellY)

			// Get particles in this cell
			if indices, found := sim.Grid.CellMap[cellIdx]; found {
				// Process particles in this cell
				for _, i := range indices {
					// Skip if already processed
					if checkedParticles[i] {
						continue
					}

					// Mark as processed
					checkedParticles[i] = true

					// Calculate vector from mouse to particle
					dx := sim.Particles[i].X - x_norm
					dy := sim.Particles[i].Y - y_norm

					// Calculate squared distance
					distanceSquared := dx*dx + dy*dy

					// Skip if outside explosion radius
					if distanceSquared > forceRadiusSq {
						continue
					}

					// Skip if too close to avoid extreme forces
					if distanceSquared < 1e-6 {
						continue
					}

					// Calculate distance and normalize direction
					distance := math.Sqrt(distanceSquared)
					dirX := dx / distance
					dirY := dy / distance

					// Force is stronger closer to the center and falls off linearly
					forceFactor := (1.0 - distance/forceRadius) * mouseForce

					// Apply force - pushing outward from click point
					sim.Particles[i].Vx += dirX * forceFactor
					sim.Particles[i].Vy += dirY * forceFactor
				}
			}
		}
	}
}
