package xwidget

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ProgressButton represents a button widget which shows a progress indicator.
type ProgressButton struct {
	widget.BaseWidget

	isDisabled bool
	button     *widget.Button
	progress   *widget.Activity
	spacer     *canvas.Rectangle
	label      string
	icon       fyne.Resource
}

// NewProgressButton creates a new button with a progress indicator.
//
// Note that action will be run in a goroutine.
// Make use to use fyne.Do when accessing the Fyne API
func NewProgressButton(label string, icon fyne.Resource, action func()) *ProgressButton {
	w := &ProgressButton{
		button:   widget.NewButtonWithIcon(label, icon, nil),
		progress: widget.NewActivity(),
		spacer:   canvas.NewRectangle(color.Transparent),
		label:    label,
		icon:     icon,
	}
	w.ExtendBaseWidget(w)
	w.progress.Hide()
	w.progress.Stop()
	w.button.OnTapped = func() {
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
	return w.isDisabled
}

// Disable disables this widget.
func (w *ProgressButton) Disable() {
	w.isDisabled = true
	w.button.Disable()
}

// Enable enables this widget.
func (w *ProgressButton) Enable() {
	w.isDisabled = false
	w.button.Enable()
}
