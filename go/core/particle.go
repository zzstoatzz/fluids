package core

import "math"

type Vector struct {
	X, Y float64
}

func (v *Vector) Add(other *Vector) {
	v.X += other.X
	v.Y += other.Y
}

func (v *Vector) Subtract(other *Vector) {
	v.X -= other.X
	v.Y -= other.Y
}

func (v *Vector) Multiply(scalar float64) {
	v.X *= scalar
	v.Y *= scalar
}

func (v Vector) MultiplyByScalar(scalar float64) *Vector {
	return &Vector{
		X: v.X * scalar,
		Y: v.Y * scalar,
	}
}

type Particle struct {
	X, Y      float64 // Position
	Vx, Vy    float64 // Velocity
	Density   float64
	Pressure  float64
	Force     Vector  // Force
	CellX, CellY int  // Current cell coordinates (optimization)
	NeighborIndices []int // Indices of neighboring particles (optimization)
	Neighbors []Particle  // Full particle objects for neighbors
	Mass      float64    // Particle mass, affects gravitational attraction
	Radius    float64    // Particle radius, mainly for rendering
}

func CalculateDistance(p1, p2 Particle) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return math.Sqrt(dx*dx + dy*dy)
}
