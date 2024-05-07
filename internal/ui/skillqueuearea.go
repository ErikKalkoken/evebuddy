package ui

import (
	"fmt"
	"image/color"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/dustin/go-humanize"
)

// skillqueueArea is the UI area that shows the skillqueue
type skillqueueArea struct {
	content *fyne.Container
	ui      *ui
}

func (u *ui) NewSkillqueueArea() *skillqueueArea {
	c := skillqueueArea{ui: u, content: container.NewStack()}
	return &c
}

func (a *skillqueueArea) Redraw() {
	a.content.RemoveAll()
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		return
	}
	qq, err := a.ui.service.ListSkillqueue(characterID)
	if err != nil {
		slog.Error("failed to fetch skillqueue", "err", err)
		a.content.Add(makeMessage("Failed to fetch skillqueue", widget.DangerImportance))
		return
	}
	if len(qq) == 0 {
		a.content.Add(makeMessage("No data found", widget.WarningImportance))
		return
	}

	now := time.Now()
	qq2 := make([]*model.SkillqueueItem, 0)
	for _, q := range qq {
		if q.FinishDate.Before(now) {
			continue
		}
		qq2 = append(qq2, q)
	}

	if len(qq2) == 0 {
		a.content.Add(makeMessage("Skill queue is not active!", widget.WarningImportance))
		return
	}

	list := widget.NewList(
		func() int {
			return len(qq2)
		},
		func() fyne.CanvasObject {
			pb := widget.NewProgressBar()
			pb.Hide()
			return container.NewStack(
				pb,
				container.NewHBox(
					widget.NewLabel("skill"),
					layout.NewSpacer(),
					widget.NewLabel("finished"),
				))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			q := qq2[i]
			row := o.(*fyne.Container).Objects[1].(*fyne.Container)
			name := q.Name()
			row.Objects[0].(*widget.Label).SetText(name)
			var finished string
			if !q.FinishDate.IsZero() {
				finished = fmt.Sprintf("%s (%s)", q.FinishDate.Format(myDateTime), humanize.Time(q.FinishDate))
			} else {
				finished = "?"
			}
			row.Objects[2].(*widget.Label).SetText(finished)
			pb := o.(*fyne.Container).Objects[0].(*widget.ProgressBar)
			if q.IsActive() {
				pb.SetValue(q.CompletionP())
				pb.Show()
			}
		})

	list.OnSelected = func(id widget.ListItemID) {
		q := qq2[id]

		var isActive string
		if q.IsActive() {
			isActive = "yes"
		} else {
			isActive = "no"
		}
		data := [][]string{
			{"Name", q.Name()},
			{"Group", q.GroupName},
			{"Start date", q.StartDate.Format(myDateTime)},
			{"Finish date", q.FinishDate.Format(myDateTime)},
			{"Duration", humanize.RelTime(q.StartDate, q.FinishDate, "", "")},
			{"Completed", fmt.Sprintf("%.0f%%", q.CompletionP()*100)},
			{"Trained SP at start", humanize.Comma(int64(q.TrainingStartSP - q.LevelStartSP))},
			{"Total SP", humanize.Comma(int64(q.LevelEndSP - q.LevelStartSP))},
			{"Active?", isActive},
		}
		dlg := dialog.NewCustom("Skill Details", "OK", makeDataForm(data), a.ui.window)
		dlg.Show()
	}

	a.content.Add(list)
}

func makeMessage(msg string, importance widget.Importance) *fyne.Container {
	var c color.Color
	switch importance {
	case widget.DangerImportance:
		c = theme.ErrorColor()
	case widget.WarningImportance:
		c = theme.WarningColor()
	default:
		c = theme.ForegroundColor()
	}
	t := canvas.NewText(msg, c)
	return container.NewHBox(layout.NewSpacer(), t, layout.NewSpacer())
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
				a.Redraw()
			}()
			<-ticker.C
		}
	}()
}

func makeDataForm(data [][]string) *widget.Form {
	form := widget.NewForm()
	for _, row := range data {
		form.Append(row[0], widget.NewLabel(row[1]))
	}
	return form
}
