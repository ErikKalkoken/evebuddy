package xwidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ButtonWithProgressIndicator represents a button widget with a progress indicator
type ButtonWithProgressIndicator struct {
	widget.BaseWidget

	isDisabled bool
	button     *widget.Button
	progress   *widget.ProgressBarInfinite
}

// NewButtonWithProgress creates a new button with a progress indicator.
//
// Note that action will be run in a goroutine.
// Make use to use fyne.Do when accessing the Fyne API
func NewButtonWithProgress(label string, icon fyne.Resource, action func()) *ButtonWithProgressIndicator {
	w := &ButtonWithProgressIndicator{
		button:   widget.NewButtonWithIcon(label, icon, nil),
		progress: widget.NewProgressBarInfinite(),
	}
	w.ExtendBaseWidget(w)
	w.progress.Hide()
	w.progress.Stop()
	w.button.OnTapped = func() {
		w.button.Disable()
		w.progress.Show()
		w.progress.Start()
		go func() {
			action()
			fyne.Do(func() {
				w.progress.Stop()
				w.progress.Hide()
				w.progress.Stop()
				if !w.isDisabled {
					w.button.Enable()
				}
			})
		}()
	}
	return w
}

func (w *ButtonWithProgressIndicator) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(w.button, w.progress)
	return widget.NewSimpleRenderer(c)
}

// SetImportance sets the importance of the button.
func (w *ButtonWithProgressIndicator) SetImportance(v widget.Importance) {
	w.button.Importance = v
	w.button.Refresh()
}

// SetText sets the text of the button.
func (w *ButtonWithProgressIndicator) SetText(text string) {
	w.button.SetText(text)
}

// SetIcon sets the icon of the button.
func (w *ButtonWithProgressIndicator) SetIcon(icon fyne.Resource) {
	w.button.SetIcon(icon)
}

// Disabled reports whether this widget is disabled.
func (w *ButtonWithProgressIndicator) Disabled() bool {
	return w.isDisabled
}

// Disable disables this widget.
func (w *ButtonWithProgressIndicator) Disable() {
	w.isDisabled = true
	w.button.Disable()
}

// Enable enables this widget.
func (w *ButtonWithProgressIndicator) Enable() {
	w.isDisabled = false
	w.button.Enable()
}
