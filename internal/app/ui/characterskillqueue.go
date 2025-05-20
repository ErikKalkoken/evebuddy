package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type characterSkillQueue struct {
	widget.BaseWidget

	OnUpdate func(statusShort, statusLong string)

	list *widget.List
	sq   *app.CharacterSkillqueue
	top  *widget.Label
	u    *baseUI
}

func newCharacterSkillQueue(u *baseUI) *characterSkillQueue {
	a := &characterSkillQueue{
		top: makeTopLabel(),
		sq:  app.NewCharacterSkillqueue(),
		u:   u,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeSkillQueue()
	return a
}

func (a *characterSkillQueue) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.list)
	return widget.NewSimpleRenderer(c)
}

func (a *characterSkillQueue) makeSkillQueue() *widget.List {
	list := widget.NewList(
		func() int {
			return a.sq.Size()
		},
		func() fyne.CanvasObject {
			return newSkillQueueItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			q := a.sq.Item(id)
			if q == nil {
				return
			}
			item := co.(*skillQueueItem)
			item.Set(q)
		})

	list.OnSelected = func(id widget.ListItemID) {
		q := a.sq.Item(id)
		if q == nil {
			list.UnselectAll()
			return
		}
		var isActive string
		if q.IsActive() {
			isActive = "yes"
		} else {
			isActive = "no"
		}
		var data = []struct {
			label string
			value string
			wrap  bool
		}{
			{"Name", app.SkillDisplayName(q.SkillName, q.FinishedLevel), false},
			{"Group", q.GroupName, false},
			{"Description", q.SkillDescription, true},
			{"Start date", timeFormattedOrFallback(q.StartDate, app.DateTimeFormat, "?"), false},
			{"Finish date", timeFormattedOrFallback(q.FinishDate, app.DateTimeFormat, "?"), false},
			{"Duration", ihumanize.Optional(q.Duration(), "?"), false},
			{"Remaining", ihumanize.Optional(q.Remaining(), "?"), false},
			{"Completed", fmt.Sprintf("%.0f%%", q.CompletionP()*100), false},
			{"SP at start", humanize.Comma(int64(q.TrainingStartSP - q.LevelStartSP)), false},
			{"Total SP", humanize.Comma(int64(q.LevelEndSP - q.LevelStartSP)), false},
			{"Active?", isActive, false},
		}
		form := widget.NewForm()
		if !a.u.isDesktop {
			form.Orientation = widget.Vertical
		}
		for _, row := range data {
			c := widget.NewLabel(row.value)
			if row.wrap {
				c.Wrapping = fyne.TextWrapWord
			}
			form.Append(row.label, c)

		}
		s := container.NewScroll(form)
		s.SetMinSize(fyne.NewSize(500, 300))
		d := dialog.NewCustom("Skill Details", "OK", s, a.u.MainWindow())
		a.u.ModifyShortcutsForDialog(d, a.u.MainWindow())
		d.SetOnClosed(func() {
			list.UnselectAll()
		})
		d.Show()
	}
	return list
}

func (a *characterSkillQueue) update() {
	var t string
	var i widget.Importance
	err := a.sq.Update(a.u.cs, a.u.currentCharacterID())
	if err != nil {
		slog.Error("Failed to refresh skill queue UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		var s1, s2 string
		isActive := a.sq.IsActive()
		if !isActive {
			s1 = "!"
			s2 = "training paused"
		} else if c := a.sq.Completion(); c.ValueOrZero() < 1 {
			s1 = fmt.Sprintf("%.0f%%", c.ValueOrZero()*100)
			s2 = fmt.Sprintf("%s (%s)", a.sq.Current(), s1)
		}
		if a.OnUpdate != nil {
			a.OnUpdate(s1, s2)
		}
		var total optional.Optional[time.Duration]
		if isActive {
			total = a.sq.Remaining()
		}
		t, i = a.makeTopText(total)
	}
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.list.Refresh()
	})
}

func (a *characterSkillQueue) makeTopText(total optional.Optional[time.Duration]) (string, widget.Importance) {
	hasData := a.u.scs.HasCharacterSection(a.u.currentCharacterID(), app.SectionSkillqueue)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	if a.sq.Size() == 0 {
		return "Training not active", widget.WarningImportance
	}
	t := fmt.Sprintf("Total training time: %s", ihumanize.Optional(total, "?"))
	return t, widget.MediumImportance
}

func timeFormattedOrFallback(t time.Time, layout, fallback string) string {
	if t.IsZero() {
		return fallback
	}
	return t.Format(layout)
}

type skillQueueItem struct {
	widget.BaseWidget

	duration   *widget.Label
	progress   *widget.ProgressBar
	name       *widget.Label
	skillLevel *skillLevel
	isMobile   bool
}

func newSkillQueueItem() *skillQueueItem {
	name := widget.NewLabel("skill")
	name.Truncation = fyne.TextTruncateEllipsis
	pb := widget.NewProgressBar()
	w := &skillQueueItem{
		duration:   widget.NewLabel("duration"),
		name:       name,
		progress:   pb,
		skillLevel: newSkillLevel(),
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

func (w *skillQueueItem) Set(q *app.CharacterSkillqueueItem) {
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

func (w *skillQueueItem) CreateRenderer() fyne.WidgetRenderer {
	queue := container.NewStack(
		w.progress,
		container.NewBorder(nil, nil, nil, w.duration, w.name))
	if w.isMobile {
		return widget.NewSimpleRenderer(queue)
	}
	c := container.NewBorder(nil, nil, w.skillLevel, nil, queue)
	return widget.NewSimpleRenderer(c)
}
