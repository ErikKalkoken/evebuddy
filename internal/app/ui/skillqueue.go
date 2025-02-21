package ui

import (
	"context"
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

// SkillqueueArea is the UI area that shows the skillqueue
type SkillqueueArea struct {
	Content *fyne.Container

	OnRefresh func(statusShort, statusLong string)

	items []*app.CharacterSkillqueueItem
	total *widget.Label
	u     *BaseUI
}

func (u *BaseUI) NewSkillqueueArea() *SkillqueueArea {
	a := SkillqueueArea{
		items: make([]*app.CharacterSkillqueueItem, 0),
		total: makeTopLabel(),
		u:     u,
	}
	list := a.makeSkillQueue()
	top := container.NewVBox(a.total, widget.NewSeparator())
	a.Content = container.NewBorder(top, nil, nil, nil, list)
	return &a
}

func (a *SkillqueueArea) makeSkillQueue() *widget.List {
	list := widget.NewList(
		func() int {
			return len(a.items)
		},
		func() fyne.CanvasObject {
			return appwidget.NewSkillQueueItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.items) {
				return
			}
			q := a.items[id]
			item := co.(*appwidget.SkillQueueItem)
			item.Set(q)
		})

	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.items) {
			list.UnselectAll()
			return
		}
		q := a.items[id]
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
			{"Name", SkillDisplayName(q.SkillName, q.FinishedLevel), false},
			{"Group", q.GroupName, false},
			{"Description", q.SkillDescription, true},
			{"Start date", timeFormattedOrFallback(q.StartDate, app.TimeDefaultFormat, "?"), false},
			{"Finish date", timeFormattedOrFallback(q.FinishDate, app.TimeDefaultFormat, "?"), false},
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
		dlg := dialog.NewCustom("Skill Details", "OK", s, a.u.Window)
		dlg.SetOnClosed(func() {
			list.UnselectAll()
		})
		dlg.Show()
	}
	return list
}

func (a *SkillqueueArea) Refresh() {
	var t string
	var i widget.Importance
	remaining, completion, current, err := a.updateItems()
	if err != nil {
		slog.Error("Failed to refresh skill queue UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		var s1, s2 string
		if remaining.IsEmpty() {
			s1 = "!"
			s2 = "training paused"
		} else if completion.ValueOrZero() < 1 {
			s1 = fmt.Sprintf("%.0f%%", completion.ValueOrZero()*100)
			s2 = fmt.Sprintf("%s (%s)", current, s1)
		}
		if a.OnRefresh != nil {
			a.OnRefresh(s1, s2)
		}
		t, i = a.makeTopText(remaining)
	}
	a.total.Text = t
	a.total.Importance = i
	a.total.Refresh()
}

func (a *SkillqueueArea) updateItems() (
	remaining optional.Optional[time.Duration],
	completion optional.Optional[float64],
	current *app.CharacterSkillqueueItem,
	err error,
) {
	ctx := context.TODO()
	if !a.u.HasCharacter() {
		a.items = make([]*app.CharacterSkillqueueItem, 0)
		return
	}
	var items []*app.CharacterSkillqueueItem
	items, err = a.u.CharacterService.ListCharacterSkillqueueItems(ctx, a.u.CharacterID())
	if err != nil {
		return
	}
	for _, item := range items {
		remaining = optional.New(remaining.ValueOrZero() + item.Remaining().ValueOrZero())
		if item.IsActive() {
			completion = optional.New(item.CompletionP())
			current = item
		}
	}
	a.items = items
	return
}

func (a *SkillqueueArea) makeTopText(total optional.Optional[time.Duration]) (string, widget.Importance) {
	hasData := a.u.StatusCacheService.CharacterSectionExists(a.u.CharacterID(), app.SectionSkillqueue)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	if len(a.items) == 0 {
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
