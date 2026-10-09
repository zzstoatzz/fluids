// oracle generates random sim states, runs them through the go sim, and dumps
// inputs + outputs as json so the zig port can replay and compare them.
//
//	go run ./cmd/oracle -cases 300 -seed 1 -portable-pow > /tmp/parity.json
//	zig build parity -- /tmp/parity.json
package main

import (
	"encoding/json"
	"flag"
	"fluids/input"
	"fluids/simulation"
	"fluids/spatial"
	"math"
	"math/rand"
	"os"
)

type State struct {
	X, Y, Vx, Vy, Density, Pressure, Fx, Fy []float64
}

type Poke struct {
	Before        int // poke before this step index
	X, Y          float64
	Force, Radius float64
}

type Case struct {
	Seed              int64
	PortablePow       bool
	N                 int
	DomainX, DomainY  float64
	Params            simulation.SimParameters
	InteractionRadius float64 // runtime radius (may differ from grid cell size)
	Gravity           float64
	Pressure          float64
	Radius, Mass      []float64
	Init              State
	Pokes             []Poke
	Tape              []float64
	Steps             []State
	Mean, Std         []float64
}

func pick(r *rand.Rand, lo, hi float64) float64 { return lo + r.Float64()*(hi-lo) }

func snapshot(sim *simulation.FluidSim) State {
	n := len(sim.Particles)
	s := State{make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)}
	for i, p := range sim.Particles {
		s.X[i], s.Y[i], s.Vx[i], s.Vy[i], s.Density[i], s.Pressure[i], s.Fx[i], s.Fy[i] = p.X, p.Y, p.Vx, p.Vy, p.Density, p.Pressure, p.Force.X, p.Force.Y
	}
	return s
}

func finite(s State) bool {
	for _, a := range [][]float64{s.X, s.Y, s.Vx, s.Vy, s.Density, s.Pressure, s.Fx, s.Fy} {
		for _, v := range a {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return false
			}
		}
	}
	return true
}

func genCase(seed int64) (Case, bool) {
	r := rand.New(rand.NewSource(seed))
	c := Case{Seed: seed}
	c.N = 1 + r.Intn(400)
	c.DomainX = pick(r, 40, 400)
	c.DomainY = pick(r, 40, 400)
	p := simulation.GetDefaultSimParameters()
	p.InteractionRadius = pick(r, 3, 25)
	p.SmoothingFactor = pick(r, 0.05, 0.4)
	p.DampeningFactor = pick(r, 0.01, 0.3)
	p.DragEnabled = r.Float64() < 0.8
	switch r.Intn(3) {
	case 0:
		p.AttractionFactor = 0
	case 1:
		p.AttractionFactor = -50000
	default:
		p.AttractionFactor = pick(r, -1e5, 1e5)
	}
	p.Nu = pick(r, 0, 3)
	p.Rho0 = []float64{1000, pick(r, 0, 0.05), pick(r, 0, 2000)}[r.Intn(3)]
	p.Dt = pick(r, 1e-4, 2e-3)
	c.Params = p
	c.InteractionRadius = p.InteractionRadius
	if r.Float64() < 0.2 { // the runtime e/t keys change h without rebuilding the grid
		c.InteractionRadius = pick(r, 1, 30)
	}
	c.Gravity = []float64{0, 100000, pick(r, -1e5, 1e5)}[r.Intn(3)]
	c.Pressure = math.Exp(pick(r, 0, math.Log(50000)))

	// positions: uniform, clustered, or hugging the walls
	mode := r.Intn(3)
	cx, cy := pick(r, 0, c.DomainX), pick(r, 0, c.DomainY)
	c.Init = State{make([]float64, c.N), make([]float64, c.N), make([]float64, c.N), make([]float64, c.N), make([]float64, c.N), make([]float64, c.N), nil, nil}
	c.Radius = make([]float64, c.N)
	c.Mass = make([]float64, c.N)
	vscale := []float64{0.1, 10, 500}[r.Intn(3)]
	for i := 0; i < c.N; i++ {
		switch mode {
		case 0:
			c.Init.X[i], c.Init.Y[i] = r.Float64()*c.DomainX, r.Float64()*c.DomainY
		case 1:
			c.Init.X[i] = math.Min(math.Max(cx+r.NormFloat64()*p.InteractionRadius*2, 0.01), c.DomainX-0.01)
			c.Init.Y[i] = math.Min(math.Max(cy+r.NormFloat64()*p.InteractionRadius*2, 0.01), c.DomainY-0.01)
		default:
			c.Init.X[i] = []float64{pick(r, 0, 1), pick(r, c.DomainX-1, c.DomainX)}[r.Intn(2)]
			c.Init.Y[i] = r.Float64() * c.DomainY
		}
		c.Init.Vx[i] = r.NormFloat64() * vscale
		c.Init.Vy[i] = r.NormFloat64() * vscale
		c.Init.Density[i] = p.Rho0
		c.Radius[i] = 2.0 * (0.8 + 0.4*r.Float64())
		c.Mass[i] = math.Pi * c.Radius[i] * c.Radius[i]
	}
	steps := 1 + r.Intn(4)
	for k := 0; k < steps; k++ {
		if r.Float64() < 0.3 {
			ww, wh := int32(c.DomainX), int32(c.DomainY)
			mx, my := int32(r.Intn(int(ww)+1)), int32(r.Intn(int(wh)+1))
			c.Pokes = append(c.Pokes, Poke{Before: k,
				X: float64(mx) / float64(ww) * c.DomainX, Y: float64(my) / float64(wh) * c.DomainY,
				Force: pick(r, 50, 5000), Radius: pick(r, 5, 200)})
		}
	}

	// run go
	sim := simulation.NewFluidSim(c.N, simulation.Domain{X: c.DomainX, Y: c.DomainY}, p, rand.New(rand.NewSource(0)))
	sim.InteractionRadius = c.InteractionRadius
	for i := range sim.Particles {
		q := &sim.Particles[i]
		q.X, q.Y, q.Vx, q.Vy, q.Density, q.Pressure = c.Init.X[i], c.Init.Y[i], c.Init.Vx[i], c.Init.Vy[i], c.Init.Density[i], 0
		q.Radius, q.Mass = c.Radius[i], c.Mass[i]
	}
	jr := rand.New(rand.NewSource(seed ^ 0x5eed))
	spatial.JitterFloat64 = func() float64 { v := jr.Float64(); c.Tape = append(c.Tape, v); return v }
	pi := 0
	for k := 0; k < steps; k++ {
		for pi < len(c.Pokes) && c.Pokes[pi].Before == k {
			pk := c.Pokes[pi]
			ww, wh := int32(c.DomainX), int32(c.DomainY)
			mx := int32(math.Round(pk.X / c.DomainX * float64(ww)))
			my := int32(math.Round(pk.Y / c.DomainY * float64(wh)))
			input.ApplyMouseForceToParticles(sim, mx, my, ww, wh, pk.Force, pk.Radius)
			pi++
		}
		mean, std := sim.Step(c.Gravity, c.Pressure, sim.Dt)
		s := snapshot(sim)
		if !finite(s) || math.IsNaN(mean) || math.IsNaN(std) || math.IsInf(mean, 0) || math.IsInf(std, 0) {
			return c, false
		}
		c.Steps = append(c.Steps, s)
		c.Mean = append(c.Mean, mean)
		c.Std = append(c.Std, std)
	}
	return c, true
}

func main() {
	cases := flag.Int("cases", 200, "number of cases")
	seed := flag.Int64("seed", 1, "base seed")
	portable := flag.Bool("portable-pow", false, "use go's portable exp/log in Pow so results are bit-reproducible")
	flag.Parse()
	if *portable {
		simulation.Pow = portablePow
	}
	simulation.SetParallelConfig(simulation.ParallelConfig{NumWorkers: 1, MinimumBatchSize: 1 << 30})
	var out []Case
	for i := int64(0); len(out) < *cases && i < int64(*cases)*20; i++ {
		if c, ok := genCase(*seed*1_000_003 + i); ok {
			c.PortablePow = *portable
			out = append(out, c)
		}
	}
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(out); err != nil {
		panic(err)
	}
}
