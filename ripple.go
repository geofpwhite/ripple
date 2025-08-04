package main

import (
	"flag"
	"image"
	"image/color"
	"math"
	"math/rand/v2"

	"fortio.org/terminal/ansipixels"
	"fortio.org/terminal/ansipixels/tcolor"
)

var colorsToChoose = [...]tcolor.RGBColor{
	randomColor(),
	randomColor(),
}

func randomColor() tcolor.RGBColor {
	return tcolor.HSLToRGB(rand.Float64(), 0.75, 0.6)
}

type coords [2]int

type click struct {
	coords    coords
	color     tcolor.RGBColor
	timeAlive int
}

type state struct {
	AP          *ansipixels.AnsiPixels
	leftClicks  []click
	rightClicks []click
}

func main() {
	fpsFlag := flag.Float64("fps", 100., "change the fps") // high fps makes it look super smooth
	flag.Parse()
	colorCount := 0
	ap := ansipixels.NewAnsiPixels(*fpsFlag)
	err := ap.Open()
	if err != nil {
		panic("can't open")
	}
	defer func() {
		ap.MouseTrackingOff()
		ap.MoveCursor(0, ap.H-2)
		ap.ShowCursor()
		ap.Restore()
	}()

	paused := false
	ap.MouseClickOn()
	ap.HideCursor()
	s := &state{
		ap, []click{}, []click{},
	}
	ap.StartSyncMode()
	ap.ClearScreen()
	ap.EndSyncMode()
	ap.HideCursor()
	for {
		img := image.NewRGBA(image.Rect(0, 0, ap.W, ap.H*2))
		_, err := ap.ReadOrResizeOrSignalOnce()
		if err != nil {
			panic("can't read/resize/signal")
		}
		if !paused {
			for i := range s.leftClicks {
				s.leftClicks[i].timeAlive++
			}
			for i := range s.rightClicks {
				s.rightClicks[i].timeAlive++
			}
		}
		if ap.LeftClick() {
			coords := coords{ap.Mx, ap.My * 2}
			rgbColor := colorsToChoose[0]
			rgbColor.R = (rgbColor.R + uint8(colorCount)) % 255
			rgbColor.G = (rgbColor.G + uint8(colorCount)) % 255
			rgbColor.B = (rgbColor.B + uint8(colorCount)) % 255
			click := click{coords, rgbColor, 0}
			s.leftClicks = append(s.leftClicks, click)
		} else if ap.RightClick() {
			coords := coords{ap.Mx, ap.My * 2}
			rgbColor := colorsToChoose[1]
			rgbColor.R = (rgbColor.R + uint8(colorCount)) % 255
			rgbColor.G = (rgbColor.G + uint8(colorCount)) % 255
			rgbColor.B = (rgbColor.B + uint8(colorCount)) % 255
			click := click{coords, rgbColor, 0}
			s.rightClicks = append(s.rightClicks, click)
		}
		colorCount = (colorCount + 73) % 255

		s.drawDiscs(img)
		s.drawCircles(img)
		if len(ap.Data) == 0 {
			continue
		}
		switch ap.Data[0] {
		case ' ':
			paused = !paused
		case 'c':
			s.leftClicks = []click{}
			s.rightClicks = []click{}
			ap.ClearScreen()
		case 'q':
			return
		}
	}
}

// circles are hollow
func (s *state) drawCircles(img *image.RGBA) {
	toDelete := 0
	for _, click := range s.rightClicks {
		s.drawCircle(click, img)
		if float64(click.timeAlive)*.3 > float64(s.AP.H) {
			toDelete++
		}
	}
	s.rightClicks = s.rightClicks[toDelete:]
	s.AP.StartSyncMode()
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
	s.AP.EndSyncMode()
}

func (s *state) drawCircle(click click, img *image.RGBA) {
	rgbColor := click.color
	color := color.RGBA{rgbColor.R, rgbColor.G, rgbColor.B, 100}
	for i := 0.; i < 2.*math.Pi; i += 2. * math.Pi / (4. * float64(click.timeAlive)) { // tbd
		ex := .3 * float64(click.timeAlive) * (math.Cos(i))
		ey := .3 * float64(click.timeAlive) * (math.Sin(i))
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
	for i := 0.; i < 2.*math.Pi; i += 2. * math.Pi / (4. * float64(click.timeAlive)) { // tbd
		upper := false
		if i < math.Pi {
			upper = true
		}
		ex := .3 * float64(click.timeAlive) * (math.Cos(i))
		ey := .3 * float64(click.timeAlive) * (math.Sin(i))
		// rx := max((int(ex) + click.coords[0]), 0)
		// ry := max((int(ey) + click.coords[1]), 0)
		rx, ry := int(ex)+click.coords[0], int(ey)+click.coords[1]
		switch {
		case rx < 0:
			rx = 0
			fallthrough
		case ry < 0:
			ry = 0
			fallthrough
		case rx > img.Bounds().Dx():
			rx = img.Bounds().Dx()
			fallthrough
		case ry > img.Bounds().Dy():
			ry = img.Bounds().Dy()
		}
		if bounds[rx] == nil {
			bounds[rx] = &coords{}
		}
		if ry == click.coords[1] {
			(*bounds[rx]) = coords{ry, ry}
			continue
		}
		if upper {
			(*bounds[rx])[1] = ry
		} else {
			(*bounds[rx])[0] = ry
		}
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

// discs are filled
func (s *state) drawDiscs(img *image.RGBA) {
	// img := image.NewRGBA(image.Rect(0, 0, ap.W, ap.H*2))
	toDelete := 0
	for _, click := range s.leftClicks {
		val := click.timeAlive
		s.drawDisc(click, img)
		if float64(val)*.3 > float64(s.AP.H) {
			toDelete++
		}
	}
	s.leftClicks = s.leftClicks[toDelete:]
}
