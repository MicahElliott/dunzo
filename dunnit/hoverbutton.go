package dun

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// hoverButton is a widget.Button that also shows a small tooltip
// popup (its own fyne.CanvasObject, not a native OS tooltip -- Fyne
// has no built-in tooltip widget) after the mouse hovers over it for
// a short delay. Used where a button's label is just an emoji/icon
// and needs a text hint (e.g. Daybook's Discard/Postpone/Done
// actions).
type hoverButton struct {
	widget.Button
	tooltip string

	popup      *widget.PopUp
	hoverTimer *time.Timer
}

// newHoverButton creates a hoverButton with the given label (shown on
// the button itself, typically just an emoji) and tooltip text (shown
// after a brief hover).
func newHoverButton(label, tooltip string, tapped func()) *hoverButton {
	b := &hoverButton{tooltip: tooltip}
	b.Text = label
	b.OnTapped = tapped
	b.ExtendBaseWidget(b)
	return b
}

const hoverButtonTooltipDelay = 400 * time.Millisecond

func (b *hoverButton) showTooltip() {
	if b.popup != nil {
		return
	}
	canvas := fyne.CurrentApp().Driver().CanvasForObject(b)
	if canvas == nil {
		return
	}
	label := widget.NewLabel(b.tooltip)
	b.popup = widget.NewPopUp(label, canvas)
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(b)
	b.popup.ShowAtPosition(pos.Add(fyne.NewPos(0, b.Size().Height)))
}

func (b *hoverButton) hideTooltip() {
	if b.hoverTimer != nil {
		b.hoverTimer.Stop()
		b.hoverTimer = nil
	}
	if b.popup != nil {
		b.popup.Hide()
		b.popup = nil
	}
}

// MouseIn/MouseMoved/MouseOut implement desktop.Hoverable.
func (b *hoverButton) MouseIn(*desktop.MouseEvent) {
	b.hoverTimer = time.AfterFunc(hoverButtonTooltipDelay, func() {
		fyne.Do(b.showTooltip)
	})
}

func (b *hoverButton) MouseMoved(*desktop.MouseEvent) {}

func (b *hoverButton) MouseOut() {
	b.hideTooltip()
}

var _ desktop.Hoverable = (*hoverButton)(nil)
