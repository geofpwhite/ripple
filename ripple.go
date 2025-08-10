package main

import (
	"context"
	"flag"
	"image"
	"image/color"
	"math"
	"math/rand/v2"

	"fortio.org/terminal/ansipixels"
	"fortio.org/terminal/ansipixels/tcolor"
)

func randomColor() tcolor.RGBColor {
	return tcolor.HSLToRGB(rand.Float64(), 0.75, 0.6)
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
}

func main() {
	filled := flag.Bool("fill", false, "set this to not clear a bubble when it dies")
	fpsFlag := flag.Float64("fps", 100., "change the fps") // high fps makes it look super smooth
	flag.Parse()
	ap := ansipixels.NewAnsiPixels(*fpsFlag)
	err := ap.Open()
	if err != nil {
		panic("can't open")
	}
	defer func() {
		ap.MouseTrackingOff()
		ap.ShowCursor()
		ap.Restore()
	}()

	paused := false
	ap.MouseClickOn()
	ap.MouseTrackingOn()
	s := &state{
		AP: ap,
	}
	ap.ClearScreen()
	// ap.HideCursor()
	img := image.NewRGBA(image.Rect(0, 0, ap.W, ap.H*2))
	ap.OnResize = func() error {
		img = image.NewRGBA(image.Rect(0, 0, ap.W, ap.H*2))
		return nil
	}
	drawCircle := s.drawFilledCircle
	if !*filled {
		drawCircle = s.drawCircle
	}
	ap.FPSTicks(context.TODO(), func(ctx context.Context) bool {
		if !*filled {
			clear(img.Pix)
		}
		// _, err := ap.ReadOrResizeOrSignalOnce()
		// if err != nil {
		// 	panic("can't read/resize/signal")
		// }
		if !paused {
			s.clock++
		}
		coords := coords{ap.Mx, ap.My * 2}
		rgbColor := randomColor()
		possibleClick := click{true, coords, rgbColor, s.clock}

		if ap.LeftClick() {
			possibleClick := possibleClick
			possibleClick.rightClick = false
			s.clicks = append(s.clicks, possibleClick)
		}
		if ap.RightClick() {
			s.clicks = append(s.clicks, possibleClick)
		}
		for _, click := range s.clicks {
			switch click.rightClick {
			case true:
				drawCircle(click, img)
			default:
				s.drawDisc(click, img)
			}
		}

		// s.drawDiscs(img)
		s.drawCircles(img)
		ap.MoveCursor(ap.Mx-1, ap.My-1)

		if len(ap.Data) != 1 { // if i do len(ap.Data) == 0 it seems like sometimes it reads a mouse input/click as a pause.
			return true
		}
		switch ap.Data[0] {
		case ' ':
			paused = !paused
		case 'c':
			s.clicks = []click{}
			// ap.ClearScreen()
			clear(img.Pix)
		case 'q':
			return false
		}
		return true
	})
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
	var err error
	switch {
	case s.AP.TrueColor:
		err = s.AP.DrawTrueColorImage(s.AP.Margin, s.AP.Margin, img)
	case s.AP.Color:
		err = s.AP.Draw216ColorImage(s.AP.Margin, s.AP.Margin, img)
	default:
		err = s.AP.Draw216ColorImage(s.AP.Margin, s.AP.Margin, img)
		// s.AP.ShowImage(&ansipixels.Image{Images: []*image.RGBA{img}}, 0., 0, 0, "")
	}
	if err != nil {
		panic("ah")
	}
	s.AP.MoveCursor(s.AP.Mx-1, s.AP.My-1)
}

func (s *state) drawCircle(click click, img *image.RGBA) {
	rgbColor := click.color
	color := color.RGBA{rgbColor.R, rgbColor.G, rgbColor.B, 100}
	radius := s.clock - click.timeAlive
	for i := 0.; i < 2.*math.Pi; i += math.Pi / (float64(radius)) { // tbd
		ex := .3 * float64(radius) * (math.Cos(i))
		ey := .3 * float64(radius) * (math.Sin(i))
		rx := max((int(ex) + click.coords[0]), 0)
		ry := max((int(ey) + click.coords[1]), 0)
		if rx < 0 || ry < 0 {
			continue
		}
		img.Set(rx, ry, color)
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
		switch {
		case rx > img.Bounds().Dx():
			rx = img.Bounds().Dx()
			fallthrough
		case ryUpper > img.Bounds().Dy():
			ryUpper = img.Bounds().Dy()
		}
		bounds[rx] = &coords{ry, ryUpper}
	}
	return bounds
}
func (s *state) drawDisc(click click, img *image.RGBA) {
	bounds := s.circleBounds(click, img)
	color := color.RGBA{click.color.R, click.color.G, click.color.B, 100}
	for x, yBounds := range bounds {
		for yValue := yBounds[0]; yValue < yBounds[1]; yValue++ {
			img.Set(x, yValue, color)
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
