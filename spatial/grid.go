package spatial

import (
	"fluids/core"
)

type Cell struct {
	Particles []int // Indices of particles in this cell
}

type Grid struct {
	Cells                [][]Cell
	CellSize             float64
	NumCellsX, NumCellsY int
}

// InitializeGrid initializes the grid and allocates memory for cells
func InitializeGrid(cellSize float64, domainX, domainY int) *Grid {
	numCellsX := int(float64(domainX) / cellSize)
	numCellsY := int(float64(domainY) / cellSize)
	cells := make([][]Cell, numCellsX)
	for i := range cells {
		cells[i] = make([]Cell, numCellsY)
	}
	return &Grid{
		Cells:     cells,
		CellSize:  cellSize,
		NumCellsX: numCellsX,
		NumCellsY: numCellsY,
	}
}

// Update populates the grid cells with particle indices
func (g *Grid) Update(particles []core.Particle) {
	// Clear existing cells
	for i := range g.Cells {
		for j := range g.Cells[i] {
			g.Cells[i][j].Particles = nil
		}
	}
	// Populate cells with particle indices
	for idx, p := range particles {
		i := int(p.X / g.CellSize)
		j := int(p.Y / g.CellSize)
		if i >= 0 && i < g.NumCellsX && j >= 0 && j < g.NumCellsY {
			g.Cells[i][j].Particles = append(g.Cells[i][j].Particles, idx)
		}
	}
}
