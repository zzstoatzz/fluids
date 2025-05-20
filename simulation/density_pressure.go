package simulation

import (
	"fluids/spatial"
	"math"
	"sync/atomic" // For CalculatePressureStats
)

// UpdateDensities calculates density for each particle based on neighbors
func (sim *FluidSim) UpdateDensities() {
	parallelFor(0, len(sim.Particles), func(i int) {
		sim.Particles[i].Density = spatial.CalculateDensity(sim.Particles, i, sim.InteractionRadius, sim.SmoothingFactor)
	})
}

// UpdatePressure calculates pressure from density
func (sim *FluidSim) UpdatePressure(pressureMultiplier float64) {
	parallelFor(0, len(sim.Particles), func(i int) {
		sim.Particles[i].Pressure = pressureMultiplier * (sim.Particles[i].Density - sim.Rho0)
	})
}

// CalculatePressureStats computes mean and standard deviation of pressure
func (sim *FluidSim) CalculatePressureStats() (float64, float64) {
	n := len(sim.Particles)
	if n == 0 {
		return 0, 0
	}
	// Reset atomic counters (sim.localAtomics is a field of FluidSim)
	for i := range sim.localAtomics {
		sim.localAtomics[i] = 0
	}
	parallelFor(0, n, func(i int) {
		workerID := i % len(sim.localAtomics)
		pressureScaled := int64(sim.Particles[i].Pressure * 1000)
		atomic.AddInt64(&sim.localAtomics[workerID], pressureScaled)
	})
	var totalSum int64
	for _, localSum := range sim.localAtomics {
		totalSum += localSum
	}
	meanPressure := float64(totalSum) / (float64(n) * 1000.0)

	// Reset stats buffers (sim.statsBuffers is a field of FluidSim)
	for i := range sim.statsBuffers {
		sim.statsBuffers[i] = 0
	}
	parallelFor(0, n, func(i int) {
		workerID := i % (len(sim.statsBuffers) / 2)
		diff := sim.Particles[i].Pressure - meanPressure
		sim.statsBuffers[workerID] += diff * diff
	})
	var varianceSum float64
	for i := 0; i < len(sim.statsBuffers)/2; i++ {
		varianceSum += sim.statsBuffers[i]
	}
	stdPressure := math.Sqrt(varianceSum / float64(n))
	return meanPressure, stdPressure
}
