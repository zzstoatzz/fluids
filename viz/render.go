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

	window, err := sdl.CreateWindow("Fluid Simulation", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 1200, 800, sdl.WINDOW_SHOWN)
	if err != nil {
		return nil, nil, err
	}

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		return nil, nil, err
	}

	return renderer, window, nil
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
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
	particleRadius float64,
	meanPressure float64,
	stdPressure float64,
) {
	// Clear the screen
	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.Clear()

	// Define scaling factors based on window size and domain size
	scaleX := float32(windowWidth) / float32(domain.X)
	scaleY := float32(windowHeight) / float32(domain.Y)

	// Draw particles based on fluid pressures
	for _, particle := range particles {
		// Normalize pressure using sigmoid function
		normalizedPressure := sigmoid((particle.Pressure - meanPressure) / stdPressure)

		// Lerp between blue and white based on normalized pressure
		r := uint8(255 * normalizedPressure)
		g := uint8(255 * normalizedPressure)
		b := uint8(255*(1-normalizedPressure) + normalizedPressure*255)
		renderer.SetDrawColor(r, g, b, 255)

		// Scale particle positions
		x := int32(particle.X * float64(scaleX))
		y := int32(particle.Y * float64(scaleY))

		// Draw circle with radius
		drawCircle(renderer, x, y, int32(particleRadius))
	}

	renderer.Present()
}
