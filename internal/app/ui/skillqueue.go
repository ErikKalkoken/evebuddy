package ui

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// skillqueueArea is the UI area that shows the skillqueue
type skillqueueArea struct {
	content *fyne.Container
	items   binding.UntypedList
	total   *widget.Label
	ui      *ui
}

func (u *ui) newSkillqueueArea() *skillqueueArea {
	a := skillqueueArea{
		items: binding.NewUntypedList(),
		total: widget.NewLabel(""),
		ui:    u,
	}

	a.total.TextStyle.Bold = true
	list := a.makeSkillqueue()

	top := container.NewVBox(a.total, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, list)
	return &a
}

func (a *skillqueueArea) makeSkillqueue() *widget.List {
	list := widget.NewListWithData(
		a.items,
		func() fyne.CanvasObject {
			return widgets.NewSkillQueueItem()
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			item := co.(*widgets.SkillQueueItem)
			q, err := convertDataItem[*app.CharacterSkillqueueItem](di)
			if err != nil {
				slog.Error("failed to render row in skillqueue table", "err", err)
				item.SetError("failed to render", err)
				return
			}
			item.Set(q.SkillName, q.FinishedLevel, q.IsActive(), q.Remaining(), q.Duration(), q.CompletionP())
		})

	list.OnSelected = func(id widget.ListItemID) {
		q, err := getItemUntypedList[*app.CharacterSkillqueueItem](a.items, id)
		if err != nil {
			slog.Error("failed to access skillqueue item in list", "err", err)
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
			{"Name", skillDisplayName(q.SkillName, q.FinishedLevel), false},
			{"Group", q.GroupName, false},
			{"Description", q.SkillDescription, true},
			{"Start date", timeFormattedOrFallback(q.StartDate, myDateTime, "?"), false},
			{"Finish date", timeFormattedOrFallback(q.FinishDate, myDateTime, "?"), false},
			{"Duration", humanizedNullDuration(q.Duration(), "?"), false},
			{"Remaining", humanizedNullDuration(q.Remaining(), "?"), false},
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
		dlg := dialog.NewCustom("Skill Details", "OK", s, a.ui.window)
		dlg.SetOnClosed(func() {
			list.UnselectAll()
		})
		dlg.Show()
		dlg.Resize(fyne.Size{
			Width:  0.8 * a.ui.window.Canvas().Size().Width,
			Height: 0.8 * a.ui.window.Canvas().Size().Height,
		})
	}
	return list
}

func (a *skillqueueArea) refresh() {
	var t string
	var i widget.Importance
	total, completion, err := a.updateItems()
	if err != nil {
		slog.Error("Failed to refresh skill queue UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		s := "Skills"
		if completion.Valid && completion.Float64 < 1 {
			s += fmt.Sprintf(" (%.0f%%)", completion.Float64*100)
		}
		a.ui.skillqueueTab.Text = s
		a.ui.tabs.Refresh()
		t, i = a.makeTopText(total)
	}
	a.total.Text = t
	a.total.Importance = i
	a.total.Refresh()
}

func (a *skillqueueArea) updateItems() (optional.Duration, sql.NullFloat64, error) {
	var remaining optional.Duration
	var completion sql.NullFloat64
	ctx := context.Background()
	if !a.ui.hasCharacter() {
		err := a.items.Set(make([]any, 0))
		if err != nil {
			return remaining, completion, err
		}
	}
	skills, err := a.ui.CharacterService.ListCharacterSkillqueueItems(ctx, a.ui.characterID())
	if err != nil {
		return remaining, completion, err
	}
	items := make([]any, len(skills))
	for i, skill := range skills {
		items[i] = skill
		remaining.Duration += skill.Remaining().Duration
		remaining.Valid = true
		if skill.IsActive() {
			completion.Valid = true
			completion.Float64 = skill.CompletionP()
		}
	}
	if err := a.items.Set(items); err != nil {
		return remaining, completion, err
	}
	return remaining, completion, nil
}

func (a *skillqueueArea) makeTopText(total optional.Duration) (string, widget.Importance) {
	hasData := a.ui.StatusCacheService.CharacterSectionExists(a.ui.characterID(), app.SectionSkillqueue)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	if a.items.Length() == 0 {
		return "Training not active", widget.WarningImportance
	}
	t := fmt.Sprintf("Total training time: %s", humanizedNullDuration(total, "?"))
	return t, widget.MediumImportance
}
