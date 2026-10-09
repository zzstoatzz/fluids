package spatial

import (
	"fluids/core"
)

// CellIndex is an efficient integer-based cell identifier
type CellIndex int64

// MakeCellIndex creates a new cell index from x,y coordinates
func MakeCellIndex(cellX, cellY int) CellIndex {
	// Pack two integers into a single int64 for faster lookup
	// Using bit shifting to combine x and y into a single value
	return CellIndex((int64(cellX) << 32) | int64(cellY&0xFFFFFFFF))
}

// GetCoordinates extracts the x,y cell coordinates from a CellIndex
func (ci CellIndex) GetCoordinates() (int, int) {
	return int(int64(ci) >> 32), int(int64(ci) & 0xFFFFFFFF)
}

type Grid struct {
	CellMap              map[CellIndex][]int
	CellSize             float64
	NumCellsX, NumCellsY int
}

func NewGrid(cellSize float64, domainX, domainY int) *Grid {
	return &Grid{
		CellMap:   make(map[CellIndex][]int),
		CellSize:  cellSize,
		NumCellsX: int(float64(domainX) / cellSize),
		NumCellsY: int(float64(domainY) / cellSize),
	}
}

// Update populates the grid cells with particle indices
// This is a hot path, so we optimize for performance
func (g *Grid) Update(particles []core.Particle) {
	// Clear cell map but try to reuse capacity where possible
	for k := range g.CellMap {
		// Keep the allocated slices but reset length to 0
		if cap(g.CellMap[k]) > 0 {
			g.CellMap[k] = g.CellMap[k][:0]
		} else {
			delete(g.CellMap, k)
		}
	}

	// Add particles to cells
	for idx := range particles { // Iterate by index to modify the original slice elements
		p := &particles[idx] // Get a pointer to the original particle
		i := int(p.X / g.CellSize)
		j := int(p.Y / g.CellSize)
		cellIdx := MakeCellIndex(i, j)

		// Store cell coordinates on the particle for faster neighbor lookup
		p.CellX = i
		p.CellY = j

		g.CellMap[cellIdx] = append(g.CellMap[cellIdx], idx)
	}
}

// GetNeighborParticles returns all particles in neighboring cells efficiently.
func (g *Grid) GetNeighborParticles(cellX, cellY int) []int {
	// Reset the pre-allocated slice
	var neighborIndices []int // Create a new slice for this call

	// Check the specified cell and all 8 neighboring cells
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			neighborCellX, neighborCellY := cellX+dx, cellY+dy
			cellIdx := MakeCellIndex(neighborCellX, neighborCellY)

			if indices, found := g.CellMap[cellIdx]; found {
				neighborIndices = append(neighborIndices, indices...)
			}
		}
	}

	return neighborIndices
}
