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

	OnAction func()

	button       *lockableButton
	disabledTemp bool
	icon         fyne.Resource
	label        string
	progress     *widget.Activity
	spacer       *canvas.Rectangle
}

// NewProgressButton creates a new button that shows a progress indicator
// while the button's action is in progress.
//
// Interactions are suspended while action is running
// and will be applied once it has completed.
//
// Note that action will run in a goroutine.
// Interactions with the Fyne API need to be wrapped in Fyne.Do or Fyne.DoAndWait.
func NewProgressButton(label string, icon fyne.Resource, action func()) *ProgressButton {
	w := &ProgressButton{
		button:   newLockableButton(label, icon, nil),
		progress: widget.NewActivity(),
		spacer:   canvas.NewRectangle(color.Transparent),
		label:    label,
		icon:     icon,
		OnAction: action,
	}
	w.ExtendBaseWidget(w)
	w.progress.Hide()
	w.progress.Stop()
	w.button.OnTapped = func() {
		if w.OnAction == nil {
			return
		}
		w.button.lock()
		// clear button
		w.spacer.SetMinSize(w.button.MinSize())
		w.button.Text = ""
		w.button.Icon = nil
		w.button.Refresh()
		// show progress
		w.progress.Show()
		w.progress.Start()
		action := w.OnAction
		go func() {
			defer func() {
				fyne.Do(func() {
					// restore button
					w.button.Text = w.label
					w.button.Icon = w.icon
					w.button.Refresh()
					if w.disabledTemp {
						w.button.Disable()
						w.disabledTemp = false
					}
					w.spacer.SetMinSize(fyne.Size{})
					// hide progress
					w.progress.Stop()
					w.progress.Hide()
					w.button.unlock()
				})
			}()
			action()
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
	if w.button.locked {
		return
	}
	w.button.Refresh()
}

// SetText sets the text of the button.
func (w *ProgressButton) SetText(label string) {
	w.label = label
	if w.button.locked {
		return
	}
	w.button.SetText(label)
}

// SetIcon sets the icon of the button.
func (w *ProgressButton) SetIcon(icon fyne.Resource) {
	w.icon = icon
	if w.button.locked {
		return
	}
	w.button.SetIcon(icon)
}

// Disabled reports whether this widget is disabled.
func (w *ProgressButton) Disabled() bool {
	return w.disabledTemp || w.button.Disabled()
}

// Disable disables this widget.
func (w *ProgressButton) Disable() {
	if w.button.locked {
		w.disabledTemp = true
		return
	}
	w.button.Disable()
}

// Enable enables this widget.
func (w *ProgressButton) Enable() {
	if w.button.locked {
		w.disabledTemp = false
		return
	}
	w.button.Enable()
}

// lockableButton is an extension of the Fyne button which can be
// disabled / locked without changing it's appearance.
//
// This feature is used by the ProgressButton.
type lockableButton struct {
	widget.Button
	locked bool
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

func (w *lockableButton) unlock() {
	w.locked = false
}

func (w *lockableButton) lock() {
	w.locked = true
}

func (w *lockableButton) Tapped(pe *fyne.PointEvent) {
	if w.locked {
		return
	}
	w.Button.Tapped(pe)
}

func (w *lockableButton) MouseIn(me *desktop.MouseEvent) {
	if w.locked {
		return
	}
	w.Button.MouseIn(me)
}

func (w *lockableButton) MouseOut() {
	w.Button.MouseOut()
}

func (w *lockableButton) TypedKey(key *fyne.KeyEvent) {
	if w.locked {
		return
	}
	w.Button.TypedKey(key)
}

func (w *lockableButton) TypedRune(r rune) {
	if w.locked {
		return
	}
	w.Button.TypedRune(r)
}

func (w *lockableButton) FocusGained() {
	if w.locked {
		return
	}
	w.Button.FocusGained()
}

func (w *lockableButton) FocusLost() {
	if w.locked {
		return
	}
	w.Button.FocusLost()
}
