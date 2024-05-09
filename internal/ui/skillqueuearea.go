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
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/dustin/go-humanize"
)

// skillqueueArea is the UI area that shows the skillqueue
type skillqueueArea struct {
	content   *fyne.Container
	errorText string
	items     []*model.SkillqueueItem
	list      *widget.List
	total     *widget.Label
	ui        *ui
}

func (u *ui) NewSkillqueueArea() *skillqueueArea {
	a := skillqueueArea{
		ui:    u,
		items: make([]*model.SkillqueueItem, 0),
	}
	a.updateItems()
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
		func(i widget.ListItemID, o fyne.CanvasObject) {
			q := a.items[i]
			row := o.(*fyne.Container).Objects[1].(*fyne.Container)
			name := q.Name()
			row.Objects[0].(*widget.Label).SetText(name)
			var duration string
			if !q.FinishDate.IsZero() {
				duration = ihumanize.Duration(q.Duration())
			} else {
				duration = "?"
			}
			row.Objects[2].(*widget.Label).SetText(duration)
			pb := o.(*fyne.Container).Objects[0].(*widget.ProgressBar)
			if q.IsActive() {
				pb.SetValue(q.CompletionP())
				pb.Show()
			}
		})

	list.OnSelected = func(id widget.ListItemID) {
		q := a.items[id]

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
			{"Remaining", humanize.RelTime(q.FinishDate, time.Now(), "", "")},
			{"Completed", fmt.Sprintf("%.0f%%", q.CompletionP()*100)},
			{"Trained SP at start", humanize.Comma(int64(q.TrainingStartSP - q.LevelStartSP))},
			{"Total SP", humanize.Comma(int64(q.LevelEndSP - q.LevelStartSP))},
			{"Active?", isActive},
		}
		dlg := dialog.NewCustom("Skill Details", "OK", makeDataForm(data), u.window)
		dlg.Show()
	}

	s, i := a.makeBottomText()
	total := widget.NewLabel(s)
	total.TextStyle = fyne.TextStyle{Bold: true}
	total.Importance = i
	bottom := container.NewVBox(widget.NewSeparator(), total)

	a.content = container.NewBorder(nil, bottom, nil, nil, list)
	a.list = list
	a.total = total
	return &a
}

func makeDataForm(data [][]string) *widget.Form {
	form := widget.NewForm()
	for _, row := range data {
		form.Append(row[0], widget.NewLabel(row[1]))
	}
	return form
}

func (a *skillqueueArea) Refresh() {
	a.updateItems()
	a.list.Refresh()
	s, i := a.makeBottomText()
	a.total.Text = s
	a.total.Importance = i
	a.total.Refresh()
}

func (a *skillqueueArea) makeBottomText() (string, widget.Importance) {
	if a.errorText != "" {
		return a.errorText, widget.DangerImportance
	}
	if len(a.items) == 0 {
		return "Training not active", widget.WarningImportance
	}
	var total time.Duration
	for _, q := range a.items {
		total += q.Duration()
	}
	s := fmt.Sprintf("Total training time: %s", ihumanize.Duration(total))
	return s, widget.MediumImportance
}

func (a *skillqueueArea) updateItems() {
	a.items = a.items[0:0]
	a.errorText = ""
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		return
	}
	qq, err := a.ui.service.ListSkillqueue(characterID)
	if err != nil {
		slog.Error("failed to fetch skillqueue", "characterID", characterID, "err", err)
		c := a.ui.CurrentChar()
		a.errorText = fmt.Sprintf("Failed to fetch skillqueue for %s", c.Character.Name)
		return
	}
	now := time.Now()
	for _, q := range qq {
		if q.FinishDate.Before(now) {
			continue
		}
		a.items = append(a.items, q)
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
