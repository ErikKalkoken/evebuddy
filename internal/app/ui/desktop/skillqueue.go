package desktop

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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// skillqueueArea is the UI area that shows the skillqueue
type skillqueueArea struct {
	content *fyne.Container
	items   []*app.CharacterSkillqueueItem
	total   *widget.Label
	u       *DesktopUI
}

func (u *DesktopUI) newSkillqueueArea() *skillqueueArea {
	a := skillqueueArea{
		items: make([]*app.CharacterSkillqueueItem, 0),
		total: widget.NewLabel(""),
		u:     u,
	}

	a.total.TextStyle.Bold = true
	list := a.makeSkillqueue()

	top := container.NewVBox(a.total, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, list)
	return &a
}

func (a *skillqueueArea) makeSkillqueue() *widget.List {
	list := widget.NewList(
		func() int {
			return len(a.items)
		},
		func() fyne.CanvasObject {
			return widgets.NewSkillQueueItem()
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.items) {
				return
			}
			q := a.items[id]
			item := co.(*widgets.SkillQueueItem)
			item.Set(q.SkillName, q.FinishedLevel, q.IsActive(), q.Remaining(), q.Duration(), q.CompletionP())
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
			{"Name", ui.SkillDisplayName(q.SkillName, q.FinishedLevel), false},
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
		for _, row := range data {
			c := widget.NewLabel(row.value)
			if row.wrap {
				c.Wrapping = fyne.TextWrapWord
			}
			form.Append(row.label, c)

		}
		s := container.NewScroll(form)
		dlg := dialog.NewCustom("Skill Details", "OK", s, a.u.Window)
		dlg.SetOnClosed(func() {
			list.UnselectAll()
		})
		dlg.Show()
		dlg.Resize(fyne.Size{
			Width:  0.8 * a.u.Window.Canvas().Size().Width,
			Height: 0.8 * a.u.Window.Canvas().Size().Height,
		})
	}
	return list
}

func (a *skillqueueArea) refresh() {
	var t string
	var i widget.Importance
	remaining, completion, err := a.updateItems()
	if err != nil {
		slog.Error("Failed to refresh skill queue UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		s := "Skills"
		if remaining.IsEmpty() {
			s += " (!)"
		} else if completion.ValueOrZero() < 1 {
			s += fmt.Sprintf(" (%.0f%%)", completion.ValueOrZero()*100)
		}
		a.u.skillTab.Text = s
		a.u.tabs.Refresh()
		t, i = a.makeTopText(remaining)
	}
	a.total.Text = t
	a.total.Importance = i
	a.total.Refresh()
}

func (a *skillqueueArea) updateItems() (remaining optional.Optional[time.Duration], completion optional.Optional[float64], err error) {
	ctx := context.TODO()
	if !a.u.HasCharacter() {
		a.items = make([]*app.CharacterSkillqueueItem, 0)
		return
	}
	var items []*app.CharacterSkillqueueItem
	items, err = a.u.CharacterService.ListCharacterSkillqueueItems(ctx, a.u.CharacterID())
	if err != nil {
		return remaining, completion, err
	}
	for _, item := range items {
		remaining = optional.New(remaining.ValueOrZero() + item.Remaining().ValueOrZero())
		if item.IsActive() {
			completion = optional.New(item.CompletionP())
		}
	}
	a.items = items
	return
}

func (a *skillqueueArea) makeTopText(total optional.Optional[time.Duration]) (string, widget.Importance) {
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
