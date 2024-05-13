package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/helper/types"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"

	"github.com/dustin/go-humanize"
)

// skillqueueArea is the UI area that shows the skillqueue
type skillqueueArea struct {
	content       *fyne.Container
	errorText     string
	items         []*model.SkillqueueItem
	trainingTotal types.NullDuration
	list          *widget.List
	total         *widget.Label
	ui            *ui
}

func (u *ui) NewSkillqueueArea() *skillqueueArea {
	a := skillqueueArea{
		ui:    u,
		items: make([]*model.SkillqueueItem, 0),
		total: widget.NewLabel(""),
	}
	a.total.TextStyle.Bold = true
	list := widget.NewList(
		func() int {
			return len(a.items)
		},
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
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if len(a.items) <= id {
				return
			}
			q := a.items[id]
			row := o.(*fyne.Container).Objects[1].(*fyne.Container)
			name := q.Name()
			row.Objects[0].(*widget.Label).SetText(name)
			var duration string
			if !q.IsCompleted() {
				duration = ihumanize.Duration(q.Duration())
			} else {
				duration = "Completed"
			}
			row.Objects[2].(*widget.Label).SetText(duration)
			pb := o.(*fyne.Container).Objects[0].(*widget.ProgressBar)
			if q.IsActive() {
				pb.SetValue(q.CompletionP())
				pb.Show()
			} else {
				pb.Hide()
			}
		})

	list.OnSelected = func(id widget.ListItemID) {
		if len(a.items) <= id {
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
			{"Name", q.Name(), false},
			{"Group", q.GroupName, false},
			{"Description", q.SkillDescription, true},
			{"Start date", q.StartDate.Format(myDateTime), false},
			{"Finish date", q.FinishDate.Format(myDateTime), false},
			{"Duration", humanize.RelTime(q.StartDate, q.FinishDate, "", ""), false},
			{"Remaining", humanize.RelTime(q.FinishDate, time.Now(), "", ""), false},
			{"Completed", fmt.Sprintf("%.0f%%", q.CompletionP()*100), false},
			{"SP at start", humanize.Comma(int64(q.TrainingStartSP - q.LevelStartSP)), false},
			{"Total SP", humanize.Comma(int64(q.LevelEndSP - q.LevelStartSP)), false},
			{"Active?", isActive, false},
			{"Position", fmt.Sprintf("%d", q.QueuePosition), false},
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
	a.list = list
	return &a
}

func (a *skillqueueArea) Refresh() {
	a.updateItems()
	a.list.Refresh()
	s, i := a.makeTopText()
	a.total.Text = s
	a.total.Importance = i
	a.total.Refresh()
}

func (a *skillqueueArea) makeTopText() (string, widget.Importance) {
	if a.errorText != "" {
		return a.errorText, widget.DangerImportance
	}
	if len(a.items) == 0 {
		return "Training not active", widget.WarningImportance
	}
	var x string
	if a.trainingTotal.Valid {
		x = ihumanize.Duration(a.trainingTotal.Duration)
	} else {
		x = "?"
	}
	s := fmt.Sprintf("Total training time: %s", x)
	return s, widget.MediumImportance
}

func (a *skillqueueArea) updateItems() {
	a.items = a.items[0:0]
	a.errorText = ""
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		return
	}
	var err error
	a.items, err = a.ui.service.ListSkillqueue(characterID)
	if err != nil {
		slog.Error("failed to fetch skillqueue", "characterID", characterID, "err", err)
		c := a.ui.CurrentChar()
		a.errorText = fmt.Sprintf("Failed to fetch skillqueue for %s", c.Character.Name)
		return
	}
	total, err := a.ui.service.GetTotalTrainingTime(characterID)
	if err != nil {
		slog.Error("failed to fetch skillqueue", "characterID", characterID, "err", err)
		a.trainingTotal = types.NullDuration{}
	} else {
		a.trainingTotal = total
	}
}

func (a *skillqueueArea) StartUpdateTicker() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			func() {
				characterID := a.ui.CurrentCharID()
				if characterID == 0 {
					return
				}
				if !a.ui.service.SectionUpdatedExpired(characterID, service.UpdateSectionSkillqueue) {
					return
				}
				if err := a.ui.service.UpdateSkillqueueESI(characterID); err != nil {
					slog.Error(err.Error())
					return
				}
				a.Refresh()
			}()
			<-ticker.C
		}
	}()
}
