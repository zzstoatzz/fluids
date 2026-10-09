package simulation

import (
	"math/rand"
)

// Domain struct might also move here if it's primarily for initialization context,
// but for now, assume it stays with FluidSim or a general types file.

// InitialConditionFunc defines the signature for functions that set initial particle states.
type InitialConditionFunc func(i int, domain Domain, rng *rand.Rand) (x, y, vx, vy float64)

// RandomStillInitialCondition creates particles with random positions but no velocity.
func RandomStillInitialCondition(i int, domain Domain, rng *rand.Rand) (float64, float64, float64, float64) {
	x := rng.Float64() * domain.X
	y := rng.Float64() * domain.Y
	// Random jitter for more natural initial positions
	vx := (rng.Float64() * 0.2) - 0.1
	vy := (rng.Float64() * 0.2) - 0.1
	return x, y, vx, vy
}

// RandomMotionInitialCondition creates particles with random positions and velocities.
func RandomMotionInitialCondition(i int, domain Domain, rng *rand.Rand) (float64, float64, float64, float64) {
	x := rng.Float64() * domain.X
	y := rng.Float64() * domain.Y
	vx := (rng.Float64() * 2.0) - 1.0
	vy := (rng.Float64() * 2.0) - 1.0
	return x, y, vx, vy
}
