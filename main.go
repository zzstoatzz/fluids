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

const DEFAULT_GRAVITY = -10000.0

func RunSimulation(
	seed int64,
	n int,
	dt, rho0, nu, domainX, domainY, pressureMultiplier float64,
	frameRate int64,
	particleRadius, gravity, mouseForce float64,
) {
	domain := simulation.Domain{X: domainX, Y: domainY}

	fluidSim := simulation.NewFluidSim(n, domain, dt, rho0, nu)

	renderer, window, err := viz.NewWindow()
	if err != nil {
		panic(err)
	}

	windowWidth, windowHeight := window.GetSize()

	var mouseX, mouseY int32
	running := true
	paused := false

	originalGravity := gravity
	defaultGravity := DEFAULT_GRAVITY // Default gravity value

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
					case sdl.K_r: // 'R' key to reset the simulation
						fluidSim = simulation.NewFluidSim(n, domain, dt, rho0, nu)
					case sdl.K_SPACE: // Space key to pause/unpause
						paused = !paused
					}
				}
			case *sdl.MouseButtonEvent:
				if e.Type == sdl.MOUSEBUTTONDOWN {
					if e.Button == sdl.BUTTON_LEFT {
						input.ApplyMouseForceToParticles(fluidSim, mouseX, mouseY, windowWidth, windowHeight, mouseForce)
					}
				}
			}
		}
		if !paused {
			meanPressure, stdPressure := fluidSim.Step(gravity, pressureMultiplier, dt)
			viz.RenderFrame(
				renderer,
				fluidSim.Particles,
				fluidSim.Domain,
				windowWidth,
				windowHeight,
				particleRadius,
				meanPressure,
				stdPressure,
			)
		}

		// we interpret frameRate as frames per second
		// so we need to sleep for 1/frameRate seconds
		time.Sleep(time.Duration(1e9 / frameRate))
	}
}

func main() {
	var (
		n                  int
		particleRadius     float64
		dt                 float64
		rho0               float64
		nu                 float64
		domainX            float64
		domainY            float64
		pressureMultiplier float64
		frameRate          int64
		gravity            float64
		mouseForce         float64
	)

	flag.IntVar(&n, "n", 500, "Number of particles")
	flag.Float64Var(&dt, "dt", 0.0005, "Time step")
	flag.Float64Var(&rho0, "rho0", 1.0, "Reference density")
	flag.Float64Var(&nu, "nu", 1.0, "Viscosity")
	flag.Float64Var(&domainX, "domainX", 100.0, "Domain X size")
	flag.Float64Var(&domainY, "domainY", 100.0, "Domain Y size")
	flag.Float64Var(&pressureMultiplier, "pressure", 10000.0, "Pressure multiplier")
	flag.Int64Var(&frameRate, "fps", 480, "Frame rate")
	flag.Float64Var(&particleRadius, "radius", 2.4, "Particle radius")
	flag.Float64Var(&gravity, "g", 0, "Gravity")
	flag.Float64Var(&mouseForce, "boom", 100.0, "Mouse force")

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
		pressureMultiplier,
		frameRate,
		particleRadius,
		gravity,
		mouseForce,
	)
}
