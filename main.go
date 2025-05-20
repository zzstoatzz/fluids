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
const DEFAULT_GRAVITY_TOGGLE_VALUE = 100000.0 // Value when gravity is toggled on from zero state

func RunSimulation(
	seed int64,
	numParticles int,
	domainX, domainY float64, // Domain dimensions
	particleVisRadius float64, // Visual radius of particles
	frameRate int64,
	numWorkers int,
	// Simulation parameters, now passed as a struct
	simParams simulation.SimParameters,
	rng *rand.Rand, // Add random number generator
) {
	domain := simulation.Domain{X: domainX, Y: domainY}

	// Create simulation using the passed SimParameters struct and rng
	fluidSim := simulation.NewFluidSim(numParticles, domain, simParams, rng)

	// These were direct fields, now they are part of simParams used in NewFluidSim
	// fluidSim.SmoothingFactor = simParams.SmoothingFactor
	// fluidSim.DampeningFactor = simParams.DampeningFactor

	if numWorkers > 0 {
		simulation.SetParallelConfig(simulation.ParallelConfig{
			NumWorkers:       numWorkers,
			MinimumBatchSize: 32,
		})
	}

	renderer, window, err := viz.NewWindow()
	if err != nil {
		panic(err)
	}
	defer viz.CleanupFonts()
	windowWidth, windowHeight := window.GetSize()

	var mouseX, mouseY int32
	running := true
	paused := false
	showDebug := false
	var mouseEffects []viz.MouseEffect
	const MAX_MOUSE_EFFECTS = 10
	blueRipple := sdl.Color{R: 50, G: 150, B: 255, A: 128}

	// Operational parameters that can change during simulation loop
	currentGravity := simParams.Gravity
	currentPressureMultiplier := simParams.PressureMultiplier
	currentMouseForce := simParams.MouseForce
	currentMouseForceRadius := simParams.MouseForceRadius // New operational variable
	// Dt from simParams is the base physics timestep, fluidSim.Dt holds this.
	// The dt passed to fluidSim.Step is currentDt (which is simParams.Dt initially)

	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.MouseMotionEvent:
				mouseX, mouseY = e.X, e.Y
			case *sdl.KeyboardEvent:
				if e.Type == sdl.KEYDOWN {
					switch e.Keysym.Sym {
					case sdl.K_g:
						if currentGravity != 0 {
							currentGravity = 0
						} else {
							// If original simParam gravity was 0, use a default toggle-on value
							if simParams.Gravity == 0 {
								currentGravity = DEFAULT_GRAVITY_TOGGLE_VALUE
							} else {
								currentGravity = simParams.Gravity // Restore to original config gravity
							}
						}
					case sdl.K_r:
						fluidSim = simulation.NewFluidSim(numParticles, domain, simParams, rng) // Re-init with original params and rng
					case sdl.K_SPACE:
						paused = !paused
					case sdl.K_d:
						showDebug = !showDebug
					case sdl.K_b:
						input.ApplyMouseForceToParticles(fluidSim, mouseX, mouseY, windowWidth, windowHeight, currentMouseForce*5, currentMouseForceRadius)
					case sdl.K_a:
						currentPressureMultiplier = math.Max(5000, currentPressureMultiplier-1000)
						fmt.Printf("Pressure: %.0f\n", currentPressureMultiplier)
					case sdl.K_s:
						currentPressureMultiplier = math.Min(50000, currentPressureMultiplier+1000)
						fmt.Printf("Pressure: %.0f\n", currentPressureMultiplier)
					case sdl.K_z:
						fluidSim.DampeningFactor = math.Max(0.01, fluidSim.DampeningFactor-0.01)
						fmt.Printf("Drag: %.2f\n", fluidSim.DampeningFactor)
					case sdl.K_x:
						fluidSim.DampeningFactor = math.Min(0.3, fluidSim.DampeningFactor+0.01)
						fmt.Printf("Drag: %.2f\n", fluidSim.DampeningFactor)
					case sdl.K_c:
						fluidSim.SmoothingFactor = math.Max(0.05, fluidSim.SmoothingFactor-0.01)
						fmt.Printf("Smoothing: %.2f\n", fluidSim.SmoothingFactor)
					case sdl.K_v:
						fluidSim.SmoothingFactor = math.Min(0.4, fluidSim.SmoothingFactor+0.01)
						fmt.Printf("Smoothing: %.2f\n", fluidSim.SmoothingFactor)
					case sdl.K_LEFTBRACKET:
						currentMouseForce = math.Max(50, currentMouseForce-50)
						fmt.Printf("Mouse Force: %.0f\n", currentMouseForce)
					case sdl.K_RIGHTBRACKET:
						currentMouseForce = math.Min(2000, currentMouseForce+50)
						fmt.Printf("Mouse Force: %.0f\n", currentMouseForce)
					case sdl.K_q:
						fluidSim.AttractionFactor -= 10
						fmt.Printf("Attraction: %.0f\n", fluidSim.AttractionFactor)
					case sdl.K_w:
						fluidSim.AttractionFactor += 10
						fmt.Printf("Attraction: %.0f\n", fluidSim.AttractionFactor)
					case sdl.K_e:
						fluidSim.InteractionRadius = math.Max(1.0, fluidSim.InteractionRadius-1.0)
						fmt.Printf("Interaction Radius: %.1f\n", fluidSim.InteractionRadius)
					case sdl.K_t:
						fluidSim.InteractionRadius = math.Min(50.0, fluidSim.InteractionRadius+1.0)
						fmt.Printf("Interaction Radius: %.1f\n", fluidSim.InteractionRadius)
					case sdl.K_COMMA: // '<' key
						currentMouseForceRadius = math.Max(5.0, currentMouseForceRadius-5.0)
						fmt.Printf("Mouse Force Radius: %.1f\n", currentMouseForceRadius)
					case sdl.K_PERIOD: // '>' key
						currentMouseForceRadius = math.Min(200.0, currentMouseForceRadius+5.0)
						fmt.Printf("Mouse Force Radius: %.1f\n", currentMouseForceRadius)
					}
				}
			case *sdl.MouseButtonEvent:
				if e.Type == sdl.MOUSEBUTTONDOWN && e.Button == sdl.BUTTON_LEFT {
					input.ApplyMouseForceToParticles(fluidSim, mouseX, mouseY, windowWidth, windowHeight, currentMouseForce, currentMouseForceRadius)
					if len(mouseEffects) < MAX_MOUSE_EFFECTS {
						// Use currentMouseForceRadius for the visual effect's MaxRadius
						mouseEffects = append(mouseEffects, viz.MouseEffect{
							X: mouseX, Y: mouseY, MaxRadius: currentMouseForceRadius,
							StartTime: uint32(sdl.GetTicks64()), Duration: 500, Color: blueRipple,
						})
					}
				}
			}
		}

		if !paused {
			// Step simulation using current operational values and the base Dt from fluidSim
			meanPressure, stdPressure := fluidSim.Step(currentGravity, currentPressureMultiplier, fluidSim.Dt)

			currentTime := uint32(sdl.GetTicks64())
			i := 0
			for _, effect := range mouseEffects {
				if currentTime-effect.StartTime < effect.Duration {
					mouseEffects[i] = effect
					i++
				}
			}
			mouseEffects = mouseEffects[:i]

			settings := viz.SimSettings{
				Gravity:           currentGravity,
				Pressure:          currentPressureMultiplier,
				Drag:              fluidSim.DampeningFactor,
				Smoothing:         fluidSim.SmoothingFactor,
				Attraction:        fluidSim.AttractionFactor,
				InteractionRadius: fluidSim.InteractionRadius,
				ParticleCount:     len(fluidSim.Particles),
				MouseForce:        currentMouseForce,
				MouseForceRadius:  currentMouseForceRadius, // Pass to viz settings
			}
			viz.RenderFrame(renderer, fluidSim.Particles, fluidSim.Domain, windowWidth, windowHeight,
				particleVisRadius, meanPressure, stdPressure, showDebug, mouseEffects, currentTime, settings)
		}
		time.Sleep(time.Duration(1e9 / frameRate))
	}
}

func main() {
	// Start pprof server for profiling
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	defaultParams := simulation.GetDefaultSimParameters()

	var (
		nParticles         int
		particleVisRadius  float64 // Visual radius, distinct from interaction radius
		dt                 float64
		rho0               float64
		nu                 float64
		domainX            float64
		domainY            float64
		pressureMultiplier float64
		frameRate          int64
		gravity            float64
		mouseForce         float64
		mouseForceRadius   float64 // New flag variable
		smoothingFactor    float64
		dampeningFactor    float64
		numWorkers         int
		seed               int64
	)

	flag.IntVar(&nParticles, "n", 1000, "Number of particles")
	flag.Float64Var(&particleVisRadius, "radius", 2.4, "Visual radius of particles (distinct from interaction radius)")
	flag.Int64Var(&frameRate, "fps", 120, "Target frame rate")
	flag.Float64Var(&domainX, "domainX", 1024, "Width of the simulation domain")
	flag.Float64Var(&domainY, "domainY", 768, "Height of the simulation domain")
	flag.IntVar(&numWorkers, "workers", runtime.NumCPU(), "Number of worker threads (0 = auto-detected by Go runtime for GOMAXPROCS)")
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "Random seed for simulation initialization")

	// Flags using defaults from simulation.GetDefaultSimParameters()
	flag.Float64Var(&dt, "dt", defaultParams.Dt, "Physics time step")
	flag.Float64Var(&rho0, "rho0", defaultParams.Rho0, "Reference density")
	flag.Float64Var(&nu, "nu", defaultParams.Nu, "Viscosity of the fluid")
	flag.Float64Var(&pressureMultiplier, "pressure", defaultParams.PressureMultiplier, "Pressure multiplier affecting fluid stiffness")
	flag.Float64Var(&gravity, "g", defaultParams.Gravity, "Gravity strength (0 = disabled, >0 = downward)")
	flag.Float64Var(&mouseForce, "boom", defaultParams.MouseForce, "Magnitude of mouse explosion force")
	flag.Float64Var(&mouseForceRadius, "mouseRadius", defaultParams.MouseForceRadius, "Radius of mouse explosion force") // New flag
	flag.Float64Var(&smoothingFactor, "smooth", defaultParams.SmoothingFactor, "Smoothing factor for forces")
	flag.Float64Var(&dampeningFactor, "drag", defaultParams.DampeningFactor, "Drag/dampening factor for particle velocity")

	flag.Parse()
	// rand.Seed is deprecated, use rand.New(rand.NewSource(seed)) instead
	rng := rand.New(rand.NewSource(seed))

	// Construct SimParameters from parsed flags to pass to RunSimulation
	simParams := simulation.SimParameters{
		InteractionRadius:  defaultParams.InteractionRadius, // Uses default from config, not a flag for now
		SmoothingFactor:    smoothingFactor,
		DampeningFactor:    dampeningFactor,
		DragEnabled:        true,                           // Defaulting DragEnabled to true, can be made a flag if needed
		AttractionFactor:   defaultParams.AttractionFactor, // Uses default from config, not a flag for now
		Rho0:               rho0,
		Nu:                 nu,
		PressureMultiplier: pressureMultiplier,
		Dt:                 dt,
		Gravity:            gravity,
		MouseForce:         mouseForce,
		MouseForceRadius:   mouseForceRadius, // Set from flag
	}

	RunSimulation(
		seed,
		nParticles,
		domainX, domainY,
		particleVisRadius,
		frameRate,
		numWorkers,
		simParams,
		rng, // Pass the new random number generator
	)
}
