package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

type infoWindow struct {
	content fyne.CanvasObject
	ui      *ui
	et      *model.EveType
	window  fyne.Window
}

func (u *ui) showTypeWindow(typeID int32) {
	iw, err := u.newInfoWindow(typeID)
	if err != nil {
		panic(err)
	}
	w := u.app.NewWindow(iw.title())
	w.SetContent(iw.content)
	w.Resize(fyne.Size{Width: 500, Height: 500})
	w.Show()
	iw.window = w
}

func (u *ui) newInfoWindow(typeID int32) (*infoWindow, error) {
	et, err := u.sv.EveUniverse.GetEveType(context.Background(), typeID)
	if err != nil {
		return nil, err
	}
	a := &infoWindow{ui: u, et: et}
	a.content = a.makeContent()
	return a, nil
}

func (a *infoWindow) makeContent() fyne.CanvasObject {
	description := widget.NewLabel(a.et.DescriptionPlain())
	description.Wrapping = fyne.TextWrapWord
	tabs := container.NewAppTabs(
		container.NewTabItem("Traits", widget.NewLabel("PLACEHOLDER")),
		container.NewTabItem("Description", container.NewVScroll(description)),
		container.NewTabItem("Attributes", widget.NewLabel("PLACEHOLDER")),
	)
	tabs.SelectIndex(1)
	icon := newImageResourceAsync(resourceQuestionmarkSvg, func() (fyne.Resource, error) {
		if a.et.IsSKIN() {
			return resourceSkinicon64pxPng, nil
		} else if a.et.IsBlueprint() {
			return a.ui.sv.EveImage.InventoryTypeBPO(a.et.ID, 64)
		} else {
			return a.ui.sv.EveImage.InventoryTypeIcon(a.et.ID, 64)
		}
	})
	icon.FillMode = canvas.ImageFillOriginal
	c := container.NewBorder(container.NewHBox(icon), nil, nil, nil, tabs)
	return c
}

func (a *infoWindow) title() string {
	return fmt.Sprintf("%s (%s): Information", a.et.Name, a.et.Group.Name)
}
