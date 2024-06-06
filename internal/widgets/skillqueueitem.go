package widgets

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/helper/mytypes"
)

type SkillQueueItem struct {
	widget.BaseWidget
	duration *widget.Label
	progress *widget.ProgressBar
	name     *widget.Label
}

func NewSkillQueueItem() *SkillQueueItem {
	w := &SkillQueueItem{
		duration: widget.NewLabel("duration"),
		name:     widget.NewLabel("skill"),
		progress: widget.NewProgressBar(),
	}
	w.progress.Hide()
	w.ExtendBaseWidget(w)
	return w
}

func (w *SkillQueueItem) Set(name string, level int, isActive bool, remaining, duration mytypes.OptionalDuration, completionP float64) {
	var i widget.Importance
	var d string
	if completionP == 1 {
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
	w.name.Text = fmt.Sprintf("%s %s", name, ihumanize.ToRomanLetter(level))
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
}

func (w *SkillQueueItem) SetError(message string, err error) {
	w.name.Text = fmt.Sprintf("%s: %s", message, ihumanize.Error(err))
	w.name.Importance = widget.DangerImportance
	w.name.Refresh()
	w.duration.SetText("")
	w.progress.Hide()
}

func (w *SkillQueueItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewStack(
		w.progress, container.NewHBox(w.name, layout.NewSpacer(), w.duration))
	return widget.NewSimpleRenderer(c)
}
