package spatial

import (
	"fluids/core"
	"math"
)

const (
	SMOOTHING_RADIUS = 4.0
	SMOOTHING_FACTOR = 0.2 // Like the JS implementation, smooths out extreme forces at small distances
)

// Precompute constants for better performance
var (
	// Volume normalization for smoothing kernel
	smoothingVolume = (math.Pi + math.Pow(SMOOTHING_RADIUS, 4)) / 6
	
	// Scale factor for smoothing kernel derivative
	smoothingScale = 12 / (math.Pow(math.Pi, 4) * math.Pi)
	
	// Square of smoothing radius for faster distance checks
	SmoothingRadiusSq = SMOOTHING_RADIUS * SMOOTHING_RADIUS
)

// SmoothingKernel calculates the kernel function
// This is optimized by precomputing the volume denominator
func SmoothingKernel(distance float64) float64 {
	// thank you mr. sebastian lague - https://www.youtube.com/watch?v=rSKMYc1CQHE
	if distance >= SMOOTHING_RADIUS {
		return 0
	}
	
	// Compute with precomputed volume constant
	return (SMOOTHING_RADIUS - distance) * (SMOOTHING_RADIUS - distance) / smoothingVolume
}

// SmoothingKernelDerivative gives the derivative of the kernel
// Optimized with precomputed scale factor
func SmoothingKernelDerivative(distance float64) float64 {
	if distance >= SMOOTHING_RADIUS {
		return 0
	}
	
	// Use precomputed scale constant
	return (distance - SMOOTHING_RADIUS) * smoothingScale
}

// CalculateDistance returns the distance between two particles
// Inlined here for performance (avoiding function call overhead)
func CalculateDistance(p1, p2 *core.Particle) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// CalculateDistanceSq returns the squared distance between two particles
// This avoids a square root operation when just comparing distances
func CalculateDistanceSq(p1, p2 *core.Particle) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return dx*dx + dy*dy
}

// CalculateDensity computes the density at a point based on its neighbors
func CalculateDensity(particles []core.Particle, idx int) float64 {
	density := 0.0
	p := &particles[idx]
	
	// Uses the index list for faster traversal
	for _, neighborIdx := range p.NeighborIndices {
		neighbor := &particles[neighborIdx]
		
		// Skip self-reference
		if idx == neighborIdx {
			continue
		}
		
		// Calculate distance
		dx := p.X - neighbor.X
		dy := p.Y - neighbor.Y
		distSq := dx*dx + dy*dy
		
		// Check if within smoothing radius
		if distSq < SmoothingRadiusSq {
			distance := math.Sqrt(distSq)
			influence := SmoothingKernel(distance)
			density += influence
		}
	}
	
	// Add self influence for stability
	density += SmoothingKernel(0)
	
	return density
}

// SmoothingKernelGradient calculates the gradient of the kernel
func SmoothingKernelGradient(particles []core.Particle, idx int) core.Vector {
	gradW := core.Vector{}
	p := &particles[idx]
	
	for _, neighborIdx := range p.NeighborIndices {
		// Skip self
		if idx == neighborIdx {
			continue
		}
		
		neighbor := &particles[neighborIdx]
		
		// Direction vector to neighbor
		dx := neighbor.X - p.X
		dy := neighbor.Y - p.Y
		distSq := dx*dx + dy*dy
		
		// Skip if out of range
		if distSq >= SmoothingRadiusSq {
			continue
		}
		
		// Calculate smoothed distance with minimum bound to avoid extreme forces
		distance := math.Sqrt(distSq)
		smoothedDistance := math.Max(distance, SMOOTHING_FACTOR * SMOOTHING_RADIUS)
		
		// Skip if something went wrong
		if smoothedDistance < EPSILON {
			continue
		}
		
		// Create direction vector from p to neighbor
		dir := core.Vector{
			X: dx / smoothedDistance,
			Y: dy / smoothedDistance,
		}
		
		// Apply derivative scaling
		derivative := SmoothingKernelDerivative(smoothedDistance)
		dir.Multiply(derivative)
		
		// Add to gradient
		gradW.Add(&dir)
	}
	
	return gradW
}

// SmoothingKernelLaplacian calculates the Laplacian of the kernel
func SmoothingKernelLaplacian(particles []core.Particle, idx int) float64 {
	laplacian := 0.0
	p := &particles[idx]
	
	for _, neighborIdx := range p.NeighborIndices {
		// Skip self
		if idx == neighborIdx {
			continue
		}
		
		neighbor := &particles[neighborIdx]
		
		// Calculate distance
		dx := neighbor.X - p.X
		dy := neighbor.Y - p.Y
		distSq := dx*dx + dy*dy
		
		// Skip if out of range
		if distSq >= SmoothingRadiusSq {
			continue
		}
		
		// Calculate smoothed distance with minimum bound
		distance := math.Sqrt(distSq)
		smoothedDistance := math.Max(distance, SMOOTHING_FACTOR * SMOOTHING_RADIUS)
		
		// Add to laplacian
		laplacian += SmoothingKernelDerivative(smoothedDistance)
	}
	
	return laplacian
}
