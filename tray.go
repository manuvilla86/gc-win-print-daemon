package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"

	"github.com/fyne-io/systray"
	"printbridge/printer"
)

func runTray() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(iconGray())
	systray.SetTooltip("PrintBridge")

	mStatus := systray.AddMenuItem("Iniciando...", "")
	mStatus.Disable()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Salir", "Detener PrintBridge")

	go func() {
		for {
			updateTrayStatus(mStatus)
			time.Sleep(5 * time.Second)
		}
	}()

	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func onExit() {
	os.Exit(0)
}

func updateTrayStatus(mStatus *systray.MenuItem) {
	name, err := printer.Detect()
	if err != nil || name == "" {
		systray.SetIcon(iconYellow())
		systray.SetTooltip("PrintBridge — Sin impresora")
		mStatus.SetTitle("Sin impresora detectada")
	} else {
		systray.SetIcon(iconGreen())
		systray.SetTooltip("PrintBridge — " + name)
		mStatus.SetTitle(name)
	}
}

func circleIcon(r, g, b uint8) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			dx := float64(x) - 7.5
			dy := float64(y) - 7.5
			if dx*dx+dy*dy <= 36 {
				img.Set(x, y, color.RGBA{r, g, b, 255})
			}
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func iconGreen()  []byte { return circleIcon(34, 197, 94) }
func iconYellow() []byte { return circleIcon(234, 179, 8) }
func iconGray()   []byte { return circleIcon(156, 163, 175) }
