package ui

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/unit"
)

// Run launches the Gio UI and blocks until the window closes.
func Run(state *AppState) error {
	if state == nil {
		state = NewState()
	}

	go func() {
		w := new(app.Window)
		w.Option(app.Title("JTAG Tool"), app.Size(unit.Dp(1024), unit.Dp(720)))
		ui := New(w, state)
		if err := ui.Run(); err != nil {
			log.Printf("ui: %v", err)
		}
		os.Exit(0)
	}()

	app.Main()
	return nil
}
