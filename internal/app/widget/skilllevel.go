package widget

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	iwidgets "github.com/ErikKalkoken/evebuddy/internal/widgets"
)

// SkillLevel shows the skill level status for a character.
// Which level is currently active, which level is trained, but disabled.
// It can also show which level is required.
type SkillLevel struct {
	widget.BaseWidget
	dots          []*canvas.Image
	levelBlocked  fyne.Resource
	levelRequired fyne.Resource
	levelTrained  fyne.Resource
	levelDisabled fyne.Resource
}

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
		dots:          dots,
		levelBlocked:  theme.NewWarningThemedResource(theme.MediaStopIcon()),
		levelRequired: theme.NewPrimaryThemedResource(theme.MediaStopIcon()),
		levelTrained:  theme.NewThemedResource(theme.MediaStopIcon()),
		levelDisabled: untrainedIcon,
	}
	w.ExtendBaseWidget(w)
	return w
}

// Set updates the widget to show a skill level.
// requiredLevel is optional and will be ignored when zero valued
func (w *SkillLevel) Set(activeLevel, trainedLevel, requiredLevel int) {
	for i := range 5 {
		y := w.dots[i]
		if activeLevel > i {
			y.Resource = w.levelTrained
		} else if trainedLevel > i {
			y.Resource = w.levelBlocked
		} else if requiredLevel > i {
			y.Resource = w.levelRequired
		} else {
			y.Resource = w.levelDisabled
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
