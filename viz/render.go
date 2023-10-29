package viz

import (
	"math"

	"github.com/veandco/go-sdl2/sdl"
)

// InitializeRenderer initializes SDL and returns a renderer for drawing
func InitializeRenderer() (*sdl.Renderer, error) {
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		return nil, err
	}

	window, err := sdl.CreateWindow("Fluid Simulation", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, 800, 600, sdl.WINDOW_SHOWN)
	if err != nil {
		return nil, err
	}

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		return nil, err
	}

	return renderer, nil
}

// Helper function to draw a circle
func drawCircle(renderer *sdl.Renderer, centerX, centerY, radius int32) {
	for w := -radius; w < radius; w++ {
		for h := -radius; h < radius; h++ {
			if w*w+h*h <= radius*radius {
				renderer.DrawPoint(centerX+w, centerY+h)
			}
		}
	}
}

// RenderFrame renders a single frame
func RenderFrame(renderer *sdl.Renderer, U []float64, V []float64, nx int, ny int) {
	// Clear the screen
	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.Clear()

	// Draw bounding box
	renderer.SetDrawColor(255, 255, 255, 255)
	renderer.DrawRect(&sdl.Rect{X: 0, Y: 0, W: 800, H: 600})

	// Draw particles based on fluid velocities
	for i := 0; i < nx; i++ {
		for j := 0; j < ny; j++ {
			index := i*ny + j
			velocity := math.Sqrt(U[index]*U[index] + V[index]*V[index])
			t := math.Min(1, velocity/3.0) // Adjust this value based on your max velocity

			// Lerp between blue and red based on velocity
			r, g, b := uint8(0*(1-t)+255*t), uint8(0), uint8(255*(1-t)+0*t)
			renderer.SetDrawColor(r, g, b, 255)

			x := int32(i * 20)
			y := int32(600 - int(V[index]*500))
			drawCircle(renderer, x, y, 2) // Draw circle with radius 5
		}
	}

	renderer.Present()
}
