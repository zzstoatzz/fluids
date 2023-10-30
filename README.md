```console
git clone https://github.com/zzstoatzz/fluids.git

cd fluids

go run main.go
```
### flags
- n: number of particles (defaults to 500)
- radius: radius of particles (defaults to 2.4)
- fps: frames per second (defaults to 480)
- g: gravity (defaults to disabled and -100000 if gravity toggled while not set by flag)
- dt: time step (defaults to 0.0005 seconds)
- boom: magntiude of left click blast (defaults to 100.0)

### example
```console
go run main.go -n 100 -radius 4 -pressure 100000 -fps 240 -dt 0.0001 -boom 1000
```

### in-simulation controls
- click to create a small blast radius
- press g to toggle gravity
- press space to pause
- press r to reset