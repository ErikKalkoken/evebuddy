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
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
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

// CharacterAttributes shows the attributes for the current character.
type CharacterAttributes struct {
	widget.BaseWidget

	attributes []attribute
	list       *widget.List
	top        *widget.Label
	u          *BaseUI
}

func NewCharacterAttributes(u *BaseUI) *CharacterAttributes {
	w := &CharacterAttributes{
		attributes: make([]attribute, 0),
		top:        appwidget.MakeTopLabel(),
		u:          u,
	}
	w.list = w.makeAttributeList()
	w.ExtendBaseWidget(w)
	return w
}

func (a *CharacterAttributes) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.list)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterAttributes) makeAttributeList() *widget.List {
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
	l.HideSeparators = true
	return l
}

func (a *CharacterAttributes) update() {
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
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
		a.list.Refresh()
	})
}

func (a *CharacterAttributes) makeTopText(total int) (string, widget.Importance) {
	hasData := a.u.StatusCacheService().CharacterSectionExists(a.u.CurrentCharacterID(), app.SectionAttributes)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	return fmt.Sprintf("Total points: %d", total), widget.MediumImportance
}

func (a *CharacterAttributes) updateData() (int, error) {
	if !a.u.hasCharacter() {
		a.attributes = make([]attribute, 0)
		return 0, nil
	}
	ctx := context.TODO()
	ca, err := a.u.CharacterService().GetAttributes(ctx, a.u.CurrentCharacterID())
	if errors.Is(err, app.ErrNotFound) {
		a.attributes = make([]attribute, 0)
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	resPerception := eveicon.FromName(eveicon.Perception)
	resMemory := eveicon.FromName(eveicon.Memory)
	resWillpower := eveicon.FromName(eveicon.Willpower)
	resIntelligence := eveicon.FromName(eveicon.Intelligence)
	resCharisma := eveicon.FromName(eveicon.Charisma)
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
