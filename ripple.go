package main

import (
	"flag"
	"fmt"
	"math"
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
	if *filled && *fpsFlag == 600. {
		*fpsFlag = 9000
	}
	ap := ansipixels.NewAnsiPixels(*fpsFlag)
	var draw func(ap *ansipixels.AnsiPixels, clicks map[[2]int]int, colors map[[2]int]string, filled bool)
	if *filled {
		draw = Draw
	} else {
		draw = Draw2
	}

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
	last := make([][][3]int, ap.W)
	for i := range last {
		last[i] = make([][3]int, ap.H)
	}
	// defer func() {
	// 	for _, row := range last {
	// 		fmt.Println(row)
	// 	}
	// }()

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

		draw(ap, clicks, colors, *filled)
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

func circle(ap *ansipixels.AnsiPixels, r, x, y int, color string) {
	ap.StartSyncMode()
	for i := 0.; i < 2.*math.Pi; i += 2. * math.Pi / 360. {
		ex := float64(r) * (math.Cos(i))
		ey := float64(r) * (math.Sin(i)) / 2
		r, g, b := toRGB(color)
		rx := max((int(ex) + x), 0)
		ry := max((int(ey) + y), 0)
		if rx/2 >= ap.W {
			rx = ap.W - 1
		}
		if ry/2 >= ap.H {
			ry = ap.H - 1
		}
		if rx < 0 || ry < 0 {
			continue
		}

		ap.WriteAtStr(rx, ry, fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)+string(ansipixels.FullPixel))
	}
	ap.EndSyncMode()
}

func Draw2(ap *ansipixels.AnsiPixels, clicks map[[2]int]int, colors map[[2]int]string, filled bool) {
	if !filled {
		ap.ClearScreen()
	}
	for coords, val := range clicks {

		if float64(val) <= math.Sqrt(float64(ap.W*ap.W)+float64(ap.H*ap.H)) {
			circle(ap, val, coords[0], coords[1], colors[coords])
		} else {
			delete(clicks, coords)
			ap.StartSyncMode()
			ap.ClearScreen()
			ap.EndSyncMode()
		}

	}
}

func Draw(ap *ansipixels.AnsiPixels, clicks map[[2]int]int, colors map[[2]int]string, filled bool) {
	ap.StartSyncMode()
	if !filled {
		ap.ClearScreen()
	}
	for i := range ap.W {
		var prev rune = 0
		for j := range ap.H * 2 {
			// color := image.RGBA{}
			r, g, b := 0, 0, 0
			count := 0
			var pixel rune
			for coords, radius := range clicks {
				if filled {
					if distance := (i-coords[0])*(i-coords[0]) + ((j)-(coords[1]*2))*((j)-(2*coords[1])); float64(distance) <= float64(radius)+(float64(radius)/5.) {
						// ap.WriteAtStr(i, j, colors[coords]+string(ansipixels.FullPixel))
						r1, g1, b1 := toRGB(colors[coords])
						r += r1
						g += g1
						b += b1
						count++
					}
				} else {
					if distance := (i-coords[0])*(i-coords[0]) + ((j)-(coords[1]*2))*((j)-(2*coords[1])); float64(distance) <= float64(radius)+(float64(radius)/5.) && float64(distance) >= float64(radius)-(float64(radius)/5.) {
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
				if j%2 == 0 {
					prev = ansipixels.TopHalfPixel
					ap.WriteAtStr(i, j/2, fmt.Sprintf("\033[38;2;%d;%d;%dm", r/(count+1), g/(count+1), b/(count+1))+string(prev))
				} else {
					if prev == ansipixels.TopHalfPixel {
						pixel = ansipixels.FullPixel
					} else {
						pixel = ansipixels.BottomHalfPixel
					}
					r /= count + 1
					g /= count + 1
					b /= count + 1
					ap.WriteAtStr(i, j/2, fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)+string(pixel))
				}
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
