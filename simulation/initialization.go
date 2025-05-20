package simulation

import (
	"math/rand"
)

// Domain struct might also move here if it's primarily for initialization context,
// but for now, assume it stays with FluidSim or a general types file.

// InitialConditionFunc defines the signature for functions that set initial particle states.
type InitialConditionFunc func(i int, domain Domain) (x, y, vx, vy float64)

// RandomStillInitialCondition creates particles with random positions but no velocity.
func RandomStillInitialCondition(i int, domain Domain) (float64, float64, float64, float64) {
	x := rand.Float64() * domain.X
	y := rand.Float64() * domain.Y
	// Random jitter for more natural initial positions
	vx := (rand.Float64() * 0.2) - 0.1
	vy := (rand.Float64() * 0.2) - 0.1
	return x, y, vx, vy
}

// RandomMotionInitialCondition creates particles with random positions and velocities.
func RandomMotionInitialCondition(i int, domain Domain) (float64, float64, float64, float64) {
	x := rand.Float64() * domain.X
	y := rand.Float64() * domain.Y
	vx := (rand.Float64() * 2.0) - 1.0
	vy := (rand.Float64() * 2.0) - 1.0
	return x, y, vx, vy
}
