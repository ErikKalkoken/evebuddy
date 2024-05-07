package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
		t := canvas.NewText("Failed to fetch skillqueue", theme.ErrorColor())
		a.content.Add(t)
		return
	}
	if len(qq) == 0 {
		a.content.Add(widget.NewLabel("No data"))
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
		a.content.Add(widget.NewLabel("Skill queue is not active!"))
		return
	}

	list := widget.NewList(
		func() int {
			return len(qq2)
		},
		func() fyne.CanvasObject {
			x := widget.NewProgressBarInfinite()
			x.Stop()
			x.Hide()
			return container.NewStack(
				x,
				container.NewHBox(
					widget.NewLabel("skill"),
					layout.NewSpacer(),
					widget.NewLabel("finished"),
				))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			q := qq2[i]
			row := o.(*fyne.Container).Objects[1].(*fyne.Container)
			name := fmt.Sprintf("%s %s", q.SkillName, romanLetter(q.FinishedLevel))
			row.Objects[0].(*widget.Label).SetText(name)
			var finished string
			if !q.FinishDate.IsZero() {
				finished = fmt.Sprintf("%s (%s)", q.FinishDate.Format(myDateTime), humanize.Time(q.FinishDate))
			} else {
				finished = "?"
			}
			row.Objects[2].(*widget.Label).SetText(finished)
			progressBar := o.(*fyne.Container).Objects[0].(*widget.ProgressBarInfinite)
			if q.StartDate.Before(now) && q.FinishDate.After(now) {
				progressBar.Show()
				progressBar.Start()
			}
		})

	a.content.Add(list)
}

func romanLetter(v int) string {
	m := map[int]string{
		1: "I",
		2: "II",
		3: "III",
		4: "IV",
		5: "V",
	}
	r, ok := m[v]
	if !ok {
		panic(fmt.Sprintf("invalid value: %d", v))
	}
	return r
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
