package awidget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	iwidgets "github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// SkillLevel shows the skill level status for a character.
// Which level is currently active, which level is trained, but disabled.
// It can also show which level is required.
type SkillLevel struct {
	widget.BaseWidget

	blocked  fyne.Resource
	disabled fyne.Resource
	dots     []*canvas.Image
	queued   fyne.Resource
	trained  fyne.Resource
}

// NewSkillLevel returns a new SkillLevel widget.
func NewSkillLevel() *SkillLevel {
	const s = 12
	size := fyne.Size{Width: s, Height: s}
	untrainedIcon := theme.NewDisabledResource(theme.MediaStopIcon())
	dots := make([]*canvas.Image, 5)
	for i := range 5 {
		dot := iwidgets.NewImageFromResource(untrainedIcon, size)
		dots[i] = dot
	}
	w := &SkillLevel{
		blocked:  theme.NewWarningThemedResource(theme.MediaStopIcon()),
		disabled: untrainedIcon,
		dots:     dots,
		queued:   theme.NewPrimaryThemedResource(theme.MediaStopIcon()),
		trained:  theme.NewThemedResource(theme.MediaStopIcon()),
	}
	w.ExtendBaseWidget(w)
	return w
}

// Set updates the widget to show a skill level.
// queued is optional and will be ignored when zero.
func (w *SkillLevel) Set(active, trained, queued int64) {
	for i := range int64(5) {
		y := w.dots[i]
		if active > i {
			y.Resource = w.trained
		} else if trained > i {
			y.Resource = w.blocked
		} else if queued > i {
			y.Resource = w.queued
		} else {
			y.Resource = w.disabled
		}
		y.Refresh()
	}
}

func (w *SkillLevel) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewHBox()
	for i := range 5 {
		c.Add(w.dots[i])
	}
	return widget.NewSimpleRenderer(container.NewPadded(c))
}
