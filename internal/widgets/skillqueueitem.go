package widgets

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type SkillQueueItem struct {
	widget.BaseWidget
	duration   *widget.Label
	progress   *widget.ProgressBar
	name       *widget.Label
	skillLevel *SkillLevel
}

func NewSkillQueueItem() *SkillQueueItem {
	w := &SkillQueueItem{
		duration:   widget.NewLabel("duration"),
		name:       widget.NewLabel("skill"),
		progress:   widget.NewProgressBar(),
		skillLevel: NewSkillLevel(),
	}
	w.progress.Hide()
	w.ExtendBaseWidget(w)
	return w
}

func (w *SkillQueueItem) Set(name string, targetLevel int, isActive bool, remaining, duration optional.Duration, completionP float64) {
	var (
		i widget.Importance
		d string
	)
	isCompleted := completionP == 1
	if isCompleted {
		i = widget.LowImportance
		d = "Completed"
	} else if isActive {
		i = widget.MediumImportance
		d = ihumanize.OptionalDuration(remaining, "?")
	} else {
		i = widget.MediumImportance
		d = ihumanize.OptionalDuration(duration, "?")
	}
	w.name.Importance = i
	w.name.Text = fmt.Sprintf("%s %s", name, ihumanize.ToRomanLetter(targetLevel))
	w.name.Refresh()
	w.duration.Text = d
	w.duration.Importance = i
	w.duration.Refresh()
	if isActive {
		w.progress.SetValue(completionP)
		w.progress.Show()
	} else {
		w.progress.Hide()
	}
	var active, trained, required int
	if isCompleted {
		active = targetLevel
		trained = targetLevel
		required = targetLevel
	} else if isActive {
		active = targetLevel - 1
		trained = targetLevel - 1
		required = 0
	} else {
		active = targetLevel - 1
		trained = targetLevel - 1
		required = targetLevel
	}
	w.skillLevel.Set(active, trained, required)
}

func (w *SkillQueueItem) SetError(message string, err error) {
	w.name.Text = fmt.Sprintf("%s: %s", message, ihumanize.Error(err))
	w.name.Importance = widget.DangerImportance
	w.name.Refresh()
	w.duration.SetText("")
	w.progress.Hide()
}

func (w *SkillQueueItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		nil, nil, w.skillLevel, nil,
		container.NewStack(w.progress, container.NewHBox(w.name, layout.NewSpacer(), w.duration)))
	return widget.NewSimpleRenderer(c)
}
