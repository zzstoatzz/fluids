package simulation

import (
	"fluids/core"
	"fluids/spatial"
	"math"
)

// ApplyGravityForces adds gravitational force to each particle's Force vector.
func (sim *FluidSim) ApplyGravityForces(gravity float64) {
	parallelFor(0, len(sim.Particles), func(i int) {
		if sim.Particles[i].Mass > 0 { // Or consider density if gravity is mass-proportional
			sim.Particles[i].Force.Y += gravity * sim.Particles[i].Mass // Assuming gravity acts on mass
		}
	})
}

// CalculateStandardForces calculates pressure, viscosity, and repulsion forces
// and adds them to each particle's Force vector.
func (sim *FluidSim) CalculateStandardForces(pressureMultiplier float64) {
	parallelFor(0, len(sim.Particles), func(i int) {
		pressureForce := sim.CalculatePressureForce(i, pressureMultiplier)
		viscosityForce := sim.CalculateViscosityForce(i)
		repulsionForce := sim.CalculateRepulsionForce(i, pressureMultiplier)

		sim.Particles[i].Force.X += pressureForce.X + viscosityForce.X + repulsionForce.X
		sim.Particles[i].Force.Y += pressureForce.Y + viscosityForce.Y + repulsionForce.Y
	})
}

// CalculatePressureForce computes pressure-based force for a particle
func (sim *FluidSim) CalculatePressureForce(idx int, pressureMultiplier float64) core.Vector {
	var force core.Vector
	p := &sim.Particles[idx]
	hSq := sim.InteractionRadius * sim.InteractionRadius

	for _, neighborIdx := range p.NeighborIndices {
		neighbor := &sim.Particles[neighborIdx]
		dx := neighbor.X - p.X
		dy := neighbor.Y - p.Y
		distSq := dx*dx + dy*dy
		if distSq < spatial.KERNEL_EPSILON || distSq >= hSq {
			continue
		}
		dist := math.Sqrt(distSq)
		smoothedDist := math.Max(dist, sim.SmoothingFactor*sim.InteractionRadius)
		invDist := 1.0 / smoothedDist
		dirX := dx * invDist
		dirY := dy * invDist
		avgPressure := (p.Pressure + neighbor.Pressure) * 0.5
		forceMagnitude := avgPressure * spatial.SmoothingKernelDerivative(smoothedDist, sim.InteractionRadius) * pressureMultiplier
		force.X += dirX * forceMagnitude
		force.Y += dirY * forceMagnitude
	}
	return force
}

// CalculateViscosityForce computes viscosity-based force for a particle
func (sim *FluidSim) CalculateViscosityForce(idx int) core.Vector {
	var force core.Vector
	p := &sim.Particles[idx]
	hSq := sim.InteractionRadius * sim.InteractionRadius

	for _, neighborIdx := range p.NeighborIndices {
		neighbor := &sim.Particles[neighborIdx]
		dvx := neighbor.Vx - p.Vx
		dvy := neighbor.Vy - p.Vy
		dx := neighbor.X - p.X
		dy := neighbor.Y - p.Y
		distSq := dx*dx + dy*dy
		if distSq < spatial.KERNEL_EPSILON || distSq >= hSq {
			continue
		}
		dist := math.Sqrt(distSq)
		smoothedDist := math.Max(dist, sim.SmoothingFactor*sim.InteractionRadius)
		viscosityFactor := spatial.SmoothingKernelDerivative(smoothedDist, sim.InteractionRadius) * sim.Nu
		force.X += dvx * viscosityFactor
		force.Y += dvy * viscosityFactor
	}
	return force
}

// CalculateRepulsionForce computes density-based repulsion forces
func (sim *FluidSim) CalculateRepulsionForce(idx int, pressureMultiplier float64) core.Vector {
	var force core.Vector
	p := &sim.Particles[idx]
	hSq := sim.InteractionRadius * sim.InteractionRadius

	for _, neighborIdx := range p.NeighborIndices {
		neighbor := &sim.Particles[neighborIdx]
		densityDiff := neighbor.Density - p.Density
		if densityDiff <= 0 {
			continue
		}
		dx := p.X - neighbor.X
		dy := p.Y - neighbor.Y
		distSq := dx*dx + dy*dy
		if distSq < spatial.KERNEL_EPSILON || distSq >= hSq {
			continue
		}
		dist := math.Sqrt(distSq)
		dirX := dx / dist
		dirY := dy / dist
		repulsionStrength := math.Pow(densityDiff, 1.5) * 0.1 * pressureMultiplier
		force.X += dirX * repulsionStrength
		force.Y += dirY * repulsionStrength
	}
	return force
}

// CalculateAttractionForces applies gravitational-like attraction/repulsion between particles
func (sim *FluidSim) CalculateAttractionForces() {
	// ... implementation from sph.go ...
	if math.Abs(sim.AttractionFactor) < 1e-6 || sim.InteractionRadius <= 0 {
		return
	}
	cellKeys := make([]spatial.CellIndex, 0, len(sim.Grid.CellMap))
	for k := range sim.Grid.CellMap {
		cellKeys = append(cellKeys, k)
	}
	numTasks := len(cellKeys)
	if numTasks == 0 {
		return
	}
	taskAccumulators := make([]map[int]*core.Vector, numTasks)
	for i := 0; i < numTasks; i++ {
		taskAccumulators[i] = make(map[int]*core.Vector)
	}
	interactionRadiusSq := sim.InteractionRadius * sim.InteractionRadius
	forceScale := sim.AttractionFactor
	parallelFor(0, numTasks, func(taskIdx int) {
		cellKey := cellKeys[taskIdx]
		cellParticleIndices, ok := sim.Grid.CellMap[cellKey]
		if !ok {
			return
		}
		sim.accumulateCellAttractionForces(cellKey, cellParticleIndices, forceScale, interactionRadiusSq, taskAccumulators[taskIdx])
	})
	sim.reduceLocalForces(taskAccumulators)
}

// accumulateCellAttractionForces calculates attraction forces for particles in a specific cell
func (sim *FluidSim) accumulateCellAttractionForces(
	cellKey spatial.CellIndex,
	cellParticleIndices []int,
	forceScale float64,
	interactionRadiusSq float64,
	targetAccumulator map[int]*core.Vector,
) {
	cellX, cellY := cellKey.GetCoordinates()
	for nx := cellX - 1; nx <= cellX+1; nx++ {
		for ny := cellY - 1; ny <= cellY+1; ny++ {
			neighborCellMapKey := spatial.MakeCellIndex(nx, ny)
			neighborCellParticleIndices, found := sim.Grid.CellMap[neighborCellMapKey]
			if !found {
				continue
			}
			for _, p1Idx := range cellParticleIndices {
				p1 := &sim.Particles[p1Idx]
				for _, p2Idx := range neighborCellParticleIndices {
					if p1Idx >= p2Idx {
						continue
					}
					p2 := &sim.Particles[p2Idx]
					dx := p2.X - p1.X
					dy := p2.Y - p1.Y
					distSq := dx*dx + dy*dy
					if distSq >= interactionRadiusSq || distSq < spatial.KERNEL_EPSILON {
						continue
					}
					distance := math.Sqrt(distSq)
					smoothedDist := math.Max(distance, sim.SmoothingFactor*sim.InteractionRadius)
					if smoothedDist < spatial.KERNEL_EPSILON {
						continue
					}
					massFactor := p1.Mass * p2.Mass
					forceMagnitude := forceScale * massFactor / (smoothedDist * smoothedDist)
					G := forceMagnitude / distance
					forceX := G * dx
					forceY := G * dy
					if math.IsNaN(forceX) || math.IsNaN(forceY) {
						continue
					}
					p1ForceVec, found := targetAccumulator[p1Idx]
					if !found {
						p1ForceVec = &core.Vector{}
						targetAccumulator[p1Idx] = p1ForceVec
					}
					p1ForceVec.X += forceX
					p1ForceVec.Y += forceY
					p2ForceVec, found := targetAccumulator[p2Idx]
					if !found {
						p2ForceVec = &core.Vector{}
						targetAccumulator[p2Idx] = p2ForceVec
					}
					p2ForceVec.X -= forceX
					p2ForceVec.Y -= forceY
				}
			}
		}
	}
}

// reduceLocalForces merges forces from local accumulators into global particle forces.
func (sim *FluidSim) reduceLocalForces(localAccumulators []map[int]*core.Vector) {
	// ... implementation from sph.go ...
	for _, workerMap := range localAccumulators {
		for particleIndex, deltaForceVec := range workerMap {
			if particleIndex < 0 || particleIndex >= len(sim.Particles) {
				continue
			}
			if deltaForceVec == nil {
				continue
			}
			atomicAddFloat64(&sim.Particles[particleIndex].Force.X, deltaForceVec.X)
			atomicAddFloat64(&sim.Particles[particleIndex].Force.Y, deltaForceVec.Y)
		}
	}
}

// ApplyDrag applies velocity-dependent drag to all particles
func (sim *FluidSim) ApplyDrag(dt float64) {
	// ... implementation from sph.go ...
	if !sim.DragEnabled {
		return
	}
	dragFactor := 1.0 - (sim.DampeningFactor * dt * 60.0)
	parallelFor(0, len(sim.Particles), func(i int) {
		p := &sim.Particles[i]
		speedSq := p.Vx*p.Vx + p.Vy*p.Vy
		if speedSq < 1e-6 {
			return
		}
		p.Vx *= dragFactor
		p.Vy *= dragFactor
	})
}
