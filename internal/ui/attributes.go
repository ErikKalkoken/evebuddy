package ui

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

const attributesUpdateTicker = 10 * time.Second

type attribute struct {
	icon   fyne.Resource
	name   string
	points int
}

func (a attribute) IsText() bool {
	return a.points == 0
}

// attributesArea is the UI area that shows the skillqueue
type attributesArea struct {
	content    fyne.CanvasObject
	attributes binding.UntypedList
	ui         *ui
}

func (u *ui) NewAttributesArea() *attributesArea {
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
			if q.IsText() {
				icon.Hide()
			} else {
				icon.Show()
				icon.Resource = q.icon
				icon.Refresh()
			}

			points := row.Objects[3].(*widget.Label)
			if q.IsText() {
				points.Hide()
			} else {
				points.Show()
				points.SetText(fmt.Sprintf("%d points", q.points))
			}
		})

	list.OnSelected = func(id widget.ListItemID) {
		x, err := getFromBoundUntypedList[attribute](a.attributes, id)
		if err != nil {
			slog.Error("failed to access implant item in list", "err", err)
			return
		}
		if x.IsText() {
			list.UnselectAll()
			return
		}
		var data = []struct {
			label string
			value string
			wrap  bool
		}{
			{"Name", x.name, false},
			{"Description", "tbd", true},
			{"Points", strconv.Itoa(x.points), true},
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
		dlg := dialog.NewCustom("Attribute Detail", "OK", s, u.window)
		dlg.SetOnClosed(func() {
			list.UnselectAll()
		})
		dlg.Show()
		dlg.Resize(fyne.Size{
			Width:  0.8 * a.ui.window.Canvas().Size().Width,
			Height: 0.8 * a.ui.window.Canvas().Size().Height,
		})
	}

	a.content = list
	return &a
}

func (a *attributesArea) Refresh() {
	err := a.updateData()
	if err != nil {
		slog.Error("failed to render attributes for character", "characterID", a.ui.CurrentCharID(), "err", err)
		return
	}
}

func (a *attributesArea) updateData() error {
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		err := a.attributes.Set(make([]any, 0))
		if err != nil {
			return err
		}
	}
	x, err := a.ui.service.GetCharacterAttributes(characterID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			err := a.attributes.Set(make([]any, 0))
			if err != nil {
				return err
			}
			return nil
		}
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

func (a *attributesArea) StartUpdateTicker() {
	ticker := time.NewTicker(attributesUpdateTicker)
	go func() {
		for {
			func() {
				cc, err := a.ui.service.ListCharactersShort()
				if err != nil {
					slog.Error("Failed to fetch list of characters", "err", err)
					return
				}
				for _, c := range cc {
					a.MaybeUpdateAndRefresh(c.ID)
				}
			}()
			<-ticker.C
		}
	}()
}

func (a *attributesArea) MaybeUpdateAndRefresh(characterID int32) {
	changed, err := a.ui.service.UpdateCharacterSectionIfExpired(characterID, model.CharacterSectionAttributes)
	if err != nil {
		slog.Error("Failed to update attributes", "character", characterID, "err", err)
		return
	}
	if changed && characterID == a.ui.CurrentCharID() {
		a.Refresh()
	}
}