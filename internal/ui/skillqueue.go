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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/helper/types"
	"github.com/ErikKalkoken/evebuddy/internal/model"
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
			pb := widget.NewProgressBar()
			pb.Hide()
			return container.NewStack(
				pb,
				container.NewHBox(
					widget.NewLabel("skill"),
					layout.NewSpacer(),
					widget.NewLabel("duration"),
				))
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			row := co.(*fyne.Container).Objects[1].(*fyne.Container)
			name := row.Objects[0].(*widget.Label)
			duration := row.Objects[2].(*widget.Label)
			pb := co.(*fyne.Container).Objects[0].(*widget.ProgressBar)

			q, err := convertDataItem[*model.CharacterSkillqueueItem](di)
			if err != nil {
				slog.Error("failed to render row in skillqueue table", "err", err)
				name.Text = "failed to render"
				name.Importance = widget.DangerImportance
				name.Refresh()
				duration.SetText("")
				pb.Hide()
				return
			}
			var i widget.Importance
			var d string
			if q.IsCompleted() {
				i = widget.LowImportance
				d = "Completed"
			} else if q.IsActive() {
				i = widget.MediumImportance
				d = humanizedNullDuration(q.Remaining(), "?")
			} else {
				i = widget.MediumImportance
				d = humanizedNullDuration(q.Duration(), "?")
			}
			name.Importance = i
			name.Text = q.Name()
			name.Refresh()
			duration.Text = d
			duration.Importance = i
			duration.Refresh()
			if q.IsActive() {
				pb.SetValue(q.CompletionP())
				pb.Show()
			} else {
				pb.Hide()
			}
		})

	list.OnSelected = func(id widget.ListItemID) {
		q, err := getItemUntypedList[*model.CharacterSkillqueueItem](a.items, id)
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
			{"Name", q.Name(), false},
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
	t, i, err := func() (string, widget.Importance, error) {
		total, completion, err := a.updateItems()
		if err != nil {
			return "", 0, err
		}
		s := "Skills"
		if completion.Valid && completion.Float64 < 1 {
			s += fmt.Sprintf(" (%.0f%%)", completion.Float64*100)
		}
		a.ui.skillqueueTab.Text = s
		a.ui.tabs.Refresh()
		return a.makeTopText(total)
	}()
	if err != nil {
		slog.Error("Failed to refresh skill queue UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.total.Text = t
	a.total.Importance = i
	a.total.Refresh()
}

func (a *skillqueueArea) updateItems() (types.NullDuration, sql.NullFloat64, error) {
	var remaining types.NullDuration
	var completion sql.NullFloat64
	ctx := context.Background()
	if !a.ui.hasCharacter() {
		err := a.items.Set(make([]any, 0))
		if err != nil {
			return remaining, completion, err
		}
	}
	skills, err := a.ui.service.ListCharacterSkillqueueItems(ctx, a.ui.currentCharID())
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

func (a *skillqueueArea) makeTopText(total types.NullDuration) (string, widget.Importance, error) {
	hasData, err := a.ui.service.CharacterSectionWasUpdated(a.ui.currentCharID(), model.CharacterSectionSkillqueue)
	if err != nil {
		return "", 0, err
	}
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	if a.items.Length() == 0 {
		return "Training not active", widget.WarningImportance, nil
	}
	t := fmt.Sprintf("Total training time: %s", humanizedNullDuration(total, "?"))
	return t, widget.MediumImportance, nil
}
