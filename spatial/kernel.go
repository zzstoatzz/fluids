package spatial

import (
	"fluids/core"
	"math"
)

const SMOOTHING_RADIUS = 3.0

func SmoothingKernel(radius, distance float64) float64 {
	// thank you mr. sebastian lague - https://www.youtube.com/watch?v=rSKMYc1CQHE

	if distance >= radius {
		return 0
	}
	volume := (math.Pi + math.Pow(radius, 4)) / 6
	return (radius - distance) * (radius - distance) / volume
}

func SmoothingKernelDerivative(radius, distance float64) float64 {
	if distance >= radius {
		return 0
	}
	scale := 12 / (math.Pow(math.Pi, 4) * math.Pi)
	return (distance - radius) * scale
}

func CalculateDensity(point core.Particle) float64 {
	density := 0.0

	for _, neighbor := range point.Neighbors {
		distance := core.CalculateDistance(point, neighbor)
		influence := SmoothingKernel(SMOOTHING_RADIUS, distance)
		density += 1 * influence
	}

	return density
}

func SmoothingKernelGradient(point core.Particle) core.Vector {
	var gradW core.Vector

	for _, neighbor := range point.Neighbors {
		distance := core.CalculateDistance(point, neighbor)
		slope := SmoothingKernelDerivative(SMOOTHING_RADIUS, distance)
		gradW.X += slope * (point.X - neighbor.X) / point.Density
	}

	return gradW
}

func SmoothingKernelLaplacian(point core.Particle) float64 {
	laplacian := 0.0

	for _, neighbor := range point.Neighbors {
		distance := core.CalculateDistance(point, neighbor)
		laplacian += SmoothingKernelDerivative(SMOOTHING_RADIUS, distance)
	}

	return laplacian
}
