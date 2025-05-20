package viz

import (
	"fluids/core"
	"fluids/simulation"
	"fmt"
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

	// Initialize font settings
	fontCache.initialized = true
	fontCache.fontSize = 14

	return renderer, window, nil
}

// CleanupFonts resets font cache settings
func CleanupFonts() {
	fontCache.initialized = false
}

func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// Fast circle drawing functions using different techniques

// Draws a filled circle more efficiently than point-by-point
func drawFilledCircle(renderer *sdl.Renderer, centerX, centerY, radius int32) {
	// For tiny circles, use a single point
	if radius <= 0 {
		renderer.DrawPoint(centerX, centerY)
		return
	}

	// For small circles, use a better circle approximation
	// This creates rounded corners by using overlapping points
	if radius <= 2 {
		// Draw a plus sign
		renderer.DrawLine(centerX-radius, centerY, centerX+radius, centerY)
		renderer.DrawLine(centerX, centerY-radius, centerX, centerY+radius)

		// Draw diagonals for better circle approximation
		renderer.DrawPoint(centerX-radius+1, centerY-radius+1) // top-left
		renderer.DrawPoint(centerX+radius-1, centerY-radius+1) // top-right
		renderer.DrawPoint(centerX-radius+1, centerY+radius-1) // bottom-left
		renderer.DrawPoint(centerX+radius-1, centerY+radius-1) // bottom-right
		return
	}

	// For medium and larger circles, use horizontal scanlines for filling
	radiusSq := radius * radius

	// Draw horizontal scanlines with proper anti-aliasing logic
	for y := -radius; y <= radius; y++ {
		// Calculate width at this y position using circle equation
		width := int32(math.Sqrt(float64(radiusSq - y*y)))

		// Draw horizontal line
		renderer.DrawLine(
			centerX-width, centerY+y,
			centerX+width, centerY+y,
		)
	}

	// Add an optional outline for better definition
	// This gives a more circular appearance
	drawCircleOutline(renderer, centerX, centerY, radius)
}

// Fast circle outline drawing
func drawCircleOutline(renderer *sdl.Renderer, centerX, centerY, radius int32) {
	// Use Bresenham algorithm for circle drawing
	x := radius
	y := int32(0)
	err := int32(0)

	for x >= y {
		// Draw 8 points of the circle using symmetry
		renderer.DrawPoint(centerX+x, centerY+y)
		renderer.DrawPoint(centerX+y, centerY+x)
		renderer.DrawPoint(centerX-y, centerY+x)
		renderer.DrawPoint(centerX-x, centerY+y)
		renderer.DrawPoint(centerX-x, centerY-y)
		renderer.DrawPoint(centerX-y, centerY-x)
		renderer.DrawPoint(centerX+y, centerY-x)
		renderer.DrawPoint(centerX+x, centerY-y)

		// Update position
		y++
		if err <= 0 {
			err += 2*y + 1
		}
		if err > 0 {
			x--
			err -= 2*x + 1
		}
	}
}

// Draw circle with optimization for different sizes
func drawCircle(renderer *sdl.Renderer, centerX, centerY, radius int32) {
	// Handle tiny circles (single pixel)
	if radius <= 0 {
		renderer.DrawPoint(centerX, centerY)
		return
	}

	// For radius 1, draw a small diamond shape (more circular than a plus)
	if radius == 1 {
		// Draw center point
		renderer.DrawPoint(centerX, centerY)
		// Draw the diamond pattern
		renderer.DrawPoint(centerX+1, centerY)
		renderer.DrawPoint(centerX-1, centerY)
		renderer.DrawPoint(centerX, centerY+1)
		renderer.DrawPoint(centerX, centerY-1)
		// Add corner pixels for a more circular appearance
		renderer.DrawPoint(centerX+1, centerY+1)
		renderer.DrawPoint(centerX-1, centerY+1)
		renderer.DrawPoint(centerX+1, centerY-1)
		renderer.DrawPoint(centerX-1, centerY-1)
		return
	}

	// For small radius, use scanline filled circle - no outlines
	if radius <= 2 {
		// For small circles, scanlines alone look better than the plus-sign approach
		radiusSq := radius * radius
		for y := -radius; y <= radius; y++ {
			// Calculate width at this y position using circle equation
			width := int32(math.Sqrt(float64(radiusSq - y*y)))
			renderer.DrawLine(centerX-width, centerY+y, centerX+width, centerY+y)
		}
		return
	}

	// For medium to large circles, use horizontal scanlines
	radiusSq := radius * radius
	for y := -radius; y <= radius; y++ {
		// Calculate width at this y position using circle equation
		width := int32(math.Sqrt(float64(radiusSq - y*y)))
		renderer.DrawLine(centerX-width, centerY+y, centerX+width, centerY+y)
	}
}

// renderGrid draws a visual representation of the spatial grid
func renderGrid(
	renderer *sdl.Renderer,
	cellSize float64,
	domain simulation.Domain,
	windowWidth, windowHeight int32,
) {
	// Define scaling factors based on window size and domain size
	scaleX := float32(windowWidth) / float32(domain.X)
	scaleY := float32(windowHeight) / float32(domain.Y)

	// Set grid line color (semitransparent)
	renderer.SetDrawColor(50, 50, 150, 100)

	// Draw vertical grid lines
	for x := float64(0); x <= domain.X; x += cellSize {
		x1 := int32(x * float64(scaleX))
		renderer.DrawLine(x1, 0, x1, windowHeight)
	}

	// Draw horizontal grid lines
	for y := float64(0); y <= domain.Y; y += cellSize {
		y1 := int32(y * float64(scaleY))
		renderer.DrawLine(0, y1, windowWidth, y1)
	}
}

// renderNeighbors draws connections between particles and their neighbors
func renderNeighbors(
	renderer *sdl.Renderer,
	particles []core.Particle,
	domain simulation.Domain,
	windowWidth, windowHeight int32,
) {
	// Define scaling factors based on window size and domain size
	scaleX := float32(windowWidth) / float32(domain.X)
	scaleY := float32(windowHeight) / float32(domain.Y)

	// Set connection color (semitransparent)
	renderer.SetDrawColor(50, 200, 100, 30)

	// Draw connections for each particle
	for i, particle := range particles {
		// Scale particle position
		x1 := int32(particle.X * float64(scaleX))
		y1 := int32(particle.Y * float64(scaleY))

		// Draw connection lines to neighbors
		for _, neighborIdx := range particle.NeighborIndices {
			// Only draw each connection once (when i < neighborIdx)
			if i >= neighborIdx {
				continue
			}

			neighbor := &particles[neighborIdx]

			// Scale neighbor position
			x2 := int32(neighbor.X * float64(scaleX))
			y2 := int32(neighbor.Y * float64(scaleY))

			// Draw line connecting particles
			renderer.DrawLine(x1, y1, x2, y2)
		}
	}
}

// MouseEffect represents a visual ripple effect when clicking
type MouseEffect struct {
	X, Y      int32     // Position in screen coordinates
	Radius    float64   // Current radius
	MaxRadius float64   // Maximum size
	StartTime uint32    // When the effect started (using SDL ticks)
	Duration  uint32    // How long the effect lasts in milliseconds
	Color     sdl.Color // Effect color
}

// renderMouseEffects draws ripple effects when clicking
func renderMouseEffects(
	renderer *sdl.Renderer,
	effects []MouseEffect,
	currentTime uint32,
) {
	// Store original blend mode and color
	oldR, oldG, oldB, oldA, _ := renderer.GetDrawColor()

	// Set blend mode for transparency
	renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

	// Process each effect
	for i := len(effects) - 1; i >= 0; i-- {
		effect := &effects[i]

		// Calculate age and progress
		age := currentTime - effect.StartTime
		if age > effect.Duration {
			// Effect expired, remove from list
			continue
		}

		// Calculate progress ratio
		progress := float64(age) / float64(effect.Duration)

		// Ease out function for smoother animation
		easeOutQuad := 1.0 - (1.0-progress)*(1.0-progress)

		// Calculate current properties
		currentOpacity := uint8(float64(effect.Color.A) * (1.0 - progress))
		currentRadius := int32(effect.MaxRadius * easeOutQuad)

		// Set the color with fading opacity
		renderer.SetDrawColor(effect.Color.R, effect.Color.G, effect.Color.B, currentOpacity)

		// Draw expanding circle
		for r := int32(0); r <= currentRadius; r += 2 {
			// Calculate fading opacity based on radius
			fadingOpacity := uint8(float64(currentOpacity) * (1.0 - float64(r)/float64(currentRadius)))
			renderer.SetDrawColor(effect.Color.R, effect.Color.G, effect.Color.B, fadingOpacity)

			// Draw circle outline
			drawCircleOutline(renderer, effect.X, effect.Y, r)
		}
	}

	// Restore original color and blend mode
	renderer.SetDrawColor(oldR, oldG, oldB, oldA)
	renderer.SetDrawBlendMode(sdl.BLENDMODE_NONE)
}

// renderVelocities draws velocity vectors for each particle
func renderVelocities(
	renderer *sdl.Renderer,
	particles []core.Particle,
	domain simulation.Domain,
	windowWidth, windowHeight int32,
) {
	// Define scaling factors based on window size and domain size
	scaleX := float32(windowWidth) / float32(domain.X)
	scaleY := float32(windowHeight) / float32(domain.Y)

	// Scale factor for velocity vectors
	const velocityScale = 3.0

	// Set velocity vector color
	renderer.SetDrawColor(255, 50, 50, 200)

	// Draw velocity vector for each particle
	for _, particle := range particles {
		// Skip particles with negligible velocity
		speed := math.Sqrt(particle.Vx*particle.Vx + particle.Vy*particle.Vy)
		if speed < 0.1 {
			continue
		}

		// Scale particle position
		x1 := int32(particle.X * float64(scaleX))
		y1 := int32(particle.Y * float64(scaleY))

		// Scale velocity vector end point
		x2 := int32((particle.X + particle.Vx*velocityScale) * float64(scaleX))
		y2 := int32((particle.Y + particle.Vy*velocityScale) * float64(scaleY))

		// Draw velocity vector
		renderer.DrawLine(x1, y1, x2, y2)
	}
}

// renderCell visualizes which spatial grid cell each particle belongs to
func renderCells(
	renderer *sdl.Renderer,
	particles []core.Particle,
	cellSize float64,
	domain simulation.Domain,
	windowWidth, windowHeight int32,
) {
	// Define scaling factors based on window size and domain size
	scaleX := float32(windowWidth) / float32(domain.X)
	scaleY := float32(windowHeight) / float32(domain.Y)

	// Track which cells we've already colored
	coloredCells := make(map[string]bool)

	// Draw colored rectangles for each occupied cell
	for _, particle := range particles {
		// Get cell coordinates
		cellX := particle.CellX
		cellY := particle.CellY

		// Create cell key for map
		cellKey := string(cellX) + ":" + string(cellY)

		// Skip already colored cells
		if coloredCells[cellKey] {
			continue
		}

		// Mark this cell as colored
		coloredCells[cellKey] = true

		// Calculate cell bounds in screen space
		x1 := int32(float64(cellX) * cellSize * float64(scaleX))
		y1 := int32(float64(cellY) * cellSize * float64(scaleY))
		width := int32(cellSize * float64(scaleX))
		height := int32(cellSize * float64(scaleY))

		// Create semitransparent cell rectangle
		// Use deterministic color based on cell coordinates for visualization
		r := uint8((cellX * 123) % 200)
		g := uint8((cellY * 45) % 200)
		b := uint8(((cellX + cellY) * 67) % 200)
		renderer.SetDrawColor(r, g, b, 30)

		// Draw filled rectangle for cell
		rect := sdl.Rect{X: x1, Y: y1, W: width, H: height}
		renderer.FillRect(&rect)
	}
}

// Struct to hold precomputed color information for rendering
type ColorCache struct {
	r, g, b uint8
}

// Global cache for pressure-based colors to avoid recalculating every frame
var colorCache [256]ColorCache

// Initialize color cache for faster rendering with a smoother gradient
func initColorCache() {
	for i := 0; i < 256; i++ {
		normalizedPressure := float64(i) / 255.0

		// Create a smooth gradient from blue to cyan to white
		// Lower pressure (0.0): Deep blue
		// Medium pressure (0.5): Cyan/teal
		// Higher pressure (1.0): White with blue tint

		var r, g, b uint8

		if normalizedPressure < 0.5 {
			// Blue to cyan gradient (first half)
			// Smoothly increase green while keeping blue high
			t := normalizedPressure * 2.0 // Scale 0-0.5 to 0-1
			r = uint8(10 + 70*t)          // Small increase in red for richness
			g = uint8(120 * t)            // Green increases from 0 to 120
			b = uint8(180 + 50*t)         // Blue stays high but increases slightly
		} else {
			// Cyan to white gradient (second half)
			// Smoothly increase red while keeping blue and green high
			t := (normalizedPressure - 0.5) * 2.0 // Scale 0.5-1.0 to 0-1
			r = uint8(80 + 175*t)                 // Red increases from 80 to 255
			g = uint8(120 + 135*t)                // Green increases from 120 to 255
			b = uint8(230 + 25*t)                 // Blue stays very high
		}

		colorCache[i] = ColorCache{r, g, b}
	}
}

// Struct for batching rendering of same-colored particles
type ParticleBatch struct {
	color  ColorCache
	points []sdl.Point
}

// Get color for a given normalized pressure (0-1)
func getColor(normalizedPressure float64) ColorCache {
	// Clamp to 0-1 range
	if normalizedPressure < 0 {
		normalizedPressure = 0
	} else if normalizedPressure > 1 {
		normalizedPressure = 1
	}

	// Use precomputed color
	idx := int(normalizedPressure * 255)
	return colorCache[idx]
}

// Settings holds the current simulation parameters for display
type SimSettings struct {
	Gravity           float64
	Pressure          float64
	Drag              float64
	Smoothing         float64
	Attraction        float64 // N-body gravity-like attraction, -ve = repulsion
	InteractionRadius float64 // Max distance for inter-particle attraction
	ParticleCount     int
	MouseForce        float64
	MouseForceRadius  float64 // Add MouseForceRadius
}

// renders a single frame with optimizations for smoothness
func RenderFrame(
	renderer *sdl.Renderer,
	particles []core.Particle,
	domain simulation.Domain,
	windowWidth, windowHeight int32,
	particleRadius float64,
	meanPressure float64,
	stdPressure float64,
	showDebug bool,
	mouseEffects []MouseEffect,
	currentTime uint32,
	settings SimSettings,
) {
	// Initialize color cache on first render
	static := struct {
		initialized bool
	}{}
	if !static.initialized {
		initColorCache()
		static.initialized = true
	}

	// Clear the screen
	renderer.SetDrawColor(0, 0, 0, 255)
	renderer.Clear()

	// Define scaling factors based on window size and domain size
	scaleX := float32(windowWidth) / float32(domain.X)
	scaleY := float32(windowHeight) / float32(domain.Y)

	// Render debug visualizations first (underneath particles)
	if showDebug {
		// Visualization of spatial grid cells
		renderCells(renderer, particles, 4.0, domain, windowWidth, windowHeight)

		// Visualization of spatial grid
		renderGrid(renderer, 4.0, domain, windowWidth, windowHeight)

		// Visualization of particle neighbors
		renderNeighbors(renderer, particles, domain, windowWidth, windowHeight)
	}

	// Use batched drawing for particles to minimize rendering calls
	// We'll group particles by color to minimize renderer.SetDrawColor calls
	colorBatches := make(map[ColorCache]*ParticleBatch)

	// Calculate pressure range for a more consistent color mapping
	// Use a combination of mean + stddev with a fixed baseline pressure range
	// This makes the color distribution more consistent between frames
	pressureOffset := meanPressure
	pressureScale := math.Max(stdPressure, 1.0) * 3.0 // 3 standard deviations covers most values

	// Apply a minimum pressure range to avoid extreme color changes
	if pressureScale < 1000.0 {
		pressureScale = 1000.0
	}

	// Process particles into batches by color
	for _, particle := range particles {
		// Smooth normalization for pressure mapping to colors
		// This maps pressure to 0.0-1.0 range for coloring
		relPressure := particle.Pressure - pressureOffset
		// Use sigmoid to get smooth 0-1 mapping that focuses on the typical range
		normalizedPressure := sigmoid(relPressure / pressureScale * 2.0)
		color := getColor(normalizedPressure)

		// Get or create batch for this color
		batch, exists := colorBatches[color]
		if !exists {
			batch = &ParticleBatch{
				color:  color,
				points: make([]sdl.Point, 0, 128), // Preallocate
			}
			colorBatches[color] = batch
		}

		// Scale particle position
		x := int32(particle.X * float64(scaleX))
		y := int32(particle.Y * float64(scaleY))

		// Store point for this particle
		batch.points = append(batch.points, sdl.Point{X: x, Y: y})
	}

	// Render batches
	for color, batch := range colorBatches {
		// Set color once per batch
		renderer.SetDrawColor(color.r, color.g, color.b, 255)

		// Draw all particles with this color
		for _, point := range batch.points {
			drawCircle(renderer, point.X, point.Y, int32(particleRadius))
		}
	}

	// Draw velocity vectors on top of particles if debug mode
	if showDebug {
		renderVelocities(renderer, particles, domain, windowWidth, windowHeight)
	}

	// Render mouse click effects
	if len(mouseEffects) > 0 {
		renderMouseEffects(renderer, mouseEffects, currentTime)
	}

	// Draw settings display if debug mode is on
	if showDebug {
		renderSettingsDisplay(renderer, windowWidth, settings)
	}

	renderer.Present()
}

// FontCache holds basic text rendering settings
type FontCache struct {
	initialized bool
	fontSize    int
}

// Global font cache
var fontCache FontCache

// Initialize font settings
func initFont(renderer *sdl.Renderer, size int) error {
	if !fontCache.initialized || fontCache.fontSize != size {
		fontCache.fontSize = size
		fontCache.initialized = true
	}
	return nil
}

// renderText creates a texture with the specified text using custom bitmap rendering
func renderText(renderer *sdl.Renderer, text string, textColor sdl.Color, size int) (*sdl.Texture, int32, int32, error) {
	// Initialize font
	if err := initFont(renderer, size); err != nil {
		return nil, 0, 0, err
	}

	// Create an enhanced bitmap text rendering
	// Calculate text dimensions with a more readable spacing
	charWidth := int32(size * 2 / 3) // Better character proportions
	letterSpacing := int32(size / 8) // Spacing between letters
	width := (int32(len(text)) * (charWidth + letterSpacing))
	height := int32(size * 3 / 2) // Better height

	// Create a texture for the text
	texture, err := renderer.CreateTexture(
		uint32(sdl.PIXELFORMAT_RGBA8888),
		sdl.TEXTUREACCESS_TARGET,
		width,
		height,
	)
	if err != nil {
		return nil, 0, 0, err
	}

	// Set up the texture for rendering
	texture.SetBlendMode(sdl.BLENDMODE_BLEND)

	// Save current render target
	originalTarget := renderer.GetRenderTarget()

	// Set our texture as the rendering target
	renderer.SetRenderTarget(texture)

	// Clear the texture with transparent background
	renderer.SetDrawColor(0, 0, 0, 0)
	renderer.Clear()

	// Draw text using custom bitmap characters
	renderer.SetDrawColor(textColor.R, textColor.G, textColor.B, textColor.A)

	// Draw each character using a custom bitmap font approximation
	for i, char := range text {
		x := int32(i) * (charWidth + letterSpacing)

		if char == ' ' {
			continue // Skip spaces
		}

		// Draw bitmap character based on what it is
		drawBitmapChar(renderer, char, x, 0, charWidth, height)
	}

	// Restore original render target
	renderer.SetRenderTarget(originalTarget)

	return texture, width, height, nil
}

// drawBitmapChar draws a character using bitmap shapes
func drawBitmapChar(renderer *sdl.Renderer, char rune, x, y, width, height int32) {
	// Base character metrics
	charHeight := height - 2
	middle := y + height/2
	top := y + 2
	bottom := y + charHeight

	switch char {
	case 'A', 'a':
		// Draw an 'A' shape
		renderer.DrawLine(x+width/2, top, x, bottom)                  // Left diagonal
		renderer.DrawLine(x+width/2, top, x+width, bottom)            // Right diagonal
		renderer.DrawLine(x+width/4, middle+2, x+width*3/4, middle+2) // Crossbar
	case 'B', 'b':
		// Draw a 'B' shape
		renderer.DrawLine(x, top, x, bottom)                             // Vertical line
		renderer.DrawLine(x, top, x+width*2/3, top)                      // Top horizontal
		renderer.DrawLine(x, middle, x+width*2/3, middle)                // Middle horizontal
		renderer.DrawLine(x, bottom, x+width*2/3, bottom)                // Bottom horizontal
		renderer.DrawLine(x+width*2/3, top, x+width, top+height/4)       // Top curve
		renderer.DrawLine(x+width, top+height/4, x+width*2/3, middle)    // Top curve continuation
		renderer.DrawLine(x+width*2/3, middle, x+width, middle+height/4) // Bottom curve
		renderer.DrawLine(x+width, middle+height/4, x+width*2/3, bottom) // Bottom curve continuation
	case 'C', 'c':
		// Draw a 'C' shape
		renderer.DrawLine(x+width, top+height/5, x+width*2/3, top)       // Top curve
		renderer.DrawLine(x+width*2/3, top, x+width/3, top)              // Top horizontal
		renderer.DrawLine(x+width/3, top, x, top+height/5)               // Top curve
		renderer.DrawLine(x, top+height/5, x, bottom-height/5)           // Left vertical
		renderer.DrawLine(x, bottom-height/5, x+width/3, bottom)         // Bottom curve
		renderer.DrawLine(x+width/3, bottom, x+width*2/3, bottom)        // Bottom horizontal
		renderer.DrawLine(x+width*2/3, bottom, x+width, bottom-height/5) // Bottom curve
	case 'D', 'd':
		// Draw a 'D' shape
		renderer.DrawLine(x, top, x, bottom)                    // Vertical line
		renderer.DrawLine(x, top, x+width*2/3, top)             // Top horizontal
		renderer.DrawLine(x, bottom, x+width*2/3, bottom)       // Bottom horizontal
		renderer.DrawLine(x+width*2/3, top, x+width, middle)    // Top curve
		renderer.DrawLine(x+width, middle, x+width*2/3, bottom) // Bottom curve
	case 'E', 'e':
		// Draw an 'E' shape
		renderer.DrawLine(x, top, x, bottom)              // Vertical
		renderer.DrawLine(x, top, x+width, top)           // Top horizontal
		renderer.DrawLine(x, middle, x+width*3/4, middle) // Middle horizontal
		renderer.DrawLine(x, bottom, x+width, bottom)     // Bottom horizontal
	case 'G', 'g':
		// Draw a 'G' shape
		renderer.DrawLine(x+width, top+height/5, x+width*2/3, top)       // Top curve
		renderer.DrawLine(x+width*2/3, top, x+width/3, top)              // Top horizontal
		renderer.DrawLine(x+width/3, top, x, top+height/5)               // Top curve
		renderer.DrawLine(x, top+height/5, x, bottom-height/5)           // Left vertical
		renderer.DrawLine(x, bottom-height/5, x+width/3, bottom)         // Bottom curve
		renderer.DrawLine(x+width/3, bottom, x+width*2/3, bottom)        // Bottom horizontal
		renderer.DrawLine(x+width*2/3, bottom, x+width, bottom-height/5) // Bottom curve
		renderer.DrawLine(x+width, bottom-height/5, x+width, middle)     // Right vertical
		renderer.DrawLine(x+width, middle, x+width*2/3, middle)          // Middle horizontal
	case 'I', 'i':
		// Draw an 'I' shape
		renderer.DrawLine(x+width/2, top, x+width/2, bottom) // Vertical line
	case 'L', 'l':
		// Draw an 'L' shape
		renderer.DrawLine(x, top, x, bottom)          // Vertical
		renderer.DrawLine(x, bottom, x+width, bottom) // Horizontal
	case 'M', 'm':
		// Draw an 'M' shape
		renderer.DrawLine(x, bottom, x, top)               // Left vertical
		renderer.DrawLine(x+width, bottom, x+width, top)   // Right vertical
		renderer.DrawLine(x, top, x+width/2, middle)       // Left diagonal
		renderer.DrawLine(x+width/2, middle, x+width, top) // Right diagonal
	case 'N', 'n':
		// Draw an 'N' shape
		renderer.DrawLine(x, bottom, x, top)             // Left vertical
		renderer.DrawLine(x+width, bottom, x+width, top) // Right vertical
		renderer.DrawLine(x, top, x+width, bottom)       // Diagonal
	case 'O', 'o':
		// Draw an 'O' shape
		renderer.DrawLine(x+width/3, top, x+width*2/3, top)                // Top horizontal
		renderer.DrawLine(x+width/3, bottom, x+width*2/3, bottom)          // Bottom horizontal
		renderer.DrawLine(x, top+height/4, x, bottom-height/4)             // Left vertical
		renderer.DrawLine(x+width, top+height/4, x+width, bottom-height/4) // Right vertical
		renderer.DrawLine(x+width/3, top, x, top+height/4)                 // Top-left curve
		renderer.DrawLine(x+width*2/3, top, x+width, top+height/4)         // Top-right curve
		renderer.DrawLine(x, bottom-height/4, x+width/3, bottom)           // Bottom-left curve
		renderer.DrawLine(x+width, bottom-height/4, x+width*2/3, bottom)   // Bottom-right curve
	case 'P', 'p':
		// Draw a 'P' shape
		renderer.DrawLine(x, top, x, bottom)                          // Vertical line
		renderer.DrawLine(x, top, x+width*2/3, top)                   // Top horizontal
		renderer.DrawLine(x, middle, x+width*2/3, middle)             // Middle horizontal
		renderer.DrawLine(x+width*2/3, top, x+width, top+height/4)    // Top curve
		renderer.DrawLine(x+width, top+height/4, x+width*2/3, middle) // Bottom curve
	case 'R', 'r':
		// Draw an 'R' shape
		renderer.DrawLine(x, top, x, bottom)                          // Vertical line
		renderer.DrawLine(x, top, x+width*2/3, top)                   // Top horizontal
		renderer.DrawLine(x, middle, x+width*2/3, middle)             // Middle horizontal
		renderer.DrawLine(x+width*2/3, top, x+width, top+height/4)    // Top curve
		renderer.DrawLine(x+width, top+height/4, x+width*2/3, middle) // Middle curve
		renderer.DrawLine(x+width*2/3, middle, x+width, bottom)       // Diagonal
	case 'S', 's':
		// Draw an 'S' shape
		renderer.DrawLine(x+width, top+height/5, x+width*2/3, top)       // Top curve
		renderer.DrawLine(x+width*2/3, top, x+width/3, top)              // Top horizontal
		renderer.DrawLine(x+width/3, top, x, top+height/5)               // Top-left curve
		renderer.DrawLine(x, top+height/5, x+width/3, middle)            // Middle-top curve
		renderer.DrawLine(x+width/3, middle, x+width*2/3, middle)        // Middle horizontal
		renderer.DrawLine(x+width*2/3, middle, x+width, bottom-height/5) // Middle-bottom curve
		renderer.DrawLine(x+width, bottom-height/5, x+width*2/3, bottom) // Bottom-right curve
		renderer.DrawLine(x+width*2/3, bottom, x+width/3, bottom)        // Bottom horizontal
		renderer.DrawLine(x+width/3, bottom, x, bottom-height/5)         // Bottom-left curve
	case 'T', 't':
		// Draw a 'T' shape
		renderer.DrawLine(x, top, x+width, top)              // Top horizontal
		renderer.DrawLine(x+width/2, top, x+width/2, bottom) // Vertical
	case 'U', 'u':
		// Draw a 'U' shape
		renderer.DrawLine(x, top, x, bottom-height/4)                    // Left vertical
		renderer.DrawLine(x+width, top, x+width, bottom-height/4)        // Right vertical
		renderer.DrawLine(x, bottom-height/4, x+width/3, bottom)         // Bottom-left curve
		renderer.DrawLine(x+width/3, bottom, x+width*2/3, bottom)        // Bottom horizontal
		renderer.DrawLine(x+width*2/3, bottom, x+width, bottom-height/4) // Bottom-right curve
	case 'V', 'v':
		// Draw a 'V' shape
		renderer.DrawLine(x, top, x+width/2, bottom)       // Left diagonal
		renderer.DrawLine(x+width/2, bottom, x+width, top) // Right diagonal
	case 'W', 'w':
		// Draw a 'W' shape
		renderer.DrawLine(x, top, x+width/4, bottom)              // First diagonal
		renderer.DrawLine(x+width/4, bottom, x+width/2, middle)   // Second diagonal
		renderer.DrawLine(x+width/2, middle, x+width*3/4, bottom) // Third diagonal
		renderer.DrawLine(x+width*3/4, bottom, x+width, top)      // Fourth diagonal
	case 'X', 'x':
		// Draw an 'X' shape
		renderer.DrawLine(x, top, x+width, bottom) // First diagonal
		renderer.DrawLine(x+width, top, x, bottom) // Second diagonal
	case 'Y', 'y':
		// Draw a 'Y' shape
		renderer.DrawLine(x, top, x+width/2, middle)            // Left diagonal
		renderer.DrawLine(x+width, top, x+width/2, middle)      // Right diagonal
		renderer.DrawLine(x+width/2, middle, x+width/2, bottom) // Vertical
	case 'Z', 'z':
		// Draw a 'Z' shape
		renderer.DrawLine(x, top, x+width, top)       // Top horizontal
		renderer.DrawLine(x+width, top, x, bottom)    // Diagonal
		renderer.DrawLine(x, bottom, x+width, bottom) // Bottom horizontal
	case ':':
		// Draw a colon
		renderer.DrawPoint(x+width/2, y+height/3)
		renderer.DrawPoint(x+width/2, y+height*2/3)
	case '.':
		// Draw a period
		renderer.DrawPoint(x+width/2, y+height*4/5)
	case ',':
		// Draw a comma
		renderer.DrawLine(x+width/2, y+height*2/3, x+width/3, y+height)
	case '-':
		// Draw a hyphen
		renderer.DrawLine(x, middle, x+width, middle)
	case '+':
		// Draw a plus
		renderer.DrawLine(x, middle, x+width, middle)            // Horizontal
		renderer.DrawLine(x+width/2, top+2, x+width/2, bottom-2) // Vertical
	case '|':
		// Draw a vertical bar
		renderer.DrawLine(x+width/2, top, x+width/2, bottom)
	case '1':
		// Draw a 1
		renderer.DrawLine(x+width/2, top, x+width/2, bottom)       // Vertical
		renderer.DrawLine(x+width/4, top+height/4, x+width/2, top) // Diagonal
	case '2':
		// Draw a 2
		renderer.DrawLine(x+width/4, top, x+width*3/4, top)           // Top horizontal
		renderer.DrawLine(x+width*3/4, top, x+width, top+height/4)    // Top-right curve
		renderer.DrawLine(x+width, top+height/4, x+width*3/4, middle) // Middle-right curve
		renderer.DrawLine(x+width*3/4, middle, x, bottom)             // Diagonal
		renderer.DrawLine(x, bottom, x+width, bottom)                 // Bottom horizontal
	case '3':
		// Draw a 3
		renderer.DrawLine(x, top, x+width*3/4, top)                      // Top horizontal
		renderer.DrawLine(x+width*3/4, top, x+width, top+height/4)       // Top-right curve
		renderer.DrawLine(x+width, top+height/4, x+width*3/4, middle)    // Middle-top curve
		renderer.DrawLine(x+width*3/4, middle, x+width, middle+height/4) // Middle-bottom curve
		renderer.DrawLine(x+width, middle+height/4, x+width*3/4, bottom) // Bottom-right curve
		renderer.DrawLine(x+width*3/4, bottom, x, bottom)                // Bottom horizontal
	case '4':
		// Draw a 4
		renderer.DrawLine(x+width/4, top, x+width/4, middle)     // Left vertical
		renderer.DrawLine(x+width/4, middle, x+width, middle)    // Middle horizontal
		renderer.DrawLine(x+width*3/4, top, x+width*3/4, bottom) // Right vertical
	case '5':
		// Draw a 5
		renderer.DrawLine(x+width, top, x, top)                          // Top horizontal
		renderer.DrawLine(x, top, x, middle)                             // Left vertical
		renderer.DrawLine(x, middle, x+width*3/4, middle)                // Middle horizontal
		renderer.DrawLine(x+width*3/4, middle, x+width, middle+height/4) // Middle-bottom curve
		renderer.DrawLine(x+width, middle+height/4, x+width*3/4, bottom) // Bottom-right curve
		renderer.DrawLine(x+width*3/4, bottom, x+width/4, bottom)        // Bottom horizontal
	case '6':
		// Draw a 6
		renderer.DrawLine(x+width, top, x+width/2, top)                  // Top horizontal
		renderer.DrawLine(x+width/2, top, x, middle)                     // Top-left curve
		renderer.DrawLine(x, middle, x, bottom-height/4)                 // Left vertical
		renderer.DrawLine(x, bottom-height/4, x+width/3, bottom)         // Bottom-left curve
		renderer.DrawLine(x+width/3, bottom, x+width*2/3, bottom)        // Bottom horizontal
		renderer.DrawLine(x+width*2/3, bottom, x+width, bottom-height/4) // Bottom-right curve
		renderer.DrawLine(x+width, bottom-height/4, x+width, middle)     // Right vertical
		renderer.DrawLine(x+width, middle, x, middle)                    // Middle horizontal
	case '7':
		// Draw a 7
		renderer.DrawLine(x, top, x+width, top)            // Top horizontal
		renderer.DrawLine(x+width, top, x+width/3, bottom) // Diagonal
	case '8':
		// Draw an 8
		renderer.DrawLine(x+width/3, top, x+width*2/3, top)              // Top horizontal
		renderer.DrawLine(x+width/3, bottom, x+width*2/3, bottom)        // Bottom horizontal
		renderer.DrawLine(x+width/3, middle, x+width*2/3, middle)        // Middle horizontal
		renderer.DrawLine(x+width/3, top, x, top+height/4)               // Top-left curve
		renderer.DrawLine(x, top+height/4, x+width/3, middle)            // Middle-top-left curve
		renderer.DrawLine(x+width*2/3, top, x+width, top+height/4)       // Top-right curve
		renderer.DrawLine(x+width, top+height/4, x+width*2/3, middle)    // Middle-top-right curve
		renderer.DrawLine(x+width/3, middle, x, middle+height/4)         // Middle-bottom-left curve
		renderer.DrawLine(x, middle+height/4, x+width/3, bottom)         // Bottom-left curve
		renderer.DrawLine(x+width*2/3, middle, x+width, middle+height/4) // Middle-bottom-right curve
		renderer.DrawLine(x+width, middle+height/4, x+width*2/3, bottom) // Bottom-right curve
	case '9':
		// Draw a 9
		renderer.DrawLine(x+width/3, top, x+width*2/3, top)        // Top horizontal
		renderer.DrawLine(x+width*2/3, top, x+width, top+height/4) // Top-right curve
		renderer.DrawLine(x+width, top+height/4, x+width, middle)  // Right vertical
		renderer.DrawLine(x+width, middle, x+width/2, bottom)      // Bottom-right curve
		renderer.DrawLine(x+width/2, bottom, x, bottom)            // Bottom horizontal
		renderer.DrawLine(x+width/3, top, x, top+height/4)         // Top-left curve
		renderer.DrawLine(x, top+height/4, x, middle)              // Left vertical
		renderer.DrawLine(x, middle, x+width, middle)              // Middle horizontal
	case '0':
		// Draw a 0
		renderer.DrawLine(x+width/3, top, x+width*2/3, top)                // Top horizontal
		renderer.DrawLine(x+width/3, bottom, x+width*2/3, bottom)          // Bottom horizontal
		renderer.DrawLine(x+width/3, top, x, top+height/4)                 // Top-left curve
		renderer.DrawLine(x, top+height/4, x, bottom-height/4)             // Left vertical
		renderer.DrawLine(x, bottom-height/4, x+width/3, bottom)           // Bottom-left curve
		renderer.DrawLine(x+width*2/3, top, x+width, top+height/4)         // Top-right curve
		renderer.DrawLine(x+width, top+height/4, x+width, bottom-height/4) // Right vertical
		renderer.DrawLine(x+width, bottom-height/4, x+width*2/3, bottom)   // Bottom-right curve
	// Add more character shapes as needed
	default:
		// Draw a simple rectangle for any other character
		renderer.DrawRect(&sdl.Rect{X: x + 1, Y: top + 1, W: width - 2, H: charHeight - 2})
	}
}

// Pixel-based text rendering approach - no font measurement needed

// renderSettingsDisplay shows current simulation parameters on screen
func renderSettingsDisplay(
	renderer *sdl.Renderer,
	windowWidth int32,
	settings SimSettings,
) {
	// Save current renderer state
	oldR, oldG, oldB, oldA, _ := renderer.GetDrawColor()
	var oldBlendMode sdl.BlendMode
	renderer.GetDrawBlendMode(&oldBlendMode)

	// Set semi-transparent background
	renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)
	renderer.SetDrawColor(0, 0, 0, 180)

	// Draw background panel
	panelWidth := int32(260)
	panelHeight := int32(180)
	panelX := windowWidth - panelWidth - 10
	panelY := int32(10)

	panel := sdl.Rect{
		X: panelX,
		Y: panelY,
		W: panelWidth,
		H: panelHeight,
	}
	renderer.FillRect(&panel)

	// Draw panel border
	renderer.SetDrawColor(100, 200, 255, 255)
	renderer.DrawRect(&panel)

	// Text rendering settings
	lineSpacing := int32(22)
	dotSize := int32(6)
	startY := panelY + 15
	fontSize := 14

	// Panel title
	titleTexture, titleWidth, titleHeight, err := renderText(
		renderer,
		"Simulation Controls",
		sdl.Color{R: 220, G: 220, B: 255, A: 255},
		fontSize+2,
	)
	if err == nil {
		defer titleTexture.Destroy()

		// Center the title
		titleRect := &sdl.Rect{
			X: panelX + (panelWidth-int32(titleWidth))/2,
			Y: panelY + 8,
			W: int32(titleWidth),
			H: int32(titleHeight),
		}
		renderer.Copy(titleTexture, nil, titleRect)
	}

	// Function to draw a parameter line with text label and value bar
	drawParamLine := func(lineNum int32, label string, color sdl.Color, value float64, format string) {
		y := startY + lineNum*lineSpacing + int32(titleHeight) + 5

		// Draw colored dot/rectangle for the parameter
		renderer.SetDrawColor(color.R, color.G, color.B, color.A)
		dotRect := sdl.Rect{X: panelX + 10, Y: y + 4, W: dotSize, H: dotSize}
		renderer.FillRect(&dotRect)

		// Draw text label using proper font rendering
		labelTexture, labelWidth, labelHeight, err := renderText(
			renderer,
			label,
			sdl.Color{R: 200, G: 200, B: 200, A: 255},
			fontSize,
		)
		if err == nil {
			defer labelTexture.Destroy()

			labelRect := &sdl.Rect{
				X: panelX + 20,
				Y: y,
				W: int32(labelWidth),
				H: int32(labelHeight),
			}
			renderer.Copy(labelTexture, nil, labelRect)
		}

		// Draw value indicator bar
		renderer.SetDrawColor(color.R, color.G, color.B, color.A)

		// Scale values appropriately - different parameters have different ranges
		var valueWidth int32
		switch label {
		case "Gravity":
			valueWidth = int32(math.Min(150, math.Max(10, value/500.0)))
		case "Pressure":
			valueWidth = int32(math.Min(150, math.Max(10, value/250.0)))
		case "Attract":
			// For attraction, center at 0 and scale appropriately
			valueWidth = int32(math.Min(150, math.Max(10, (value+100)/2.0)))
		case "Particles":
			valueWidth = int32(math.Min(150, math.Max(10, value/50.0)))
		case "Radius":
			valueWidth = int32(math.Min(150, math.Max(10, value*5.0)))
		case "Mouse Force":
			valueWidth = int32(math.Min(150, math.Max(10, value*100.0)))
		case "Mouse Force Radius":
			valueWidth = int32(math.Min(150, math.Max(10, value*100.0)))
		default:
			valueWidth = int32(math.Min(150, math.Max(10, value*100.0)))
		}

		// Draw value bar
		barY := y + int32(labelHeight) + 2
		barX := panelX + 20
		barRect := sdl.Rect{X: barX, Y: barY, W: valueWidth, H: 3}
		renderer.FillRect(&barRect)

		// For attraction, add a marker at zero
		if label == "Attract" {
			// Draw zero marker (vertical line at the baseline position)
			renderer.SetDrawColor(220, 220, 220, 255)
			zeroX := barX + 75 // Middle position for zero
			renderer.DrawLine(zeroX, barY-2, zeroX, barY+5)
		}

		// Draw numerical value as proper text
		valStr := fmt.Sprintf(format, value)
		valueTexture, valueWidth, valueHeight, err := renderText(
			renderer,
			valStr,
			sdl.Color{R: 180, G: 180, B: 180, A: 255},
			fontSize-1,
		)
		if err == nil {
			defer valueTexture.Destroy()

			valueRect := &sdl.Rect{
				X: barX + barRect.W + 5,
				Y: y,
				W: int32(valueWidth),
				H: int32(valueHeight),
			}
			renderer.Copy(valueTexture, nil, valueRect)
		}
	}

	// Draw parameters
	drawParamLine(0, "Pressure", sdl.Color{R: 255, G: 100, B: 100, A: 255}, settings.Pressure/250, "%.0f")
	drawParamLine(1, "Gravity", sdl.Color{R: 100, G: 255, B: 100, A: 255}, settings.Gravity, "%.0f")
	drawParamLine(2, "Drag", sdl.Color{R: 100, G: 100, B: 255, A: 255}, settings.Drag*100, "%.2f")
	drawParamLine(3, "Smooth", sdl.Color{R: 255, G: 255, B: 100, A: 255}, settings.Smoothing*100, "%.2f")
	drawParamLine(4, "Attract", sdl.Color{R: 255, G: 150, B: 0, A: 255}, settings.Attraction, "%.0f")
	drawParamLine(5, "Radius", sdl.Color{R: 0, G: 200, B: 200, A: 255}, settings.InteractionRadius, "%.1f")
	drawParamLine(6, "Mouse Force", sdl.Color{R: 255, G: 100, B: 255, A: 255}, settings.MouseForce, "%.0f")
	drawParamLine(7, "Mouse Force Radius", sdl.Color{R: 255, G: 100, B: 255, A: 255}, settings.MouseForceRadius, "%.1f")
	drawParamLine(8, "Particles", sdl.Color{R: 255, G: 100, B: 255, A: 255}, float64(settings.ParticleCount), "%d")

	// Add key help text
	renderer.SetDrawColor(200, 200, 200, 255)
	helpRect := sdl.Rect{X: panelX + 10, Y: panelY + panelHeight - 25, W: panelWidth - 20, H: 1}
	renderer.FillRect(&helpRect)

	// Add keyboard controls help text
	helpText := "D: Toggle Debug | Space: Pause"
	helpTexture, helpWidth, helpHeight, err := renderText(
		renderer,
		helpText,
		sdl.Color{R: 180, G: 180, B: 180, A: 255},
		12,
	)
	if err == nil {
		defer helpTexture.Destroy()

		helpTextRect := &sdl.Rect{
			X: panelX + (panelWidth-int32(helpWidth))/2,
			Y: panelY + panelHeight - 20,
			W: int32(helpWidth),
			H: int32(helpHeight),
		}
		renderer.Copy(helpTexture, nil, helpTextRect)
	}

	// Restore renderer state
	renderer.SetDrawColor(oldR, oldG, oldB, oldA)
	renderer.SetDrawBlendMode(oldBlendMode)
}
