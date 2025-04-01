package currentcharacter

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

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type CharacterImplants struct {
	widget.BaseWidget

	implants []*app.CharacterImplant
	list     *widget.List
	top      *widget.Label
	u        app.UI
}

func NewCharacterImplants(u app.UI) *CharacterImplants {
	a := &CharacterImplants{
		implants: make([]*app.CharacterImplant, 0),
		top:      appwidget.MakeTopLabel(),
		u:        u,
	}
	a.ExtendBaseWidget(a)
	a.list = a.makeImplantList()
	return a
}

func (a *CharacterImplants) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.list)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterImplants) makeImplantList() *widget.List {
	p := theme.Padding()
	l := widget.NewList(
		func() int {
			return len(a.implants)
		},
		func() fyne.CanvasObject {
			iconMain := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize*1.2))
			iconInfo := widget.NewIcon(theme.InfoIcon())
			name := widget.NewLabel("placeholder")
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
			name := vbox[0].(*fyne.Container).Objects[0].(*widget.Label)
			name.SetText(o.EveType.Name)
			slot := vbox[1].(*fyne.Container).Objects[0].(*widget.Label)
			slot.SetText(fmt.Sprintf("Slot %d", o.SlotNum))
			iconMain := row[1].(*canvas.Image)
			appwidget.RefreshImageResourceAsync(iconMain, func() (fyne.Resource, error) {
				return a.u.EveImageService().InventoryTypeIcon(o.EveType.ID, app.IconPixelSize)
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

func (a *CharacterImplants) Update() {
	var t string
	var i widget.Importance
	if err := a.updateImplants(); err != nil {
		slog.Error("Failed to refresh implants UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeTopText()
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
}

func (a *CharacterImplants) updateImplants() error {
	if !a.u.HasCharacter() {
		a.implants = make([]*app.CharacterImplant, 0)
		return nil
	}
	implants, err := a.u.CharacterService().ListCharacterImplants(context.TODO(), a.u.CurrentCharacterID())
	if err != nil {
		return err
	}
	a.implants = implants
	a.list.Refresh()
	return nil
}

func (a *CharacterImplants) makeTopText() (string, widget.Importance) {
	hasData := a.u.StatusCacheService().CharacterSectionExists(a.u.CurrentCharacterID(), app.SectionImplants)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	return fmt.Sprintf("%d implants", len(a.implants)), widget.MediumImportance
}
