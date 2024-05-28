package ui

import (
	"errors"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

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
	ui         *ui
}

func (u *ui) newAttributesArena() *attributesArea {
	a := attributesArea{
		attributes: binding.NewUntypedList(),
		ui:         u,
	}

	list := widget.NewListWithData(
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

	list.OnSelected = func(id widget.ListItemID) {
		list.UnselectAll()
	}

	a.content = list
	return &a
}

func (a *attributesArea) refresh() {
	if err := a.updateData(); err != nil {
		slog.Error("failed to render attributes for character", "characterID", a.ui.currentCharID(), "err", err)
		return
	}
}

func (a *attributesArea) updateData() error {
	if !a.ui.hasCharacter() {
		err := a.attributes.Set(make([]any, 0))
		if err != nil {
			return err
		}
	}
	x, err := a.ui.service.GetCharacterAttributes(a.ui.currentCharID())
	if errors.Is(err, storage.ErrNotFound) {
		err := a.attributes.Set(make([]any, 0))
		if err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
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
		panic(err)
	}
	return nil
}
