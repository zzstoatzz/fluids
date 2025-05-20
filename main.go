package main

import (
	"flag"
	"fluids/input"
	"fluids/simulation"
	"fluids/viz"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	_ "net/http/pprof" // Import for pprof side effects
	"runtime"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

// Default gravity is positive (downward force)
const DEFAULT_GRAVITY = 100000.0

func RunSimulation(
	seed int64,
	n int,
	dt, rho0, nu, domainX, domainY, pressureMultiplier float64,
	frameRate int64,
	particleRadius, gravity, mouseForce float64,
	smoothingFactor, dampeningFactor float64,
	numWorkers int,
) {
	domain := simulation.Domain{X: domainX, Y: domainY}

	// Create simulation
	fluidSim := simulation.NewFluidSim(n, domain, dt, rho0, nu)

	// Set performance parameters
	fluidSim.SmoothingFactor = smoothingFactor
	fluidSim.DampeningFactor = dampeningFactor

	// Set parallel execution configuration if specified
	if numWorkers > 0 {
		simulation.SetParallelConfig(simulation.ParallelConfig{
			NumWorkers:       numWorkers,
			MinimumBatchSize: 32,
		})
	}

	// Create renderer window
	renderer, window, err := viz.NewWindow()
	if err != nil {
		panic(err)
	}

	// Make sure we clean up fonts when done
	defer viz.CleanupFonts()

	// Get window dimensions for mouse input
	windowWidth, windowHeight := window.GetSize()

	// Simulation control variables
	var mouseX, mouseY int32
	running := true
	paused := false
	showDebug := false

	// Mouse effect variables
	var mouseEffects []viz.MouseEffect
	const MAX_MOUSE_EFFECTS = 10

	// Create colors for mouse effects
	blueRipple := sdl.Color{R: 50, G: 150, B: 255, A: 128}

	// Store original gravity for toggling
	originalGravity := gravity
	defaultGravity := DEFAULT_GRAVITY // Default gravity value

	// Main simulation loop
	for running {
		// Handle SDL Events
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
						fluidSim.SmoothingFactor = smoothingFactor
						fluidSim.DampeningFactor = dampeningFactor
					case sdl.K_SPACE: // Space key to pause/unpause
						paused = !paused
					case sdl.K_d: // 'd' key to toggle debug visualization
						showDebug = !showDebug
					case sdl.K_b: // 'b' key for a bigger explosion effect
						input.ApplyMouseForceToParticles(fluidSim, mouseX, mouseY, windowWidth, windowHeight, mouseForce*5)
					// Interactive controls for parameters
					case sdl.K_a: // 'a' key to decrease attraction/pressure
						pressureMultiplier = math.Max(5000, pressureMultiplier-1000)
						fmt.Printf("Pressure: %.0f\n", pressureMultiplier)
					case sdl.K_s: // 's' key to increase attraction/pressure
						pressureMultiplier = math.Min(50000, pressureMultiplier+1000)
						fmt.Printf("Pressure: %.0f\n", pressureMultiplier)
					case sdl.K_z: // 'z' key to decrease drag
						fluidSim.DampeningFactor = math.Max(0.01, fluidSim.DampeningFactor-0.01)
						fmt.Printf("Drag: %.2f\n", fluidSim.DampeningFactor)
					case sdl.K_x: // 'x' key to increase drag
						fluidSim.DampeningFactor = math.Min(0.3, fluidSim.DampeningFactor+0.01)
						fmt.Printf("Drag: %.2f\n", fluidSim.DampeningFactor)
					case sdl.K_c: // 'c' key to decrease smoothing factor
						fluidSim.SmoothingFactor = math.Max(0.05, fluidSim.SmoothingFactor-0.01)
						fmt.Printf("Smoothing: %.2f\n", fluidSim.SmoothingFactor)
					case sdl.K_v: // 'v' key to increase smoothing factor
						fluidSim.SmoothingFactor = math.Min(0.4, fluidSim.SmoothingFactor+0.01)
						fmt.Printf("Smoothing: %.2f\n", fluidSim.SmoothingFactor)
					case sdl.K_LEFTBRACKET: // '[' key to decrease mouse force
						mouseForce = math.Max(50, mouseForce-50)
						fmt.Printf("Mouse Force: %.0f\n", mouseForce)
					case sdl.K_RIGHTBRACKET: // ']' key to increase mouse force
						mouseForce = math.Min(2000, mouseForce+50)
						fmt.Printf("Mouse Force: %.0f\n", mouseForce)
					case sdl.K_q: // 'q' key to decrease attraction/make more repulsive
						fluidSim.AttractionFactor -= 10
						fmt.Printf("Attraction: %.0f\n", fluidSim.AttractionFactor)
					case sdl.K_w: // 'w' key to increase attraction/make more attractive
						fluidSim.AttractionFactor += 10
						fmt.Printf("Attraction: %.0f\n", fluidSim.AttractionFactor)
					case sdl.K_e: // 'e' key to decrease interaction radius
						fluidSim.InteractionRadius = math.Max(1.0, fluidSim.InteractionRadius-1.0)
						fmt.Printf("Interaction Radius: %.1f\n", fluidSim.InteractionRadius)
					case sdl.K_t: // 't' key to increase interaction radius
						fluidSim.InteractionRadius = math.Min(50.0, fluidSim.InteractionRadius+1.0)
						fmt.Printf("Interaction Radius: %.1f\n", fluidSim.InteractionRadius)
					}
				}
			case *sdl.MouseButtonEvent:
				if e.Type == sdl.MOUSEBUTTONDOWN {
					if e.Button == sdl.BUTTON_LEFT {
						// Apply force to particles
						input.ApplyMouseForceToParticles(fluidSim, mouseX, mouseY, windowWidth, windowHeight, mouseForce)

						// Add visual ripple effect at click location
						if len(mouseEffects) < MAX_MOUSE_EFFECTS {
							// Calculate radius based on the mouseForce
							effectRadius := float64(mouseForce) / 10.0
							if effectRadius < 20 {
								effectRadius = 20
							} else if effectRadius > 100 {
								effectRadius = 100
							}

							// Create new effect
							newEffect := viz.MouseEffect{
								X:         mouseX,
								Y:         mouseY,
								MaxRadius: effectRadius,
								StartTime: uint32(sdl.GetTicks64()),
								Duration:  500, // Effect lasts 500ms
								Color:     blueRipple,
							}

							// Add to effects list
							mouseEffects = append(mouseEffects, newEffect)
						}
					}
				}
			}
		}

		if !paused {
			// Step simulation forward
			meanPressure, stdPressure := fluidSim.Step(gravity, pressureMultiplier, dt)

			// Update mouse effects - remove expired ones
			currentTime := uint32(sdl.GetTicks64())
			i := 0
			for _, effect := range mouseEffects {
				if currentTime-effect.StartTime < effect.Duration {
					mouseEffects[i] = effect
					i++
				}
			}
			mouseEffects = mouseEffects[:i]

			// Create settings for display
			settings := viz.SimSettings{
				Gravity:           gravity,
				Pressure:          pressureMultiplier,
				Drag:              fluidSim.DampeningFactor,
				Smoothing:         fluidSim.SmoothingFactor,
				Attraction:        fluidSim.AttractionFactor,
				InteractionRadius: fluidSim.InteractionRadius,
				ParticleCount:     len(fluidSim.Particles),
				MouseForce:        mouseForce,
			}

			// Render the updated particles
			viz.RenderFrame(
				renderer,
				fluidSim.Particles,
				fluidSim.Domain,
				windowWidth,
				windowHeight,
				particleRadius,
				meanPressure,
				stdPressure,
				showDebug,
				mouseEffects,
				currentTime,
				settings,
			)
		}

		// Limit frame rate
		// we interpret frameRate as frames per second
		// so we need to sleep for 1/frameRate seconds
		time.Sleep(time.Duration(1e9 / frameRate))
	}
}

func main() {
	// Start pprof server for profiling
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	var (
		// Basic simulation parameters
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

		// Performance parameters
		smoothingFactor float64
		dampeningFactor float64
		numWorkers      int
	)

	// Basic parameters
	flag.IntVar(&n, "n", 1000, "Number of particles")
	flag.Float64Var(&dt, "dt", 0.0008, "Time step (0.0001-0.001, higher = faster but less stable)")
	flag.Float64Var(&rho0, "rho0", 1.0, "Reference density")
	flag.Float64Var(&nu, "nu", 0.5, "Viscosity (0.1-5.0, higher = more viscous fluid)")
	flag.Float64Var(&domainX, "domainX", 100.0, "Domain X size")
	flag.Float64Var(&domainY, "domainY", 100.0, "Domain Y size")
	flag.Float64Var(&pressureMultiplier, "pressure", 20000.0, "Pressure multiplier (5000-30000)")
	flag.Int64Var(&frameRate, "fps", 120, "Frame rate cap (60-240, limited by hardware)")
	flag.Float64Var(&particleRadius, "radius", 2.4, "Particle radius (1.0-5.0)")
	flag.Float64Var(&gravity, "g", 0, "Gravity (0-200, 0 = disabled)")
	flag.Float64Var(&mouseForce, "boom", 500.0, "Mouse force multiplier (100-1000)")

	// Performance parameters
	flag.Float64Var(&smoothingFactor, "smooth", 0.15, "Smoothing factor (0.1-0.3, smaller = more accurate but can be unstable)")
	flag.Float64Var(&dampeningFactor, "drag", 0.1, "Drag/dampening factor (0.0-0.2, higher = more drag)")
	flag.IntVar(&numWorkers, "workers", runtime.NumCPU(), "Number of worker threads (0 = auto)")

	flag.Parse()

	// Seed random number generator
	rand.Seed(time.Now().Unix())

	// Run the simulation
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
		smoothingFactor,
		dampeningFactor,
		numWorkers,
	)
}
