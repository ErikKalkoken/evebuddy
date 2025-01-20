package desktop

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

// implantsArea is the UI area that shows the skillqueue
type implantsArea struct {
	content  *fyne.Container
	implants []*app.CharacterImplant
	list     *widget.List
	top      *widget.Label
	u        *DesktopUI
}

func (u *DesktopUI) newImplantsArea() *implantsArea {
	a := implantsArea{
		implants: make([]*app.CharacterImplant, 0),
		top:      widget.NewLabel(""),
		u:        u,
	}
	a.top.TextStyle.Bold = true
	a.list = a.makeImplantList()
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, a.list)
	return &a
}

func (a *implantsArea) makeImplantList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.implants)
		},
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromResource(ui.IconCharacterplaceholder32Jpeg)
			icon.FillMode = canvas.ImageFillContain
			icon.SetMinSize(fyne.Size{Width: 42, Height: 42})
			return container.NewHBox(icon, widget.NewLabel("placeholder\nslot"))
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.implants) {
				return
			}
			o := a.implants[id]
			row := co.(*fyne.Container).Objects
			icon := row[0].(*canvas.Image)
			label := row[1].(*widget.Label)
			label.SetText(fmt.Sprintf("%s\nSlot %d", o.EveType.Name, o.SlotNum))
			ui.RefreshImageResourceAsync(icon, func() (fyne.Resource, error) {
				return a.u.EveImageService.InventoryTypeIcon(o.EveType.ID, 64)
			})
		})

	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id >= len(a.implants) {
			return
		}
		o := a.implants[id]
		a.u.showTypeInfoWindow(o.EveType.ID, a.u.CharacterID(), descriptionTab)
	}
	return l
}

func (a *implantsArea) refresh() {
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

func (a *implantsArea) updateImplants() error {
	if !a.u.HasCharacter() {
		a.implants = make([]*app.CharacterImplant, 0)
		return nil
	}
	implants, err := a.u.CharacterService.ListCharacterImplants(context.TODO(), a.u.CharacterID())
	if err != nil {
		return err
	}
	a.implants = implants
	a.list.Refresh()
	return nil
}

func (a *implantsArea) makeTopText() (string, widget.Importance) {
	hasData := a.u.StatusCacheService.CharacterSectionExists(a.u.CharacterID(), app.SectionImplants)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	return fmt.Sprintf("%d implants", len(a.implants)), widget.MediumImportance
}
