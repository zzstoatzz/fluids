package main

import (
	"fluids/simulation"
	"fluids/viz"
	"math"
	"math/rand"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

func SineCosineInitialCondition(ampU, ampV, freqU, freqV float64) func(int, int, int, int) (float64, float64) {
	return func(i, j, nx, ny int) (float64, float64) {
		phaseU, phaseV := 0.0, 0.0 // default phase
		return ampU * math.Sin(freqU*float64(i+j)+phaseU), ampV * math.Cos(freqV*float64(i+j)+phaseV)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func CenterBunchInitialCondition(i, j, nx, ny int) (float64, float64) {
	centerX, centerY := nx/2, ny/2
	radius := min(nx, ny) / 4

	// Calculate distance from the center
	dist := math.Sqrt(float64((i-centerX)*(i-centerX) + (j-centerY)*(j-centerY)))

	if dist < float64(radius) {
		return 5.0, 5.0 // Set u and v velocities for points in the center
	}
	return 0.0, 0.0
}

func HorizontalTestInitialCondition(i, j, nx, ny int) (float64, float64) {
	if i == nx/2 {
		return 5.0, 0.0 // Set u-velocity to 5.0 for the middle row
	}
	return 0.0, 0.0
}

func ZeroInitialCondition(i, j, nx, ny int) (float64, float64) {
	return 0.0, 0.0
}

func RandomInitialCondition(seed int64) func(int, int, int, int) (float64, float64) {
	r := rand.New(rand.NewSource(seed))
	return func(i, j, nx, ny int) (float64, float64) {
		angle := 2 * math.Pi * r.Float64() // Random angle between 0 and 2*Pi
		speed := 4.0 * r.Float64()         // Random speed between 0 and 2

		return speed * math.Cos(angle), speed*math.Sin(angle) + 4.0
	}
}

func applyMouseForceToFluid(sim *simulation.FluidSim, mouseX, mouseY int32) {
	gridX, gridY := int(mouseX/8), int(mouseY/8) // Convert to grid coords; adjust scaling as needed
	forceX, forceY := 10.0, 10.0                 // Set force; you may want to calculate this based on mouse speed

	// Apply force to nearby grid points
	radius := 10 // Change as needed
	for i := max(0, gridX-radius); i < min(sim.Nx, gridX+radius); i++ {
		for j := max(0, gridY-radius); j < min(sim.Ny, gridY+radius); j++ {
			index := i*sim.Ny + j
			sim.U[index] += forceX
			sim.V[index] += forceY
		}
	}
}

func RunSimulation(seed, frameRate int64) {
	// Initialize the fluid simulation
	nx, ny := 100, 100
	dt, dx, dy, rho, nu := 0.005, 0.5, 0.5, 3.0, 0.01

	initialConditionFunc := RandomInitialCondition(seed)
	fluidSim := simulation.NewFluidSim(nx, ny, dt, dx, dy, rho, nu, initialConditionFunc)

	// Initialize the visualization
	renderer, err := viz.InitializeRenderer()
	if err != nil {
		panic(err)
	}

	var mouseX, mouseY int32
	running := true

	for running {
		// Handle SDL Events
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.MouseMotionEvent:
				mouseX, mouseY = e.X, e.Y
			}
		}

		_, _, state := sdl.GetMouseState()

		if state&sdl.BUTTON_LEFT != 0 {
			applyMouseForceToFluid(fluidSim, mouseX, mouseY)
		}

		fluidSim.Step()
		viz.RenderFrame(renderer, fluidSim.U, fluidSim.V, nx, ny)
		time.Sleep(time.Duration(1000/frameRate) * time.Millisecond)
	}
}

func main() {
	RunSimulation(time.Now().Unix(), 240)
}
