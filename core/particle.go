package core

import "math"

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

func CalculateDistance(p1, p2 Particle) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return math.Sqrt(dx*dx + dy*dy)
}
