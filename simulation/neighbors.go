package simulation

// FindNeighbors populates each particle's neighbor indices list
func (sim *FluidSim) FindNeighbors() {
	interactionRadiusSq := sim.InteractionRadius * sim.InteractionRadius
	parallelFor(0, len(sim.Particles), func(i int) {
		p := &sim.Particles[i]
		p.NeighborIndices = p.NeighborIndices[:0]
		cellX, cellY := p.CellX, p.CellY // Assumes CellX, CellY are populated on particles
		potentialNeighbors := sim.Grid.GetNeighborParticles(cellX, cellY)
		for _, neighborIdx := range potentialNeighbors {
			if i == neighborIdx {
				continue
			}
			neighbor := &sim.Particles[neighborIdx]
			dx := p.X - neighbor.X
			dy := p.Y - neighbor.Y
			distSq := dx*dx + dy*dy
			if distSq < interactionRadiusSq {
				p.NeighborIndices = append(p.NeighborIndices, neighborIdx)
			}
		}
	})
}
