package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/helper/types"
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

// skillqueueArea is the UI area that shows the skillqueue
type skillqueueArea struct {
	content       *fyne.Container
	items         binding.UntypedList
	errorText     binding.String
	trainingTotal binding.Untyped
	total         *widget.Label
	ui            *ui
}

func (u *ui) NewSkillqueueArea() *skillqueueArea {
	a := skillqueueArea{
		items:         binding.NewUntypedList(),
		errorText:     binding.NewString(),
		total:         widget.NewLabel(""),
		trainingTotal: binding.NewUntyped(),
		ui:            u,
	}
	a.total.TextStyle.Bold = true
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

			q, err := convertDataItem[*model.SkillqueueItem](di)
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
		q, err := getFromBoundUntypedList[*model.SkillqueueItem](a.items, id)
		if err != nil {
			slog.Error("failed to fetch skillqueue item in list", "err", err)
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
		dlg := dialog.NewCustom("Skill Details", "OK", s, u.window)
		dlg.Show()
		dlg.Resize(fyne.Size{
			Width:  0.8 * a.ui.window.Canvas().Size().Width,
			Height: 0.8 * a.ui.window.Canvas().Size().Height,
		})
	}

	top := container.NewVBox(a.total, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, list)
	return &a
}

func (a *skillqueueArea) Refresh() {
	if err := a.updateItems(); err != nil {
		slog.Error("failed to update skillqueue items for character", "characterID", a.ui.CurrentCharID(), "err", err)
		return
	}
	s, i := a.makeTopText()
	a.total.Text = s
	a.total.Importance = i
	a.total.Refresh()
}

func (a *skillqueueArea) makeTopText() (string, widget.Importance) {
	errorText, err := a.errorText.Get()
	if err != nil {
		panic(err)
	}
	if errorText != "" {
		return errorText, widget.DangerImportance
	}
	hasData, err := a.ui.service.SectionWasUpdated(a.ui.CurrentCharID(), model.UpdateSectionSkillqueue)
	if err != nil {
		return "ERROR", widget.DangerImportance
	}
	if !hasData {
		return "No data", widget.LowImportance
	}
	if a.items.Length() == 0 {
		return "Training not active", widget.WarningImportance
	}
	var s string
	x, err := a.trainingTotal.Get()
	if err != nil {
		panic(err)
	}
	d := x.(types.NullDuration)
	if d.Valid {
		s = ihumanize.Duration(d.Duration)
	} else {
		s = "?"
	}
	return fmt.Sprintf("Total training time: %s", s), widget.MediumImportance
}

func (a *skillqueueArea) updateItems() error {
	if err := a.errorText.Set(""); err != nil {
		return err
	}
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		err := a.items.Set(make([]any, 0))
		if err != nil {
			return err
		}
	}
	items, err := a.ui.service.ListSkillqueueItems(characterID)
	if err != nil {
		return err
	}
	var total types.NullDuration
	x := make([]any, len(items))
	for i, item := range items {
		x[i] = item
		total.Duration += item.Remaining().Duration
		total.Valid = true
	}
	if err := a.items.Set(x); err != nil {
		panic(err)
	}
	if err := a.trainingTotal.Set(total); err != nil {
		return err
	}
	return nil
}

func (a *skillqueueArea) StartUpdateTicker() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			func() {
				cc, err := a.ui.service.ListMyCharactersShort()
				if err != nil {
					slog.Error("Failed to fetch list of my characters", "err", err)
					return
				}
				for _, c := range cc {
					a.UpdateAndRefresh(c.ID)
				}
			}()
			<-ticker.C
		}
	}()
}

func (a *skillqueueArea) UpdateAndRefresh(characterID int32) {
	_, err := a.ui.service.UpdateSectionIfExpired(characterID, model.UpdateSectionSkillqueue)
	if err != nil {
		slog.Error("Failed to update skillqueue", "character", characterID, "err", err)
		return
	}
	if characterID == a.ui.CurrentCharID() {
		a.Refresh()
	}
}
