package main

import (
	"flag"
	"math/rand/v2"

	"fortio.org/terminal/ansipixels"
)

var colorsToChoose = [...]string{
	ansipixels.Red,
	ansipixels.Green,
	ansipixels.White,
	ansipixels.Blue,
	ansipixels.Purple,
	ansipixels.BrightCyan,
}

func main() {
	fpsFlag := flag.Float64("fps", 600., "change the fps") //high fps makes it look super smooth
	filled := flag.Bool("fill", false, "makes ripples filled in")
	flag.Parse()
	ap := ansipixels.NewAnsiPixels(*fpsFlag)
	err := ap.GetSize()
	paused := false
	if err != nil {
		panic("can't get term size")
	}
	ap.MouseTrackingOn()
	ap.MouseClickOn()
	defer func() {
		ap.Restore()
		ap.MouseTrackingOff()
		ap.MouseClickOff()
		ap.ShowCursor()
		ap.MoveCursor(0, 0)
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
		ap.MoveCursor(ap.Mx, ap.My)
		if !paused {
			for key := range clicks {
				clicks[key]++
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
		case 'q':
			ap.Restore()
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
			for coords, radius := range clicks {
				if distance := (i-coords[0])*(i-coords[0]) + ((j*2)-(coords[1]*2))*((j*2)-(2*coords[1])); float64(distance) >= float64(radius)-(float64(radius)/5.) && float64(distance) <= float64(radius)+(float64(radius)/5.) {
					ap.WriteAtStr(i, j, colors[coords]+string(ansipixels.FullPixel))
				}
			}
			if clicks[[2]int{i, j}] >= min(ap.W, ap.H)*min(ap.W, ap.H) {
				delete(clicks, [2]int{i, j})
			}
		}

	}
	// time.Sleep(10 * time.Millisecond)
	ap.EndSyncMode()

}
