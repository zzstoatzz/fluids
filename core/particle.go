package core

type Vector struct {
	X, Y float64
}

type Particle struct {
	X, Y      float64 // Position
	Vx, Vy    float64 // Velocity
	Density   float64
	Pressure  float64
	Force     Vector // Force
	Neighbors []Particle
}
