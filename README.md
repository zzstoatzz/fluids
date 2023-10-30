```console
git clone https://github.com/zzstoatzz/fluids.git

cd fluids

go run main.go -g 0 # 300 particles, no gravity

go run main.go -n 1000 -radius 5 -fps 60 -g 0 # 1000 particles, radius 5, 60 fps, no gravity
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