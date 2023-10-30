package spatial

import "math"

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
