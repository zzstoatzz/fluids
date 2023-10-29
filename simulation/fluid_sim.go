package simulation

const N_ITERATIONS = 100

type FluidSim struct {
	Nx, Ny              int     // Grid size
	Dt, Dx, Dy          float64 // Time and space steps
	Rho, Nu             float64 // Density and viscosity
	U, V, P, Un, Vn, Pn []float64
}

type InitialConditionFunc func(i, j int, nx, ny int) (u, v float64)

func NewFluidSim(nx, ny int, dt, dx, dy, rho, nu float64, initFunc InitialConditionFunc) *FluidSim {
	totalSize := nx * ny

	// Initialize 1D slices for u, v, p, un, vn, pn
	u := make([]float64, totalSize)
	v := make([]float64, totalSize)
	p := make([]float64, totalSize)
	un := make([]float64, totalSize)
	vn := make([]float64, totalSize)
	pn := make([]float64, totalSize)

	for i := 0; i < nx; i++ {
		for j := 0; j < ny; j++ {
			index := i*ny + j
			u[index], v[index] = initFunc(i, j, nx, ny)
		}
	}

	return &FluidSim{
		Nx:  nx,
		Ny:  ny,
		Dt:  dt,
		Dx:  dx,
		Dy:  dy,
		Rho: rho,
		Nu:  nu,
		U:   u,
		V:   v,
		P:   p,
		Un:  un,
		Vn:  vn,
		Pn:  pn,
	}
}

func (sim *FluidSim) UpdateVelocities() {
	dt := sim.Dt
	g := -10.0 // gravitational acceleration

	for i := 0; i < sim.Nx; i++ {
		for j := 0; j < sim.Ny; j++ {
			index := i*sim.Ny + j
			// Only vertical velocity (V) is affected by gravity
			sim.V[index] += g * dt
		}
	}
}

func (sim *FluidSim) ApplyBoundaryConditions() {
	for i := 0; i < sim.Nx; i++ {
		sim.U[i*sim.Ny] = 0.0
		sim.U[i*sim.Ny+sim.Ny-1] = 0.0
		sim.V[i*sim.Ny] = 0.0
		sim.V[i*sim.Ny+sim.Ny-1] = 0.0
	}
	for j := 0; j < sim.Ny; j++ {
		sim.U[j] = 0.0
		sim.U[(sim.Nx-1)*sim.Ny+j] = 0.0
		sim.V[j] = 0.0
		sim.V[(sim.Nx-1)*sim.Ny+j] = 0.0
	}
}

func (sim *FluidSim) Step() {
	sim.UpdateVelocities()
	sim.ApplyBoundaryConditions()
}
