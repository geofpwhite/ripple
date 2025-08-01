package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
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
	fpsFlag := flag.Float64("fps", 100., "change the fps") // high fps makes it look super smooth
	flag.Parse()
	colorCount := 0
	ap := ansipixels.NewAnsiPixels(*fpsFlag)

	err := ap.GetSize()
	paused := false
	if err != nil {
		panic("can't get term size")
	}
	ap.MouseClickOn()
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
	rightClicks := make(map[[2]int]int)
	colors := make(map[[2]int]string)
	ap.StartSyncMode()
	ap.ClearScreen()
	ap.EndSyncMode()
	ap.HideCursor()
	list := make([][2]int, 0)
	orderedByChosen := &list
	rightlist := make([][2]int, 0)
	rightorderedByChosen := &rightlist
	last := make([][][3]int, ap.W)
	for i := range last {
		last[i] = make([][3]int, ap.H)
	}
	for {
		img := image.NewRGBA(image.Rect(0, 0, ap.W, ap.H*2))
		_, err := ap.ReadOrResizeOrSignalOnce()
		if err != nil {
			panic("can't read/resize/signal")
		}
		if !paused {
			for key := range clicks {
				clicks[key] += clicks[key]/500 + 1
			}
			for key := range rightClicks {
				rightClicks[key] += rightClicks[key]/500 + 1
			}
		}
		if ap.LeftClick() {
			clicks[[2]int{ap.Mx, ap.My * 2}] = 0
			r, g, b := toRGB(colorsToChoose[0])
			colors[[2]int{ap.Mx, ap.My * 2}] = fmt.Sprintf("\033[38;2;%d;%d;%dm", (r+colorCount)%264, (g+colorCount)%264, (b+colorCount)%264)
			colorCount = (colorCount + 100) % 264
			*orderedByChosen = append(*orderedByChosen, [2]int{ap.Mx, ap.My * 2})
		} else if ap.RightClick() {
			rightClicks[[2]int{ap.Mx, ap.My * 2}] = 0
			// colors[[2]int{ap.Mx, ap.My * 2}] = colorsToChoose[colorChosen]
			r, g, b := toRGB(colorsToChoose[1])
			colors[[2]int{ap.Mx, ap.My * 2}] = fmt.Sprintf("\033[38;2;%d;%d;%dm", (r+colorCount)%264, (g+colorCount)%264, (b+colorCount)%264)
			colorCount = (colorCount + 20) % 264
			*rightorderedByChosen = append(*rightorderedByChosen, [2]int{ap.Mx, ap.My * 2})
		}

		drawDiscs(ap, clicks, colors, orderedByChosen, img)
		drawCircles(ap, rightClicks, colors, rightorderedByChosen, img)
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

// circles are hollow
func drawCircles(ap *ansipixels.AnsiPixels, clicks map[[2]int]int, colors map[[2]int]string, orderedByChosen *[][2]int, img *image.RGBA) {
	for _, coords := range *orderedByChosen {
		radius := clicks[coords]
		if radius < 1 {
			continue
		}
		x, y := coords[0], coords[1]
		for i := 0.; i < 2.*math.Pi; i += 2. * math.Pi / 365. {
			ex := .3 * float64(radius) * (math.Cos(i))
			ey := .3 * float64(radius) * (math.Sin(i))
			rx := max((int(ex) + x), 0)
			ry := max((int(ey) + y), 0)
			r, g, b := toRGB(colors[coords])
			img.Set(rx, ry, color.RGBA{uint8(r), uint8(g), uint8(b), 100})
		}
		if float64(radius)*.3 > float64(ap.H) {
			delete(clicks, coords)
			*orderedByChosen = (*orderedByChosen)[1:]
		}
	}
	ap.StartSyncMode()
	var err error
	switch {
	case ap.TrueColor:
		err = ap.DrawTrueColorImage(ap.Margin, ap.Margin, img)
	case ap.Color:
		err = ap.Draw216ColorImage(ap.Margin, ap.Margin, img)
	default:
		err = ap.Draw216ColorImage(ap.Margin, ap.Margin, img)
	}
	if err != nil {
		panic("ah")
	}
	ap.EndSyncMode()
}

// discs are filled
func drawDiscs(ap *ansipixels.AnsiPixels, clicks map[[2]int]int, colors map[[2]int]string, orderedByChosen *[][2]int, img *image.RGBA) {
	// img := image.NewRGBA(image.Rect(0, 0, ap.W, ap.H*2))
	toDelete := 0
	for _, coords := range *orderedByChosen {
		val := clicks[coords]
		x, y := coords[0], coords[1]
		for radius := 0; radius < val; radius++ {
			for i := 0.; i < 2.*math.Pi; i += 2. * math.Pi / 365. {
				ex := .3 * float64(radius) * (math.Cos(i))
				ey := .3 * float64(radius) * (math.Sin(i))
				r, g, b := toRGB(colors[coords])
				rx := max((int(ex) + x), 0)
				ry := max((int(ey) + y), 0)
				if rx < 0 || ry < 0 {
					continue
				}
				(*img).Set(rx, ry, color.RGBA{uint8(r), uint8(g), uint8(b), 100})
			}
		}
		if float64(val)*.3 > float64(ap.H) {
			delete(clicks, coords)
			*orderedByChosen = (*orderedByChosen)[1:]
		}
	}
	*orderedByChosen = (*orderedByChosen)[toDelete:]
}
