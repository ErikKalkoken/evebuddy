package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// implantsArea is the UI area that shows the skillqueue
type implantsArea struct {
	content  *fyne.Container
	implants binding.UntypedList
	top      *widget.Label
	ui       *ui
}

func (u *ui) newImplantsArea() *implantsArea {
	a := implantsArea{
		implants: binding.NewUntypedList(),
		top:      widget.NewLabel(""),
		ui:       u,
	}
	a.top.TextStyle.Bold = true
	list := a.makeImplantList()
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, list)
	return &a
}

func (a *implantsArea) makeImplantList() *widget.List {
	l := widget.NewListWithData(
		a.implants,
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
			icon.FillMode = canvas.ImageFillContain
			icon.SetMinSize(fyne.Size{Width: 42, Height: 42})
			return container.NewHBox(icon, widget.NewLabel("placeholder\nslot"))
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			row := co.(*fyne.Container)

			icon := row.Objects[0].(*canvas.Image)
			label := row.Objects[1].(*widget.Label)

			q, err := convertDataItem[*app.CharacterImplant](di)
			if err != nil {
				slog.Error("failed to render row in implants table", "err", err)
				label.Text = "failed to render"
				label.Importance = widget.DangerImportance
				label.Refresh()
				return
			}
			label.SetText(fmt.Sprintf("%s\nSlot %d", q.EveType.Name, q.SlotNum))

			refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
				return a.ui.sv.EveImage.InventoryTypeIcon(q.EveType.ID, 64)
			})
		})

	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		o, err := getItemUntypedList[*app.CharacterImplant](a.implants, id)
		if err != nil {
			slog.Error("Failed to select implant", "err", err)
			return
		}
		a.ui.showTypeInfoWindow(o.EveType.ID, a.ui.characterID())
	}
	return l
}

func (a *implantsArea) refresh() {
	var t string
	var i widget.Importance
	if err := a.updateData(); err != nil {
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

func (a *implantsArea) updateData() error {
	if !a.ui.hasCharacter() {
		err := a.implants.Set(make([]any, 0))
		if err != nil {
			return err
		}
	}
	implants, err := a.ui.sv.Character.ListCharacterImplants(context.Background(), a.ui.characterID())
	if err != nil {
		return err
	}
	items := make([]any, len(implants))
	for i, o := range implants {
		items[i] = o
	}
	if err := a.implants.Set(items); err != nil {
		return err
	}
	return nil
}

func (a *implantsArea) makeTopText() (string, widget.Importance) {
	hasData := a.ui.sv.StatusCache.CharacterSectionExists(a.ui.characterID(), app.SectionImplants)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	return fmt.Sprintf("%d implants", a.implants.Length()), widget.MediumImportance
}
