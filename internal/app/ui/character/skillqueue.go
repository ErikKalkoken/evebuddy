package character

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
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type SkillQueue struct {
	widget.BaseWidget

	OnUpdate func(statusShort, statusLong string)

	list *widget.List
	sq   app.CharacterSkillqueue
	top  *widget.Label
	u    app.UI
}

func NewSkillQueue(u app.UI) *SkillQueue {
	a := &SkillQueue{
		top: appwidget.MakeTopLabel(),
		sq:  app.NewCharacterSkillqueue(),
		u:   u,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeSkillQueue()
	return a
}

func (a *SkillQueue) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.list)
	return widget.NewSimpleRenderer(c)
}

func (a *SkillQueue) makeSkillQueue() *widget.List {
	list := widget.NewList(
		func() int {
			return a.sq.Size()
		},
		func() fyne.CanvasObject {
			return NewSkillQueueItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			q := a.sq.Item(id)
			if q == nil {
				return
			}
			item := co.(*SkillQueueItem)
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
		if a.u.IsMobile() {
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

func (a *SkillQueue) Update() {
	var t string
	var i widget.Importance
	err := a.sq.Update(a.u.CharacterService(), a.u.CurrentCharacterID())
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
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
}

func (a *SkillQueue) makeTopText(total optional.Optional[time.Duration]) (string, widget.Importance) {
	hasData := a.u.StatusCacheService().CharacterSectionExists(a.u.CurrentCharacterID(), app.SectionSkillqueue)
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
