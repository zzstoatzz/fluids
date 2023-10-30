package spatial

import "math"

const EPSILON = 0.001
const DAMPENING_FACTOR = 0.7

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

func HandleBoundary(position *float64, velocity *float64, limit float64, boundaryType BoundaryType) {
	if *position >= limit {
		if boundaryType == Reflective {
			*position = limit - EPSILON
			*velocity *= -DAMPENING_FACTOR
		}
	} else if *position <= 0 {
		if boundaryType == Reflective {
			*position = EPSILON
			*velocity *= -DAMPENING_FACTOR
		}
	}
}
