package spatial

import (
	"fluids/core"
	"fmt"
)

type Cell struct {
	Particles []int // Indices of particles in this cell
}

type Grid struct {
	CellMap              map[string][]int // Map from cell key to particle indices
	CellSize             float64
	NumCellsX, NumCellsY int
}

func NewGrid(cellSize float64, domainX, domainY int) *Grid {
	return &Grid{
		CellMap:   make(map[string][]int),
		CellSize:  cellSize,
		NumCellsX: int(float64(domainX) / cellSize),
		NumCellsY: int(float64(domainY) / cellSize),
	}
}

// Update populates the grid cells with particle indices
func (g *Grid) Update(particles []core.Particle) {
	g.CellMap = make(map[string][]int) // Clear existing cells

	for idx, p := range particles {
		i := int(p.X / g.CellSize)
		j := int(p.Y / g.CellSize)
		key := fmt.Sprintf("%d-%d", i, j) // Generate cell key

		g.CellMap[key] = append(g.CellMap[key], idx)
	}
}
