package viz

import (
	"fluids/core"
	"fluids/simulation"
	"math"

	"github.com/veandco/go-sdl2/sdl"
)

func NewWindow() (*sdl.Renderer, *sdl.Window, error) {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return nil, nil, err
	}

	window, err := sdl.CreateWindow("Fluid Simulation", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 800, 600, sdl.WINDOW_SHOWN)
	if err != nil {
		return nil, nil, err
	}

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		return nil, nil, err
	}

	return renderer, window, nil
}

func drawCircle(renderer *sdl.Renderer, centerX, centerY, radius int32) {
	for theta := 0.0; theta < 2*math.Pi; theta += 0.01 {
		x := centerX + int32(math.Cos(theta)*float64(radius))
		y := centerY + int32(math.Sin(theta)*float64(radius))
		renderer.DrawPoint(x, y)
	}
}

// renders a single frame
func RenderFrame(
	renderer *sdl.Renderer,
	particles []core.Particle,
	domain simulation.Domain,
	windowWidth, windowHeight int32,
	speed_scale float64,
	particleRadius float64,
) {
	// Clear the screen
	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.Clear()

	// Define scaling factors based on window size and domain size
	scaleX := float32(windowWidth) / float32(domain.X)
	scaleY := float32(windowHeight) / float32(domain.Y)

	// Draw particles based on fluid velocities
	for _, particle := range particles {
		velocity := math.Sqrt(particle.Vx*particle.Vx + particle.Vy*particle.Vy)
		t := float32(math.Min(1, velocity*speed_scale))

		// Lerp between blue and red based on velocity
		r, g, b := uint8(0*(1-t)+255*t), uint8(0), uint8(255*(1-t)+0*t)
		renderer.SetDrawColor(r, g, b, 255)

		// Scale particle positions
		x := int32(particle.X * float64(scaleX))
		y := int32(particle.Y * float64(scaleY))

		// Draw circle with radius
		drawCircle(renderer, x, y, int32(particleRadius))
	}

	renderer.Present()
}
