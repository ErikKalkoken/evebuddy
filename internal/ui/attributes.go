package ui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

type attribute struct {
	icon   fyne.Resource
	name   string
	points int
}

func (a attribute) isText() bool {
	return a.points == 0
}

// attributesArea is the UI area that shows the skillqueue
type attributesArea struct {
	content    fyne.CanvasObject
	attributes binding.UntypedList
	top        *widget.Label
	ui         *ui
}

func (u *ui) newAttributesArena() *attributesArea {
	a := attributesArea{
		attributes: binding.NewUntypedList(),
		top:        widget.NewLabel(""),
		ui:         u,
	}
	a.top.TextStyle.Bold = true
	list := a.makeAttributeList()
	a.content = container.NewBorder(container.NewVBox(a.top, widget.NewSeparator()), nil, nil, nil, list)
	return &a
}

func (a *attributesArea) makeAttributeList() *widget.List {
	l := widget.NewListWithData(
		a.attributes,
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
			icon.FillMode = canvas.ImageFillContain
			icon.SetMinSize(fyne.Size{Width: 32, Height: 32})
			return container.NewHBox(
				icon, widget.NewLabel("placeholder"), layout.NewSpacer(), widget.NewLabel("points"))
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			row := co.(*fyne.Container)

			name := row.Objects[1].(*widget.Label)
			q, err := convertDataItem[attribute](di)
			if err != nil {
				slog.Error("failed to render row in attributes table", "err", err)
				name.Text = "failed to render"
				name.Importance = widget.DangerImportance
				name.Refresh()
				return
			}
			name.SetText(q.name)

			icon := row.Objects[0].(*canvas.Image)
			if q.isText() {
				icon.Hide()
			} else {
				icon.Show()
				icon.Resource = q.icon
				icon.Refresh()
			}

			points := row.Objects[3].(*widget.Label)
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

func (a *attributesArea) refresh() {
	t, i, err := func() (string, widget.Importance, error) {
		total, err := a.updateData()
		if err != nil {
			return "", 0, err
		}
		return a.makeTopText(total)
	}()
	if err != nil {
		slog.Error("Failed to refresh attributes UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
}

func (a *attributesArea) makeTopText(total int) (string, widget.Importance, error) {
	hasData, err := a.ui.sv.Characters.CharacterSectionWasUpdated(
		context.Background(), a.ui.currentCharID(), model.CharacterSectionAttributes)
	if err != nil {
		return "", 0, err
	}
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	return fmt.Sprintf("Total points: %d", total), widget.MediumImportance, nil
}

func (a *attributesArea) updateData() (int, error) {
	if !a.ui.hasCharacter() {
		err := a.attributes.Set(make([]any, 0))
		if err != nil {
			return 0, err
		}
	}
	x, err := a.ui.sv.Characters.GetCharacterAttributes(context.Background(), a.ui.currentCharID())
	if errors.Is(err, storage.ErrNotFound) {
		err := a.attributes.Set(make([]any, 0))
		if err != nil {
			return 0, err
		}
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	items := make([]any, 6)
	items[0] = attribute{
		icon:   resourcePerceptionPng,
		name:   "Perception",
		points: x.Perception,
	}
	items[1] = attribute{
		icon:   resourceMemoryPng,
		name:   "Memory",
		points: x.Memory,
	}
	items[2] = attribute{
		icon:   resourceWillpowerPng,
		name:   "Willpower",
		points: x.Willpower,
	}
	items[3] = attribute{
		icon:   resourceIntelligencePng,
		name:   "Intelligence",
		points: x.Intelligence,
	}
	items[4] = attribute{
		icon:   resourceCharismaPng,
		name:   "Charisma",
		points: x.Charisma,
	}
	items[5] = attribute{
		name: fmt.Sprintf("Bonus Remaps Available: %d", x.BonusRemaps),
	}
	if err := a.attributes.Set(items); err != nil {
		return 0, err
	}
	total := x.Charisma + x.Intelligence + x.Memory + x.Perception + x.Willpower
	return total, nil
}
