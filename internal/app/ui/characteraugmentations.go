package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type characterAugmentations struct {
	widget.BaseWidget

	character *app.Character
	implants  []*app.CharacterImplant
	list      *widget.List
	top       *widget.Label
	u         *baseUI
}

func newCharacterAugmentations(u *baseUI) *characterAugmentations {
	a := &characterAugmentations{
		implants: make([]*app.CharacterImplant, 0),
		top:      makeTopLabel(),
		u:        u,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeImplantList()
	a.u.characterExchanged.AddListener(
		func(_ context.Context, c *app.Character) {
			a.character = c
			a.update()
		},
	)
	a.u.characterSectionChanged.AddListener(
		func(_ context.Context, arg characterSectionUpdated) {
			if characterIDOrZero(a.character) != arg.characterID {
				return
			}
			if arg.section == app.SectionCharacterImplants {
				a.update()
			}
		},
	)
	return a
}

func (a *characterAugmentations) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.list)
	return widget.NewSimpleRenderer(c)
}

func (a *characterAugmentations) makeImplantList() *widget.List {
	p := theme.Padding()
	l := widget.NewList(
		func() int {
			return len(a.implants)
		},
		func() fyne.CanvasObject {
			iconMain := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize*1.2))
			iconInfo := widget.NewIcon(theme.InfoIcon())
			name := ttwidget.NewLabel("placeholder")
			name.Truncation = fyne.TextTruncateEllipsis
			slot := widget.NewLabel("placeholder")
			slot.Truncation = fyne.TextTruncateEllipsis
			return container.NewBorder(
				nil,
				nil,
				iconMain,
				iconInfo,
				container.New(
					layout.NewCustomPaddedVBoxLayout(0),
					container.New(layout.NewCustomPaddedLayout(0, -p, 0, 0), name),
					container.New(layout.NewCustomPaddedLayout(-p, 0, 0, 0), slot),
				),
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.implants) {
				return
			}
			o := a.implants[id]
			row := co.(*fyne.Container).Objects
			vbox := row[0].(*fyne.Container).Objects
			name := vbox[0].(*fyne.Container).Objects[0].(*ttwidget.Label)
			name.SetText(o.EveType.Name)
			name.SetToolTip(o.EveType.Description)
			slot := vbox[1].(*fyne.Container).Objects[0].(*widget.Label)
			slot.SetText(fmt.Sprintf("Slot %d", o.SlotNum))
			iconMain := row[1].(*canvas.Image)
			iwidget.RefreshImageAsync(iconMain, func() (fyne.Resource, error) {
				return a.u.eis.InventoryTypeIcon(o.EveType.ID, app.IconPixelSize)
			})
		})

	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.implants) {
			return
		}
		a.u.ShowTypeInfoWindow(a.implants[id].EveType.ID)
	}
	l.HideSeparators = true
	return l
}

func (a *characterAugmentations) update() {
	var err error
	implants := make([]*app.CharacterImplant, 0)
	characterID := characterIDOrZero(a.character)
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionCharacterImplants)
	if hasData {
		implants2, err2 := a.u.cs.ListImplants(context.Background(), characterID)
		if err2 != nil {
			slog.Error("Failed to refresh implants UI", "err", err)
			err = err2
		} else {
			implants = implants2
		}
	}
	t, i := a.u.makeTopText(characterID, hasData, err, func() (string, widget.Importance) {
		return fmt.Sprintf("%d implants", len(implants)), widget.MediumImportance
	})
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.implants = implants
		a.list.Refresh()
	})
}
