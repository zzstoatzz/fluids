package main

import (
	"fluids/simulation"
	"fluids/viz"
	"math/rand"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

func applyMouseForceToParticles(sim *simulation.FluidSim, mouseX, mouseY int32) {
	forceX, forceY := 10.0, 10.0
	for i := range sim.Particles {
		// TODO: Apply the force based on mouse position
		// For now, directly modifying particle velocities
		sim.Particles[i].Vx += forceX
		sim.Particles[i].Vy += forceY
	}
}

// Define an interface for external forces that can be applied to the simulation
type ExternalForce interface {
	Apply(sim *simulation.FluidSim)
}

// MouseForce applies a force to particles based on mouse interaction
type MouseForce struct {
	X, Y  int32
	State uint32
}

func (f *MouseForce) Apply(sim *simulation.FluidSim) {
	if f.State&sdl.BUTTON_LEFT != 0 {
		applyMouseForceToParticles(sim, f.X, f.Y)
	}
}

func RunSimulation(seed, frameRate int64) {
	// Initialize the fluid simulation
	n := 1000 // Number of particles
	dt, rho0, nu := 0.001, 10.0, 1.0
	domain := simulation.Domain{X: 100, Y: 100}

	fluidSim := simulation.NewFluidSim(n, dt, domain.X, domain.Y, rho0, nu)

	// Initialize the visualization
	renderer, err := viz.InitializeRenderer()
	if err != nil {
		panic(err)
	}

	var mouseX, mouseY int32
	running := true

	// Initialize external forces
	mouseForce := &MouseForce{}

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
		mouseForce.X, mouseForce.Y, mouseForce.State = mouseX, mouseY, state

		// Apply external forces
		mouseForce.Apply(fluidSim)

		fluidSim.Step()
		viz.RenderFrame(renderer, fluidSim.Particles, fluidSim.Domain)
		time.Sleep(time.Duration(1000/frameRate) * time.Millisecond)
	}
}

func main() {
	rand.Seed(time.Now().Unix())
	RunSimulation(time.Now().Unix(), 240)
}
