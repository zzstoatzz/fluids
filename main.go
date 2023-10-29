package main

import (
	"flag"
	"fluids/input"
	"fluids/simulation"
	"fluids/viz"
	"math/rand"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

func RunSimulation(
	seed int64,
	n int,
	dt, rho0, nu, domainX, domainY, speed_scale float64,
	frameRate int64,
	particleRadius float64,
	gravity float64,
) {
	domain := simulation.Domain{X: 100, Y: 100}

	fluidSim := simulation.NewFluidSim(n, dt, domain.X, domain.Y, rho0, nu)

	renderer, window, err := viz.NewWindow()
	if err != nil {
		panic(err)
	}

	windowWidth, windowHeight := window.GetSize()

	var mouseX, mouseY int32
	running := true

	originalGravity := gravity
	defaultGravity := -9.81 // Default gravity value

	for running {
		// handle SDL Events
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.MouseMotionEvent:
				mouseX, mouseY = e.X, e.Y
			case *sdl.KeyboardEvent:
				if e.Type == sdl.KEYDOWN {
					switch e.Keysym.Sym {
					case sdl.K_g: // 'g' key to toggle gravity
						if gravity != 0 {
							gravity = 0
						} else {
							if originalGravity == 0 {
								gravity = defaultGravity
							} else {
								gravity = originalGravity
							}
						}
					}
				}
			case *sdl.MouseButtonEvent:
				if e.Type == sdl.MOUSEBUTTONDOWN {
					if e.Button == sdl.BUTTON_LEFT {
						input.ApplyMouseForceToParticles(fluidSim, mouseX, mouseY, windowWidth, windowHeight)
					}
				}
			}
		}
		fluidSim.Step(gravity)
		viz.RenderFrame(
			renderer, fluidSim.Particles, fluidSim.Domain, windowWidth, windowHeight, speed_scale, particleRadius,
		)

		// we interpret frameRate as frames per second
		// so we need to sleep for 1/frameRate seconds
		time.Sleep(time.Duration(1e9 / frameRate))
	}
}

func main() {
	var (
		n              int
		particleRadius float64
		dt             float64
		rho0           float64
		nu             float64
		domainX        float64
		domainY        float64
		speedScale     float64
		frameRate      int64
		gravity        float64
	)

	flag.IntVar(&n, "n", 1000, "Number of particles")
	flag.Float64Var(&particleRadius, "radius", 1.0, "Particle radius")
	flag.Float64Var(&dt, "dt", 0.001, "Time step")
	flag.Float64Var(&rho0, "rho0", 1.0, "Reference density")
	flag.Float64Var(&nu, "nu", 10.0, "Viscosity")
	flag.Float64Var(&domainX, "domainX", 100.0, "Domain X size")
	flag.Float64Var(&domainY, "domainY", 100.0, "Domain Y size")
	flag.Float64Var(&domainY, "speedScale", 6.0, "Speed scale")
	flag.Int64Var(&frameRate, "fps", 120, "Frame rate")
	flag.Float64Var(&gravity, "g", -9.81, "Gravity")

	flag.Parse()

	rand.Seed(time.Now().Unix())
	RunSimulation(
		time.Now().Unix(),
		n,
		dt,
		rho0,
		nu,
		domainX,
		domainY,
		speedScale,
		frameRate,
		particleRadius,
		gravity,
	)
}
