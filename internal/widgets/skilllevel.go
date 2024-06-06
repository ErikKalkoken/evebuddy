package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type SkillLevel struct {
	widget.BaseWidget
	dots           []*canvas.Image
	levelBlocked   *theme.ErrorThemedResource
	levelTrained   *theme.PrimaryThemedResource
	levelUnTrained *theme.DisabledResource
}

func NewSkillLevel() *SkillLevel {
	const s = 12
	size := fyne.Size{Width: s, Height: s}
	f := canvas.ImageFillContain
	untrainedIcon := theme.NewDisabledResource(theme.MediaStopIcon())
	dots := make([]*canvas.Image, 5)
	for i := range 5 {
		dot := canvas.NewImageFromResource(untrainedIcon)
		dot.FillMode = f
		dot.SetMinSize(size)
		dots[i] = dot
	}
	w := &SkillLevel{
		dots:           dots,
		levelBlocked:   theme.NewErrorThemedResource(theme.MediaStopIcon()),
		levelTrained:   theme.NewPrimaryThemedResource(theme.MediaStopIcon()),
		levelUnTrained: untrainedIcon,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *SkillLevel) Set(activeLevel, trainedLevel int) {
	for i := range 5 {
		y := w.dots[i]
		if activeLevel > i {
			y.Resource = w.levelTrained
		} else if trainedLevel > i {
			y.Resource = w.levelBlocked
		} else {
			y.Resource = w.levelUnTrained
		}
		y.Refresh()
	}
}

func (w *SkillLevel) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewHBox()
	for i := range 5 {
		c.Add(w.dots[i])
	}
	return widget.NewSimpleRenderer(c)
}
