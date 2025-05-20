package spatial

import (
	"math"
	"math/rand"
)

const EPSILON = 0.001
const DAMPENING_FACTOR = 0.8  // Using 0.8 instead of 0.7 for better elasticity
const RANDOMIZATION_FACTOR = 0.05  // Add small random variations to prevent perfect reflections

func Clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func Fmod(a, b float64) float64 {
	return a - b*math.Floor(a/b)
}

type BoundaryType int

const (
	Reflective BoundaryType = iota
	Periodic
)

// RandomJitter adds a small random value to prevent perfect reflections
func RandomJitter(baseVelocity float64) float64 {
	// Add random jitter in the range of [-RANDOMIZATION_FACTOR, RANDOMIZATION_FACTOR] * |baseVelocity|
	return (rand.Float64()*2 - 1.0) * RANDOMIZATION_FACTOR * math.Abs(baseVelocity)
}

// HandleBoundary processes collision with domain boundaries
// Now with added randomization for more natural looking behavior
func HandleBoundary(position *float64, velocity *float64, limit float64, boundaryType BoundaryType) {
	// Tiny safety offset to prevent particles from getting stuck at boundaries
	const pushOut = EPSILON * 2.0
	
	if *position >= limit {
		if boundaryType == Reflective {
			*position = limit - pushOut
			
			// Reverse and dampen velocity
			*velocity *= -DAMPENING_FACTOR
			
			// Add small random angle variation to avoid perfect reflection loops
			jitter := RandomJitter(*velocity)
			// For particles hitting top/bottom, add jitter to X velocity
			// This is a simplification - in a more complex system we'd use reflection vectors
			otherVelocity := jitter
			*velocity += otherVelocity * 0.1 // Scale to keep the effect subtle
		} else if boundaryType == Periodic {
			*position = 0
		}
	} else if *position <= 0 {
		if boundaryType == Reflective {
			*position = pushOut
			
			// Reverse and dampen velocity
			*velocity *= -DAMPENING_FACTOR
			
			// Add small random angle variation
			jitter := RandomJitter(*velocity)
			otherVelocity := jitter
			*velocity += otherVelocity * 0.1
		} else if boundaryType == Periodic {
			*position = limit
		}
	}
}
