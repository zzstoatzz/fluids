package simulation

// SimParameters holds the tunable parameters for the simulation physics and behavior.
// These can be overridden by command-line flags.
type SimParameters struct {
	InteractionRadius  float64 // Kernel radius h, also used for grid cell size
	SmoothingFactor    float64 // Factor to adjust effective distance for kernel derivatives (prevents extreme forces)
	DampeningFactor    float64 // General fluid drag/dampening on velocity
	DragEnabled        bool    // Whether fluid drag is active
	AttractionFactor   float64 // Strength of n-body attraction/repulsion force
	Rho0               float64 // Reference density for pressure calculations
	Nu                 float64 // Viscosity coefficient
	PressureMultiplier float64 // Stiffness of the fluid (pressure response to density change)
	Dt                 float64 // Simulation physics time step
	Gravity            float64 // Strength of gravity
	MouseForce         float64 // Strength of mouse interaction force
}

// GetDefaultSimParameters returns a new instance of SimParameters with default values.
func GetDefaultSimParameters() SimParameters {
	return SimParameters{
		InteractionRadius:  4.0,  // Default kernel radius h
		SmoothingFactor:    0.15, // Default from original -smooth flag
		DampeningFactor:    0.1,  // Default from original -drag flag
		DragEnabled:        true,
		AttractionFactor:   -100,
		Rho0:               1000.0,
		Nu:                 0.5,
		PressureMultiplier: 20000.0,
		Dt:                 0.0008,
		Gravity:            0,     // Default gravity off, can be toggled
		MouseForce:         500.0, // Default mouse interaction strength
	}
}
