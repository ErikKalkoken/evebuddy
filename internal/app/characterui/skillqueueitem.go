package characterui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
)

type SkillQueueItem struct {
	widget.BaseWidget

	duration   *widget.Label
	progress   *widget.ProgressBar
	name       *widget.Label
	skillLevel *appwidget.SkillLevel
	isMobile   bool
}

func NewSkillQueueItem() *SkillQueueItem {
	name := widget.NewLabel("skill")
	name.Truncation = fyne.TextTruncateEllipsis
	pb := widget.NewProgressBar()
	w := &SkillQueueItem{
		duration:   widget.NewLabel("duration"),
		name:       name,
		progress:   pb,
		skillLevel: appwidget.NewSkillLevel(),
		isMobile:   fyne.CurrentDevice().IsMobile(),
	}
	w.ExtendBaseWidget(w)
	pb.Hide()
	if w.isMobile {
		pb.TextFormatter = func() string {
			return ""
		}
	}
	return w
}

// func (w *SkillQueueItem) Set(isActive bool, remaining, duration optional.Optional[time.Duration], completionP float64) {

func (w *SkillQueueItem) Set(q *app.CharacterSkillqueueItem) {
	var (
		i widget.Importance
		d string
	)
	isActive := q.IsActive()
	completionP := q.CompletionP()
	isCompleted := completionP == 1
	if isCompleted {
		i = widget.LowImportance
		d = "Completed"
	} else if isActive {
		i = widget.MediumImportance
		d = ihumanize.Optional(q.Remaining(), "?")
	} else {
		i = widget.MediumImportance
		d = ihumanize.Optional(q.Duration(), "?")
	}
	w.name.Importance = i
	w.name.Text = q.String()
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
		active = q.FinishedLevel
		trained = q.FinishedLevel
		required = q.FinishedLevel
	} else if isActive {
		active = q.FinishedLevel - 1
		trained = q.FinishedLevel - 1
		required = 0
	} else {
		active = q.FinishedLevel - 1
		trained = q.FinishedLevel - 1
		required = q.FinishedLevel
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
	queue := container.NewStack(
		w.progress,
		container.NewBorder(nil, nil, nil, w.duration, w.name))
	if w.isMobile {
		return widget.NewSimpleRenderer(queue)
	}
	c := container.NewBorder(nil, nil, w.skillLevel, nil, queue)
	return widget.NewSimpleRenderer(c)
}
