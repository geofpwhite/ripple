package main

import (
	"flag"
	"fmt"

	"fortio.org/terminal/ansipixels"
)

func main() {
	fpsFlag := flag.Float64("fps", 60., "change the fps")
	flag.Parse()
	ap := ansipixels.NewAnsiPixels(*fpsFlag)
	err := ap.GetSize()
	if err != nil {
		panic("can't get term size")
	}
	ap.MouseTrackingOn()
	ap.MouseClickOn()
	ap.GetSize()
	defer func() {
		ap.Restore()
		ap.MouseTrackingOff()
		ap.MouseClickOff()
		ap.ShowCursor()
		ap.MoveCursor(0, 0)
	}()

	err = ap.Open()
	if err != nil {
		panic("can't open")
	}
	clicks := make(map[[2]int]int)
	ap.StartSyncMode()
	ap.ClearScreen()
	ap.EndSyncMode()

	for {
		ap.MouseDecode()
		_, err := ap.ReadOrResizeOrSignalOnce()
		if err != nil {
			panic("can't read/resize/signal")
		}

		for key := range clicks {
			clicks[key]++
		}
		Draw(ap, clicks)
		if len(ap.Data) == 0 {
			continue
		}
		switch ap.Data[0] {
		case 'q':
			ap.Restore()
			return
		case 32:
			ap.Mx = int(ap.Data[1] - 32)
			ap.My = int(ap.Data[2] - 32)
			fmt.Println(ap.Mx, ap.My)
			clicks[[2]int{ap.Mx, ap.My}] = 0
		default:
		}
		if ap.LeftClick() {
			fmt.Println(ap.Mx, ap.My, ap.Mbuttons)
		}
	}
}

func Draw(ap *ansipixels.AnsiPixels, clicks map[[2]int]int) {
	ap.StartSyncMode()
	ap.ClearScreen()
	ap.GetSize()
	for i := range ap.W {
		for j := range ap.H {
			for coords, radius := range clicks {
				if distance := (i-coords[0])*(i-coords[0]) + (j-coords[1])*(j-coords[1]); float64(distance) >= float64(radius)-(float64(radius)/20.) && float64(distance) <= float64(radius)+(float64(radius)/20.) {
					ap.WriteAtStr(i, j, string(ansipixels.FullPixel))
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
