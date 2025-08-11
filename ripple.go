package main

import (
	"context"
	"flag"
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"runtime"

	"fortio.org/terminal/ansipixels"
	"fortio.org/terminal/ansipixels/tcolor"
)

func randomColor() tcolor.RGBColor {
	return tcolor.HSLToRGB(rand.Float64(), 0.5, 0.5)
}

type coords [2]int

type click struct {
	rightClick bool
	coords     coords
	color      tcolor.RGBColor
	timeAlive  uint64
}

type state struct {
	AP     *ansipixels.AnsiPixels
	clicks []click
	clock  uint64
	filled bool
}

func main() {
	// Force truecolor on windows for now.
	defaultTrueColor := (runtime.GOOS == "windows") || ansipixels.DetectColorMode().TrueColor
	truecolor := flag.Bool("truecolor", defaultTrueColor, "Use 24 bit colors")
	filled := flag.Bool("fill", false, "set this to not clear a bubble when it dies")
	fpsFlag := flag.Float64("fps", 30., "change the fps") // high fps makes it look super smooth
	flag.Parse()
	ap := ansipixels.NewAnsiPixels(*fpsFlag)
	err := ap.Open()
	var hasClicked bool
	if err != nil {
		panic("can't open")
	}
	defer func() {
		ap.MouseClickOff()
		ap.ShowCursor()
		ap.Restore()
	}()

	ap.TrueColor = *truecolor
	ap.MouseClickOn()
	ap.ClearScreen()
	ap.HideCursor()



	s := &state{
		AP:     ap,
		filled: *filled,
	}
	img := image.NewRGBA(image.Rect(0, 0, ap.W, ap.H*2))
	ap.OnResize = func() error {
		img = image.NewRGBA(image.Rect(0, 0, ap.W, ap.H*2))
		return nil
	}
	drawCircle := s.drawFilledCircle
	if s.filled {
		drawCircle = s.drawCircle
	}

	paused := false
	err = ap.FPSTicks(context.Background(), func(ctx context.Context) bool {
		if !s.filled {
			clear(img.Pix)
		}
		if paused {
			return true
		}
		s.clock++
		var possibleClick click

		switch {
		case ap.RightClick():
			possibleClick.rightClick = true
			fallthrough
		case ap.LeftClick():
			possibleClick.timeAlive = s.clock
			possibleClick.coords = coords{ap.Mx, ap.My * 2}
			possibleClick.color = randomColor()
			s.clicks = append(s.clicks, possibleClick)
			hasClicked = true
		case !hasClicked:
			return true
		}

		for _, click := range s.clicks {
			if click.rightClick {
				drawCircle(click, img)
			} else {
				s.drawDisc(click, img)
			}
		}

		s.drawCircles(img)

		if len(ap.Data) == 0 {

			return true
		}
		switch ap.Data[0] {
		case ' ':
			paused = !paused
		case 'c':
			s.clicks = []click{}

			clear(img.Pix)
		case 'q':
			return false
		}
		return true
	})
	if err != nil {
		panic(err)
	}
}

// circles are hollow
func (s *state) drawCircles(img *image.RGBA) {
	toDelete := 0
	// var draw = s.drawCircle
	// if filled {
	// 	draw = s.drawFilledCircle
	// }
	for _, click := range s.clicks {
		// draw(click, img)
		if float64(s.clock-click.timeAlive)*.3 > float64(s.AP.H) {
			toDelete++
		} else {
			break
		}
	}
	s.clicks = s.clicks[toDelete:]

	err := s.AP.ShowScaledImage(img)
	if err != nil {
		panic("ah")
	}
}

func (s *state) DrawColor(c tcolor.RGBColor) color.RGBA {
	if s.filled {
		return color.RGBA{c.R, c.G, c.B, 100}
	}
	return ansipixels.NRGBAtoRGBA(color.NRGBA{c.R, c.G, c.B, 85})
}

func (s *state) AddPixel(img *image.RGBA, x, y int, color color.RGBA) {
	if s.filled {
		img.SetRGBA(x, y, color)
	} else {
		ansipixels.AddPixel(img, x, y, color)
	}
}

func (s *state) drawCircle(click click, img *image.RGBA) {
	color := s.DrawColor(click.color)
	radius := s.clock - click.timeAlive
	for i := 0.; i < 2.*math.Pi; i += math.Pi / (float64(radius)) { // tbd
		ex := .3 * float64(radius) * (math.Cos(i))
		ey := .3 * float64(radius) * (math.Sin(i))
		rx := max((int(ex) + click.coords[0]), 0)
		ry := max((int(ey) + click.coords[1]), 0)
		if rx < 0 || ry < 0 {
			continue
		}
		s.AddPixel(img, rx, ry, color)
	}
}

func (s *state) circleBounds(click click, img *image.RGBA) map[int]*coords {
	bounds := make(map[int]*coords)
	radius := s.clock - click.timeAlive
	for i := 0.; i < math.Pi; i += math.Pi / (float64(radius)) { // tbd
		ex := .3 * float64(radius) * (math.Cos(i))
		ey := .3 * float64(radius) * (math.Sin(2*math.Pi - i))
		eyUpper := .3 * float64(radius) * (math.Sin(i))
		rx, ry := int(ex)+click.coords[0], int(ey)+click.coords[1]
		ryUpper := int(eyUpper) + click.coords[1]
		if rx > img.Bounds().Dx() {
			rx = img.Bounds().Dx()
		}
		if ryUpper > img.Bounds().Dy() {
			ryUpper = img.Bounds().Dy()
		}
		bounds[rx] = &coords{ry, ryUpper}
	}
	return bounds
}
func (s *state) drawDisc(click click, img *image.RGBA) {
	bounds := s.circleBounds(click, img)
	color := s.DrawColor(click.color)
	for x, yBounds := range bounds {
		for yValue := yBounds[0]; yValue < yBounds[1]; yValue++ {
			s.AddPixel(img, x, yValue, color)
		}
	}
	s.drawCircle(click, img)
}

func (s *state) drawFilledCircle(click click, img *image.RGBA) {
	bounds := s.circleBounds(click, img)
	color := color.RGBA{0, 0, 0, 0}
	for x, yBounds := range bounds {
		for yValue := yBounds[0]; yValue < yBounds[1]; yValue++ {
			img.Set(x, yValue, color)
		}
	}
	s.drawCircle(click, img)
}
