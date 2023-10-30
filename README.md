```console
git clone https://github.com/zzstoatzz/fluids.git

cd fluids

# no gravity, 500 particles, time step 0.007
go run main.go

# gravity, 800 particles, time step 0.01, radius 5, 120 fps, gravity -1000
go run main.go -n 800 -dt 0.01 -radius 4 -fps 240 -g -1000
```
### flags
- n: number of particles
- radius: radius of particles
- fps: frames per second
- g: gravity (defaults to -9.81)
- dt: time step (defaults to 0.005)

### controls
- click to create a small blast radius
- press g to toggle gravity
- press space to pause
- press r to reset