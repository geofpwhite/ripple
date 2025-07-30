package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"

	"fortio.org/terminal/ansipixels"
)

const (
	red   = "\033[38;2;250;0;0m"
	blue  = "\033[38;2;0;0;250m"
	green = "\033[38;2;0;250;0m"
)

var colorsToChoose = [...]string{
	randomColor(),
	randomColor(),
	randomColor(),
	randomColor(),
	red, blue, green,
}

func randomColor() string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", rand.IntN(250), rand.IntN(250), rand.IntN(250))
}

func toRGB(s string) (int, int, int) {
	s = s[6 : len(s)-1]
	fields := strings.Split(s, ";")[1:]

	r, err := strconv.Atoi(strings.Trim(fields[0], " "))
	if err != nil {
		panic(strings.Trim(fields[0], " "))
	}
	g, err := strconv.Atoi(strings.Trim(fields[1], " "))
	if err != nil {
		panic("2")
	}
	b, err := strconv.Atoi(strings.Trim(fields[2], " "))
	if err != nil {
		panic("3")
	}

	return r, g, b
}

func main() {
	fpsFlag := flag.Float64("fps", 600., "change the fps") // high fps makes it look super smooth
	filled := flag.Bool("fill", false, "makes ripples filled in")
	flag.Parse()
	ap := ansipixels.NewAnsiPixels(*fpsFlag)

	err := ap.GetSize()
	paused := false
	if err != nil {
		panic("can't get term size")
	}
	ap.MouseTrackingOn()

	ap.OnResize = func() error {
		return ap.GetSize()
	}
	defer func() {
		ap.MouseTrackingOff()

		ap.MoveCursor(0, 0)
		ap.MoveCursor(0, ap.H-2)
		ap.Restore()
	}()

	err = ap.Open()
	if err != nil {
		panic("can't open")
	}
	clicks := make(map[[2]int]int)
	colors := make(map[[2]int]string)
	ap.StartSyncMode()
	ap.ClearScreen()
	ap.EndSyncMode()
	ap.HideCursor()
	for {
		_, err := ap.ReadOrResizeOrSignalOnce()
		if err != nil {
			panic("can't read/resize/signal")
		}
		if !paused {
			for key := range clicks {
				clicks[key] += clicks[key]/500 + 1
			}
		}
		if ap.LeftClick() {
			clicks[[2]int{ap.Mx, ap.My}] = 0
			colors[[2]int{ap.Mx, ap.My}] = colorsToChoose[rand.IntN(len(colorsToChoose))]
		}
		Draw(ap, clicks, colors, *filled)
		if len(ap.Data) == 0 {
			continue
		}
		switch ap.Data[0] {
		case ' ':
			paused = !paused
		case 'c':
			ap.StartSyncMode()
			clear(clicks)
			ap.ClearScreen()
			ap.EndSyncMode()
		case 'q':
			return
		}
	}
}

func Draw(ap *ansipixels.AnsiPixels, clicks map[[2]int]int, colors map[[2]int]string, filled bool) {
	ap.StartSyncMode()
	if !filled {
		ap.ClearScreen()
	}
	for i := range ap.W {
		for j := range ap.H {
			// color := image.RGBA{}
			r, g, b := 0, 0, 0
			count := 0
			for coords, radius := range clicks {
				if filled {
					if distance := (i-coords[0])*(i-coords[0]) + ((j*2)-(coords[1]*2))*((j*2)-(2*coords[1])); float64(distance) <= float64(radius)+(float64(radius)/5.) {
						// ap.WriteAtStr(i, j, colors[coords]+string(ansipixels.FullPixel))
						r1, g1, b1 := toRGB(colors[coords])
						r += r1
						g += g1
						b += b1
						count++
					}
				} else {
					if distance := (i-coords[0])*(i-coords[0]) + ((j*2)-(coords[1]*2))*((j*2)-(2*coords[1])); float64(distance) <= float64(radius)+(float64(radius)/5.) && float64(distance) >= float64(radius)-(float64(radius)/5.) {
						// ap.WriteAtStr(i, j, colors[coords]+string(ansipixels.FullPixel))
						r1, g1, b1 := toRGB(colors[coords])
						r += r1
						g += g1
						b += b1
						count++
					}
				}
			}
			if count != 0 {
				r /= count + 1
				g /= count + 1
				b /= count + 1
				ap.WriteAtStr(i, j, fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)+string(ansipixels.FullPixel))
			}

			if clicks[[2]int{i, j}] >= min(ap.W, ap.H)*min(ap.W, ap.H) {
				delete(clicks, [2]int{i, j})
				ap.ClearScreen()
			}
		}
	}
	// time.Sleep(10 * time.Millisecond)
	ap.EndSyncMode()
}
