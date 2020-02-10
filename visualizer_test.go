package visual

import (
	"flag"
	"testing"
	"time"

	"github.com/faiface/pixel"
	"golang.org/x/image/colornames"
)

func TestMain(*testing.M) {
	{
		var bFlagHeadless = false
		flag.BoolVar(&bFlagHeadless, "headless", false, "If set to true, the test runs in non-windowed mode.")
		flag.Parse()
		if bFlagHeadless {
			return
		}
	}

	visualizer := NewVisualizer(
		Config{
			Bg:                  pixel.ToRGBA(colornames.Coral),
			OnPaused:            nil,
			OnResumed:           nil,
			OnDrawn:             nil,
			OnUpdated:           nil,
			OnResized:           nil,
			OnClose:             nil,
			OnHandlingEvents:    nil,
			OnLogging:           nil,
			WinCentered:         false,
			Undecorated:         false,
			Title:               "testing visualizer",
			Version:             "undefined",
			Width:               60000.0,
			Height:              20000.0,
			WinWidth:            900.0,
			WinHeight:           600.0,
			InitialZoomLevel:    -1.0,
			InitialRotateDegree: -360.0,
		}, nil,
	)
	// 1
	go func() {
		time.Sleep(time.Second * 3)
		visualizer.Close()
	}()
	visualizer.Run()
	// 2
	go func() {
		time.Sleep(time.Second * 3)
		visualizer.Close()
	}()
	visualizer.Run()
}
