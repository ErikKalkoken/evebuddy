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
	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/dustin/go-humanize"
)

// skillqueueArea is the UI area that shows the skillqueue
type skillqueueArea struct {
	content fyne.CanvasObject
	items   *fyne.Container
	ui      *ui
}

func (u *ui) NewSkillqueueArea() *skillqueueArea {
	items := container.NewVBox()
	c := skillqueueArea{ui: u, content: container.NewScroll(items), items: items}
	return &c
}

func (a *skillqueueArea) Redraw() {
	a.items.RemoveAll()
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		return
	}
	qq, err := a.ui.service.ListSkillqueue(characterID)
	if err != nil {
		t := canvas.NewText("Failed to fetch skillqueue", theme.ErrorColor())
		a.items.Add(t)
		return
	}
	if len(qq) == 0 {
		a.items.Add(widget.NewLabel("No data"))
		return
	}
	now := time.Now()
	isActive := false
	for _, q := range qq {
		if q.StartDate.Before(now) && q.FinishDate.After(now) {
			isActive = true
			break
		}
	}
	if !isActive {
		a.items.Add(widget.NewLabel("Skill queue is not active!"))
		a.items.Add(widget.NewLabel(""))
	}
	header := container.NewHBox(widget.NewLabel("Skill"), layout.NewSpacer(), widget.NewLabel("Finished At"))
	a.items.Add(container.NewStack(
		canvas.NewRectangle(theme.DisabledButtonColor()),
		header,
	))
	for _, q := range qq {
		if isActive && q.FinishDate.Before(now) {
			continue
		}
		name := widget.NewLabel(fmt.Sprintf("%s %s", q.SkillName, romanLetter(q.FinishedLevel)))
		var x string
		if !q.FinishDate.IsZero() {
			x = fmt.Sprintf("%s (%s)", q.FinishDate.Format(myDateTime), humanize.Time(q.FinishDate))
		} else {
			x = "?"
		}
		finished := widget.NewLabel(x)
		row := container.NewHBox(name, layout.NewSpacer(), finished)
		wrapper := container.NewStack()
		if q.StartDate.Before(now) && q.FinishDate.After(now) {
			wrapper.Add(widget.NewProgressBarInfinite())
		}
		wrapper.Add(row)
		a.items.Add(wrapper)
		a.items.Add(widget.NewSeparator())
	}
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
