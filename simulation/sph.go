package simulation

import (
	"fluids/core"
	"fluids/spatial"
	"math"
	"math/rand"
	"runtime"
	"sync/atomic"
)

type InitialConditionFunc func(i int, domain Domain) (x, y, vx, vy float64)

type Domain struct {
	X, Y float64
}

// RandomStillInitialCondition creates particles with random positions but no velocity
func RandomStillInitialCondition(i int, domain Domain) (float64, float64, float64, float64) {
	x := rand.Float64() * domain.X
	y := rand.Float64() * domain.Y
	// Random jitter for more natural initial positions
	vx := (rand.Float64() * 0.2) - 0.1
	vy := (rand.Float64() * 0.2) - 0.1
	return x, y, vx, vy
}

// RandomMotionInitialCondition creates particles with random positions and velocities
func RandomMotionInitialCondition(i int, domain Domain) (float64, float64, float64, float64) {
	x := rand.Float64() * domain.X
	y := rand.Float64() * domain.Y
	vx := (rand.Float64() * 2.0) - 1.0
	vy := (rand.Float64() * 2.0) - 1.0
	return x, y, vx, vy
}

// FluidSim represents a fluid simulation using Smoothed Particle Hydrodynamics
type FluidSim struct {
	Particles    []core.Particle
	N            int     // Number of particles
	Dt           float64 // Time step
	Rho0, Nu     float64 // Reference density and viscosity
	Domain       Domain  // Domain of the simulation
	Grid         *spatial.Grid
	LeftBoundary spatial.BoundaryType
	TopBoundary  spatial.BoundaryType
	
	// Performance parameters
	SmoothingFactor    float64 // Smoothing factor for force calculations (like in JS)
	DampeningFactor    float64 // How much to dampen velocity (drag)
	DragEnabled        bool    // Whether to apply drag to particles
	AttractionFactor   float64 // N-body gravity-like attraction between particles (-ve = repulsion)
	InteractionRadius  float64 // Maximum distance for particle interaction
	
	// Temporary vectors/values for reuse (reduces allocation)
	tempVectors      []core.Vector  // Pre-allocated vectors for reuse
	localAtomics     []int64        // Atomic counter for thread-local sums
	statsBuffers     []float64      // Buffers for statistical calculations
}

// NewFluidSim creates a new fluid simulation
func NewFluidSim(n int, domain Domain, dt, rho0, nu float64) *FluidSim {
	// Create particles with initial conditions
	particles := make([]core.Particle, n)
	for i := 0; i < n; i++ {
		// Initialize position and velocity
		particles[i].X, particles[i].Y, particles[i].Vx, particles[i].Vy = RandomStillInitialCondition(i, domain)
		particles[i].Density = rho0
		
		// Pre-allocate neighbor list with reasonable capacity
		particles[i].NeighborIndices = make([]int, 0, 32)
		
		// Set particle radius with slight variation for more natural look
		baseRadius := 2.0
		particles[i].Radius = baseRadius * (0.8 + 0.4*rand.Float64()) // 0.8-1.2 times base radius
		
		// Calculate mass based on radius (assuming constant density)
		// Mass = π * r² (in 2D, area of circle)
		particles[i].Mass = math.Pi * particles[i].Radius * particles[i].Radius
	}

	// Create grid using smoothing radius as cell size
	grid := spatial.NewGrid(spatial.SMOOTHING_RADIUS, int(domain.X), int(domain.Y))
	
	// Create simulation
	sim := &FluidSim{
		Particles:        particles,
		N:                n,
		Dt:               dt,
		Domain:           domain,
		Rho0:             rho0,
		Nu:               nu,
		Grid:             grid,
		SmoothingFactor:  0.2,
		DampeningFactor:  0.05,
		DragEnabled:      true,
		AttractionFactor: -100, // Default: negative = repulsion like fluid, positive = attraction like gravity
		InteractionRadius: spatial.SMOOTHING_RADIUS * 2, // Double the smoothing radius
		LeftBoundary:     spatial.Reflective,
		TopBoundary:      spatial.Reflective,
		
		// Pre-allocate memory for temporary values
		tempVectors:      make([]core.Vector, n),
		localAtomics:     make([]int64, runtime.NumCPU()),
		statsBuffers:     make([]float64, runtime.NumCPU()*2),
	}
	
	return sim
}

// PredictPositions updates particle positions based on current velocities
// This is part of a semi-implicit integration scheme
func (sim *FluidSim) PredictPositions(dt float64) {
	parallelFor(0, len(sim.Particles), func(i int) {
		p := &sim.Particles[i]
		p.X += p.Vx * dt
		p.Y += p.Vy * dt
	})
}

// FindNeighbors populates each particle's neighbor indices list
// This uses the spatial grid for efficiency
func (sim *FluidSim) FindNeighbors() {
	parallelFor(0, len(sim.Particles), func(i int) {
		p := &sim.Particles[i]
		
		// Clear existing neighbors but keep capacity
		p.NeighborIndices = p.NeighborIndices[:0]
		
		// Get cell coordinates
		cellX, cellY := p.CellX, p.CellY
		
		// Get all potential neighbors from grid
		potentialNeighbors := sim.Grid.GetNeighborParticles(cellX, cellY)
		
		// Iterate through potential neighbors
		for _, neighborIdx := range potentialNeighbors {
			// Skip self
			if i == neighborIdx {
				continue
			}
			
			neighbor := &sim.Particles[neighborIdx]
			
			// Compute squared distance
			dx := p.X - neighbor.X
			dy := p.Y - neighbor.Y
			distSq := dx*dx + dy*dy
			
			// Add to neighbors if within smoothing radius
			if distSq < spatial.SmoothingRadiusSq {
				p.NeighborIndices = append(p.NeighborIndices, neighborIdx)
			}
		}
	})
}

// UpdateDensities calculates density for each particle based on neighbors
func (sim *FluidSim) UpdateDensities() {
	parallelFor(0, len(sim.Particles), func(i int) {
		sim.Particles[i].Density = spatial.CalculateDensity(sim.Particles, i)
	})
}

// UpdatePressure calculates pressure from density
func (sim *FluidSim) UpdatePressure(pressureMultiplier float64) {
	parallelFor(0, len(sim.Particles), func(i int) {
		// Simple equation of state: pressure = k * (density - rest_density)
		// Higher pressure multiplier = stiffer fluid
		sim.Particles[i].Pressure = pressureMultiplier * (sim.Particles[i].Density - sim.Rho0)
	})
}

// CalculatePressureForce computes pressure-based force for a particle
func (sim *FluidSim) CalculatePressureForce(idx int, pressureMultiplier float64) core.Vector {
	var force core.Vector
	p := &sim.Particles[idx]
	
	// Iterate through neighbors by index for better efficiency
	for _, neighborIdx := range p.NeighborIndices {
		neighbor := &sim.Particles[neighborIdx]
		
		// Direction vector
		dx := neighbor.X - p.X
		dy := neighbor.Y - p.Y
		distSq := dx*dx + dy*dy
		
		// Skip if too close (avoid division by near-zero)
		if distSq < spatial.EPSILON {
			continue
		}
		
		// Smoothing to prevent extreme forces
		dist := math.Sqrt(distSq)
		smoothedDist := math.Max(dist, spatial.SMOOTHING_FACTOR*spatial.SMOOTHING_RADIUS)
		
		// Calculate force direction
		invDist := 1.0 / smoothedDist
		dirX := dx * invDist
		dirY := dy * invDist
		
		// Pressure term using average pressure of both particles
		avgPressure := (p.Pressure + neighbor.Pressure) * 0.5
		
		// Calculate force magnitude
		forceMagnitude := avgPressure * spatial.SmoothingKernelDerivative(smoothedDist) * pressureMultiplier
		
		// Apply force
		force.X += dirX * forceMagnitude
		force.Y += dirY * forceMagnitude
	}
	
	return force
}

// CalculateViscosityForce computes viscosity-based force for a particle
func (sim *FluidSim) CalculateViscosityForce(idx int) core.Vector {
	var force core.Vector
	p := &sim.Particles[idx]
	
	// Iterate through neighbors by index
	for _, neighborIdx := range p.NeighborIndices {
		neighbor := &sim.Particles[neighborIdx]
		
		// Velocity difference
		dvx := neighbor.Vx - p.Vx
		dvy := neighbor.Vy - p.Vy
		
		// Direction vector
		dx := neighbor.X - p.X
		dy := neighbor.Y - p.Y
		distSq := dx*dx + dy*dy
		
		// Skip if too close
		if distSq < spatial.EPSILON {
			continue
		}
		
		// Smoothing to prevent extreme forces
		dist := math.Sqrt(distSq)
		smoothedDist := math.Max(dist, spatial.SMOOTHING_FACTOR*spatial.SMOOTHING_RADIUS)
		
		// Calculate viscosity factor
		viscosityFactor := spatial.SmoothingKernelDerivative(smoothedDist) * sim.Nu
		
		// Apply viscosity forces (proportional to velocity difference)
		force.X += dvx * viscosityFactor
		force.Y += dvy * viscosityFactor
	}
	
	return force
}

// CalculateRepulsionForce computes density-based repulsion forces
func (sim *FluidSim) CalculateRepulsionForce(idx int, pressureMultiplier float64) core.Vector {
	var force core.Vector
	p := &sim.Particles[idx]
	
	// Iterate through neighbors by index
	for _, neighborIdx := range p.NeighborIndices {
		neighbor := &sim.Particles[neighborIdx]
		
		// Only apply repulsion from higher density areas
		densityDiff := neighbor.Density - p.Density
		if densityDiff <= 0 {
			continue
		}
		
		// Direction vector (away from higher density)
		dx := p.X - neighbor.X
		dy := p.Y - neighbor.Y
		distSq := dx*dx + dy*dy
		
		// Skip if too close
		if distSq < spatial.EPSILON {
			continue
		}
		
		// Normalize direction
		dist := math.Sqrt(distSq)
		dirX := dx / dist
		dirY := dy / dist
		
		// Force is proportional to density difference and repulsion scaling
		// The power function creates a non-linear falloff - stronger repulsion at higher density differences
		repulsionStrength := math.Pow(densityDiff, 1.5) * 0.1 * pressureMultiplier
		
		// Apply repulsion force
		force.X += dirX * repulsionStrength
		force.Y += dirY * repulsionStrength
	}
	
	return force
}

// CalculateAttractionForces applies gravitational-like attraction/repulsion between particles
func (sim *FluidSim) CalculateAttractionForces(dt float64) {
	// Skip if attraction factor is near zero or interaction radius is invalid
	if math.Abs(sim.AttractionFactor) < 1e-6 || sim.InteractionRadius <= 0 {
		return
	}

	// Square the interaction radius for faster distance checks
	interactionRadiusSq := sim.InteractionRadius * sim.InteractionRadius
	
	// Pre-calculate the force scaling factor including deltaTime
	// This is the G constant in F = G * (m1*m2)/r²
	forceScale := sim.AttractionFactor * dt
	
	// Calculate forces between particles using spatial partitioning for efficiency
	// We'll only process unique pairs (i,j) where i < j to avoid duplicate calculations
	for cellIdx, indices := range sim.Grid.CellMap {
		cellX, cellY := cellIdx.GetCoordinates()
		
		// Check the current cell and all neighboring cells
		for nx := cellX - 1; nx <= cellX + 1; nx++ {
			for ny := cellY - 1; ny <= cellY + 1; ny++ {
				neighborIdx := spatial.MakeCellIndex(nx, ny)
				neighborIndices, found := sim.Grid.CellMap[neighborIdx]
				
				if !found {
					continue
				}
				
				// For each pair of particles, apply mutual attraction/repulsion
				for _, i := range indices {
					p1 := &sim.Particles[i]
					
					for _, j := range neighborIndices {
						// Only process unique pairs where i < j
						if i >= j {
							continue
						}
						
						p2 := &sim.Particles[j]
						
						// Calculate distance between particles
						dx := p2.X - p1.X
						dy := p2.Y - p1.Y
						distSq := dx*dx + dy*dy
						
						// Skip if too far apart or too close
						if distSq >= interactionRadiusSq || distSq < 1e-6 {
							continue
						}
						
						// Calculate the actual distance
						distance := math.Sqrt(distSq)
						
						// Apply smoothing factor to avoid extreme forces at small distances
						smoothedDist := math.Max(distance, sim.SmoothingFactor * sim.InteractionRadius)
						if smoothedDist < 1e-6 {
							continue
						}
						
						// Calculate force magnitude using inverse square law (like gravity)
						// F = G * (m1*m2) / r²
						massFactor := p1.Mass * p2.Mass // For future use if particles have different masses
						forceMagnitude := forceScale * massFactor / (smoothedDist * smoothedDist)
						
						// Normalize to get force direction and combine with magnitude
						// Optimization: divide by distance once instead of normalizing separately
						G := forceMagnitude / distance
						forceX := G * dx
						forceY := G * dy
						
						// Skip if NaN (can happen with extreme values)
						if math.IsNaN(forceX) || math.IsNaN(forceY) {
							continue
						}
						
						// Calculate acceleration for each particle (F = ma, so a = F/m)
						accX1 := forceX / p1.Mass
						accY1 := forceY / p1.Mass
						accX2 := -forceX / p2.Mass
						accY2 := -forceY / p2.Mass
						
						// Apply accelerations to velocities of both particles
						p1.Vx += accX1
						p1.Vy += accY1
						p2.Vx += accX2
						p2.Vy += accY2
					}
				}
			}
		}
	}
}

// ApplyDrag applies velocity-dependent drag to all particles
func (sim *FluidSim) ApplyDrag(dt float64) {
	if !sim.DragEnabled {
		return
	}
	
	// Only apply drag to moving particles to save computation
	// Use adjusted drag factor based on time step
	dragFactor := 1.0 - (sim.DampeningFactor * dt * 60.0) // 60 = normalization to 60fps
	
	parallelFor(0, len(sim.Particles), func(i int) {
		p := &sim.Particles[i]
		
		// Skip particles that are barely moving
		speedSq := p.Vx*p.Vx + p.Vy*p.Vy
		if speedSq < 1e-6 {
			return
		}
		
		// Apply dampening
		p.Vx *= dragFactor
		p.Vy *= dragFactor
	})
}

// UpdateForces calculates and applies all forces to particles
func (sim *FluidSim) UpdateForces(gravity, pressureMultiplier float64) {
	parallelFor(0, len(sim.Particles), func(i int) {
		// Reset force and apply gravity (positive gravity means downward force)
		// In screen coordinates, Y increases downward, so positive gravity pulls particles down
		sim.Particles[i].Force = core.Vector{X: 0, Y: gravity * sim.Particles[i].Density}
		
		// Calculate forces
		pressureForce := sim.CalculatePressureForce(i, pressureMultiplier)
		viscosityForce := sim.CalculateViscosityForce(i)
		repulsionForce := sim.CalculateRepulsionForce(i, pressureMultiplier)
		
		// Apply all forces
		sim.Particles[i].Force.X += pressureForce.X + viscosityForce.X + repulsionForce.X
		sim.Particles[i].Force.Y += pressureForce.Y + viscosityForce.Y + repulsionForce.Y
	})
}

// Integrate updates velocities and positions based on forces
func (sim *FluidSim) Integrate() {
	parallelFor(0, len(sim.Particles), func(i int) {
		p := &sim.Particles[i]
		
		// Update velocities based on forces
		p.Vx += p.Force.X * sim.Dt
		p.Vy += p.Force.Y * sim.Dt
		
		// Update positions based on velocities
		p.X += p.Vx * sim.Dt
		p.Y += p.Vy * sim.Dt
		
		// Handle boundary conditions
		spatial.HandleBoundary(&p.X, &p.Vx, sim.Domain.X, sim.LeftBoundary)
		spatial.HandleBoundary(&p.Y, &p.Vy, sim.Domain.Y, sim.TopBoundary)
	})
}

// CalculatePressureStats computes mean and standard deviation of pressure
// Uses atomic operations for thread-safe concurrent summing
func (sim *FluidSim) CalculatePressureStats() (float64, float64) {
	n := len(sim.Particles)
	if n == 0 {
		return 0, 0
	}
	
	// Reset atomic counters
	for i := range sim.localAtomics {
		sim.localAtomics[i] = 0
	}
	
	// Calculate mean pressure (sum all pressures divided by count)
	// Using parallel accumulation with thread-local sums to avoid locking
	parallelFor(0, n, func(i int) {
		// Get worker ID to use thread-local sum (avoids contention)
		workerID := i % len(sim.localAtomics)
		
		// Convert to int64 for atomic operations (with scaling for precision)
		// We scale by 1000 to maintain fractional precision in the int64
		pressureScaled := int64(sim.Particles[i].Pressure * 1000)
		
		// Add to thread-local sum atomically
		atomic.AddInt64(&sim.localAtomics[workerID], pressureScaled)
	})
	
	// Sum up all thread-local results
	var totalSum int64
	for _, localSum := range sim.localAtomics {
		totalSum += localSum
	}
	
	// Convert back to float64 and divide by scaling factor
	meanPressure := float64(totalSum) / (float64(n) * 1000.0)
	
	// Now calculate standard deviation
	// Reset buffers
	for i := range sim.statsBuffers {
		sim.statsBuffers[i] = 0
	}
	
	// Calculate sum of squared differences from mean
	parallelFor(0, n, func(i int) {
		workerID := i % (len(sim.statsBuffers) / 2)
		diff := sim.Particles[i].Pressure - meanPressure
		// Accumulate squared differences in thread-local buffer
		sim.statsBuffers[workerID] += diff * diff
	})
	
	// Sum up thread-local variances
	var varianceSum float64
	for i := 0; i < len(sim.statsBuffers)/2; i++ {
		varianceSum += sim.statsBuffers[i]
	}
	
	// Calculate standard deviation
	stdPressure := math.Sqrt(varianceSum / float64(n))
	
	return meanPressure, stdPressure
}

// Step advances the simulation by one time step
func (sim *FluidSim) Step(gravity, pressureMultiplier, dt float64) (float64, float64) {
	// 1. Move particles based on current velocities
	sim.PredictPositions(dt)
	
	// 2. Update spatial partitioning grid
	sim.Grid.Update(sim.Particles)
	
	// 3. Find neighbors for each particle
	sim.FindNeighbors()
	
	// 4. Calculate densities
	sim.UpdateDensities()
	
	// 5. Calculate pressures from densities
	sim.UpdatePressure(pressureMultiplier)
	
	// 6. Apply inter-particle attraction/repulsion forces
	sim.CalculateAttractionForces(dt)
	
	// 7. Calculate other forces (pressure, viscosity, repulsion)
	sim.UpdateForces(gravity, pressureMultiplier)
	
	// 8. Apply drag forces
	sim.ApplyDrag(dt)
	
	// 9. Update velocities and positions
	sim.Integrate()
	
	// 10. Calculate statistics for visualization/debugging
	meanPressure, stdPressure := sim.CalculatePressureStats()
	
	return meanPressure, stdPressure
}
