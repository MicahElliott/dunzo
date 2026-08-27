package dun

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

func SetThings() {
	fmt.Println("Setting things")
}

// showSettings pops up a placeholder settings window with a single
// dummy boolean preference. Real prefs (schedule times, DUNZO_DIR,
// etc.) can be added here later.
func showSettings(a fyne.App) {
	w := a.NewWindow("Dunzo Settings")

	dummyPref := widget.NewCheck("Enable dummy setting", func(checked bool) {
		fmt.Println("dummy setting changed:", checked)
	})

	w.SetContent(widget.NewForm(
		widget.NewFormItem("Placeholder", dummyPref),
	))
	w.Resize(fyne.NewSize(300, 100))
	w.Show()
}
