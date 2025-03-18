package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type attribute struct {
	icon   fyne.Resource
	name   string
	points int
}

func (a attribute) isText() bool {
	return a.points == 0
}

// Attributes is the UI area that shows the skillqueue
type Attributes struct {
	Content fyne.CanvasObject

	attributes []attribute
	top        *widget.Label
	u          *BaseUI
}

func NewAttributes(u *BaseUI) *Attributes {
	a := Attributes{
		attributes: make([]attribute, 0),
		top:        MakeTopLabel(),
		u:          u,
	}
	list := a.makeAttributeList()
	a.Content = container.NewBorder(container.NewVBox(a.top, widget.NewSeparator()), nil, nil, nil, list)
	return &a
}

func (a *Attributes) makeAttributeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.attributes)
		},
		func() fyne.CanvasObject {
			icon := iwidget.NewImageFromResource(icons.QuestionmarkSvg, fyne.NewSquareSize(app.IconUnitSize))
			return container.NewHBox(
				icon, widget.NewLabel("placeholder"), layout.NewSpacer(), widget.NewLabel("points"))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.attributes) {
				return
			}
			q := a.attributes[id]
			hbox := co.(*fyne.Container).Objects
			name := hbox[1].(*widget.Label)
			name.SetText(q.name)

			icon := hbox[0].(*canvas.Image)
			if q.isText() {
				icon.Hide()
			} else {
				icon.Show()
				icon.Resource = q.icon
				icon.Refresh()
			}

			points := hbox[3].(*widget.Label)
			if q.isText() {
				points.Hide()
			} else {
				points.Show()
				points.SetText(fmt.Sprintf("%d points", q.points))
			}
		})

	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *Attributes) Refresh() {
	var t string
	var i widget.Importance
	total, err := a.updateData()
	if err != nil {
		slog.Error("Failed to refresh attributes UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeTopText(total)
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
}

func (a *Attributes) makeTopText(total int) (string, widget.Importance) {
	hasData := a.u.StatusCacheService.CharacterSectionExists(a.u.CurrentCharacterID(), app.SectionAttributes)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	return fmt.Sprintf("Total points: %d", total), widget.MediumImportance
}

func (a *Attributes) updateData() (int, error) {
	if !a.u.HasCharacter() {
		a.attributes = make([]attribute, 0)
		return 0, nil
	}
	ctx := context.TODO()
	ca, err := a.u.CharacterService.GetCharacterAttributes(ctx, a.u.CurrentCharacterID())
	if errors.Is(err, character.ErrNotFound) {
		a.attributes = make([]attribute, 0)
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	resPerception := eveicon.GetResourceByName(eveicon.Perception)
	resMemory := eveicon.GetResourceByName(eveicon.Memory)
	resWillpower := eveicon.GetResourceByName(eveicon.Willpower)
	resIntelligence := eveicon.GetResourceByName(eveicon.Intelligence)
	resCharisma := eveicon.GetResourceByName(eveicon.Charisma)
	items := make([]attribute, 6)
	items[0] = attribute{
		icon:   resPerception,
		name:   "Perception",
		points: ca.Perception,
	}
	items[1] = attribute{
		icon:   resMemory,
		name:   "Memory",
		points: ca.Memory,
	}
	items[2] = attribute{
		icon:   resWillpower,
		name:   "Willpower",
		points: ca.Willpower,
	}
	items[3] = attribute{
		icon:   resIntelligence,
		name:   "Intelligence",
		points: ca.Intelligence,
	}
	items[4] = attribute{
		icon:   resCharisma,
		name:   "Charisma",
		points: ca.Charisma,
	}
	items[5] = attribute{
		name: fmt.Sprintf("Bonus Remaps Available: %d", ca.BonusRemaps),
	}
	a.attributes = items
	total := ca.Charisma + ca.Intelligence + ca.Memory + ca.Perception + ca.Willpower
	return total, nil
}
