// Command macos-docker-spy is a menu bar app showing the state of the Docker
// Desktop VM.
package main

import (
	"fyne.io/systray"
)

func main() {
	systray.Run(onReady, func() {})
}

func onReady() {
	// Not running: the bare whale, tinted by macOS like other menu bar icons.
	systray.SetTemplateIcon(whalePNG, whalePNG)
	systray.SetTooltip("Docker Desktop VM")

	quit := systray.AddMenuItem("Quit", "Quit macos-docker-spy")
	go func() {
		<-quit.ClickedCh
		systray.Quit()
	}()
}
