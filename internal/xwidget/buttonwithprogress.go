package xwidget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// ProgressButton represents a button widget which shows a progress indicator.
type ProgressButton struct {
	widget.BaseWidget

	button   *lockableButton
	progress *widget.Activity
	spacer   *canvas.Rectangle
	label    string
	icon     fyne.Resource
}

// NewProgressButton creates a new button with a progress indicator.
//
// The action will be run in a goroutine.
// Note that the button remains clickable
func NewProgressButton(label string, icon fyne.Resource, action func()) *ProgressButton {
	w := &ProgressButton{
		button:   newLockableButton(label, icon, nil),
		progress: widget.NewActivity(),
		spacer:   canvas.NewRectangle(color.Transparent),
		label:    label,
		icon:     icon,
	}
	w.ExtendBaseWidget(w)
	w.progress.Hide()
	w.progress.Stop()
	w.button.OnTapped = func() {
		w.button.disable()
		// clear button
		w.spacer.SetMinSize(w.button.Size())
		w.button.Text = ""
		w.button.Icon = nil
		w.button.Refresh()
		// show progress
		w.progress.Show()
		w.progress.Start()
		go func() {
			action()
			fyne.Do(func() {
				// restore button
				w.button.Text = label
				w.button.Icon = icon
				w.button.Refresh()
				w.spacer.SetMinSize(fyne.Size{})
				// hide progress
				w.progress.Stop()
				w.progress.Hide()
				w.progress.Stop()
				w.button.enable()
			})
		}()
	}
	return w
}

func (w *ProgressButton) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(w.spacer, w.button, w.progress)
	return widget.NewSimpleRenderer(c)
}

// SetImportance sets the importance of the button.
func (w *ProgressButton) SetImportance(v widget.Importance) {
	w.button.Importance = v
	w.button.Refresh()
}

// SetText sets the text of the button.
func (w *ProgressButton) SetText(text string) {
	w.button.SetText(text)
}

// SetIcon sets the icon of the button.
func (w *ProgressButton) SetIcon(icon fyne.Resource) {
	w.button.SetIcon(icon)
}

// Disabled reports whether this widget is disabled.
func (w *ProgressButton) Disabled() bool {
	return w.button.Disabled()
}

// Disable disables this widget.
func (w *ProgressButton) Disable() {
	w.button.Disable()
}

// Enable enables this widget.
func (w *ProgressButton) Enable() {
	w.button.Enable()
}

// lockableButton is an extension of the Fyne button which can be fully disabled
// without changing it's appearance.
//
// This feature is used by the ProgressButton.
type lockableButton struct {
	widget.Button
	disabled bool
}

func newLockableButton(label string, icon fyne.Resource, tapped func()) *lockableButton {
	w := &lockableButton{
		Button: widget.Button{
			Text:     label,
			Icon:     icon,
			OnTapped: tapped,
		},
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *lockableButton) enable() {
	w.disabled = false
}

func (w *lockableButton) disable() {
	w.disabled = true
}

func (w *lockableButton) Tapped(pe *fyne.PointEvent) {
	if w.disabled {
		return
	}
	w.Button.Tapped(pe)
}

func (w *lockableButton) MouseIn(me *desktop.MouseEvent) {
	if w.disabled {
		return
	}
	w.Button.MouseIn(me)
}

func (w *lockableButton) MouseOut() {
	if w.disabled {
		return
	}
	w.Button.MouseOut()

}
