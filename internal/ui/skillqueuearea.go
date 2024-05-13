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
			name := row.Objects[0].(*widget.Label)
			name.Importance = i
			name.Text = q.Name()
			name.Refresh()
			duration := row.Objects[2].(*widget.Label)
			duration.Text = d
			duration.Importance = i
			duration.Refresh()
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
	items, err := a.ui.service.ListSkillqueueItems(characterID)
	if err != nil {
		slog.Error("failed to fetch skillqueue", "characterID", characterID, "err", err)
		c := a.ui.CurrentChar()
		a.errorText = fmt.Sprintf("Failed to fetch skillqueue for %s", c.Character.Name)
		return
	}
	a.items = make([]*model.SkillqueueItem, 0)
	for _, item := range items {
		if item.StartDate.IsZero() || item.FinishDate.IsZero() {
			continue
		}
		a.items = append(a.items, item)
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
				isExpired, err := a.ui.service.SectionIsUpdateExpired(characterID, service.UpdateSectionSkillqueue)
				if err != nil {
					slog.Error(err.Error())
					return
				}
				if !isExpired {
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
