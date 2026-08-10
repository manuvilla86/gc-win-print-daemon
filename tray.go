package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"

	"fyne.io/systray"
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

func solidIcon(r, g, b uint8) []byte {
	const size = 32
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	c := color.RGBA{r, g, b, 255}
	for x := 0; x < size; x++ {
		for y := 0; y < size; y++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func iconGreen()  []byte { return solidIcon(34, 197, 94) }
func iconYellow() []byte { return solidIcon(234, 179, 8) }
func iconGray()   []byte { return solidIcon(156, 163, 175) }
