package main

import (
	"context"
	"flag"
	"image"
	"image/color"
	"image/draw"
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
	fpsFlag := flag.Float64("fps", 60., "change the fps") // high fps makes it look super smooth
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
	drawDisc := s.drawFilledDisc
	if !s.filled {
		drawCircle = s.drawCircle
		drawDisc = s.drawDisc
	}
	draw.Draw(img, image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()), &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.Point{}, draw.Over)
	ap.WriteCentered(ap.H/2, "Left click to make a bubble appear. right click to make just the outline appear")
	paused := false
	err = ap.FPSTicks(context.Background(), func(ctx context.Context) bool {
		if !s.filled {
			clear(img.Pix)
			draw.Draw(img, image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()), &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.Point{}, draw.Over)
		}
		if !paused {
			s.clock++
		}
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
				drawDisc(click, img)
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

	for _, click := range s.clicks {
		if float64(s.clock-click.timeAlive)*.25 > float64(s.AP.H) {
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

func (s *state) drawCircle(click click, img *image.RGBA) {
	color := s.DrawColor(click.color)
	radius := s.clock - click.timeAlive
	for i := 0.; i < 2.*math.Pi; i += math.Pi / (float64(radius)) { // tbd
		ex := .25 * float64(radius) * (math.Cos(i))
		ey := .25 * float64(radius) * (math.Sin(i))
		rx := (int(ex) + click.coords[0])
		ry := (int(ey) + click.coords[1])
		if rx < 0 || ry < 0 {
			continue
		}
		ansipixels.AddPixel(img, rx, ry, color)
	}
}

func (s *state) circleBounds(click click, img *image.RGBA) map[int]*coords {
	bounds := make(map[int]*coords)
	radius := s.clock - click.timeAlive
	for i := 0.; i < math.Pi; i += math.Pi / (float64(radius)) { // tbd
		ex := .25 * float64(radius) * (math.Cos(i))
		ey := .25 * float64(radius) * (math.Sin(2*math.Pi - i))
		eyUpper := .25 * float64(radius) * (math.Sin(i))
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
func (s *state) filledCircleBounds(click click, img *image.RGBA) map[int][2]coords {
	bounds := make(map[int][2]coords)
	click.timeAlive--
	prevBounds := s.circleBounds(click, img)
	click.timeAlive++
	radius := s.clock - click.timeAlive
	for i := 0.; i < math.Pi; i += math.Pi / (float64(radius)) {

		ex := .25 * float64(radius) * (math.Cos(i))
		ey := .25 * float64(radius) * (math.Sin(2*math.Pi - i))
		eyUpper := .25 * float64(radius) * (math.Sin(i))
		rx, ry := int(ex)+click.coords[0], int(ey)+click.coords[1]
		ryUpper := int(eyUpper) + click.coords[1]
		if rx > img.Bounds().Dx() {
			rx = img.Bounds().Dx()
		}
		if ryUpper > img.Bounds().Dy() {
			ryUpper = img.Bounds().Dy()
		}
		var lowerUntouched, upperUntouched coords
		if prevBounds[rx] == nil {
			lowerUntouched = coords{ry, ryUpper}
			upperUntouched = coords{0, 0}
		} else {
			lowerUntouched = coords{ry, prevBounds[rx][0]}
			upperUntouched = coords{prevBounds[rx][1], ryUpper}
		}
		bounds[rx] = ([2]coords{lowerUntouched, upperUntouched})
	}
	return bounds
}
func (s *state) drawDisc(click click, img *image.RGBA) {
	bounds := s.circleBounds(click, img)
	color := s.DrawColor(click.color)
	for x, yBounds := range bounds {
		for yValue := yBounds[0]; yValue < yBounds[1]; yValue++ {
			ansipixels.AddPixel(img, x, yValue, color)
		}
	}
}

// discs are always filled in, but we must not redraw over the same cell twice
func (s *state) drawFilledDisc(click click, img *image.RGBA) {
	bounds := s.filledCircleBounds(click, img)
	color := s.DrawColor(click.color)
	for x, yBounds := range bounds {
		draw.Draw(img, image.Rect(x, yBounds[0][0], x+1, yBounds[0][1]), &image.Uniform{color}, image.Point{}, draw.Over)
		draw.Draw(img, image.Rect(x, yBounds[1][0], x+1, yBounds[1][1]), &image.Uniform{color}, image.Point{}, draw.Over)
	}
}

func (s *state) drawFilledCircle(click click, img *image.RGBA) {
	bounds := s.circleBounds(click, img)
	color := color.RGBA{0, 0, 0, 255}
	for x, yBounds := range bounds {
		for yValue := yBounds[0]; yValue < yBounds[1]; yValue++ {
			img.Set(x, yValue, color)
		}
	}
	s.drawCircle(click, img)
}
