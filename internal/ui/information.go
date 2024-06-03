package ui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/dustin/go-humanize"
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
	w := u.app.NewWindow(iw.makeTitle("Information"))
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

func (a *infoWindow) makeTitle(suffix string) string {
	return fmt.Sprintf("%s (%s): %s", a.et.Name, a.et.Group.Name, suffix)
}

func (a *infoWindow) makeContent() fyne.CanvasObject {
	description := widget.NewLabel(a.et.DescriptionPlain())
	description.Wrapping = fyne.TextWrapWord
	tabs := container.NewAppTabs(
		container.NewTabItem("Traits", widget.NewLabel("PLACEHOLDER")),
		container.NewTabItem("Description", container.NewVScroll(description)),
		container.NewTabItem("Attributes", a.makeAttributesTab()),
	)
	tabs.SelectIndex(1)
	image := newImageResourceAsync(resourceQuestionmarkSvg, func() (fyne.Resource, error) {
		if a.et.IsSKIN() {
			return resourceSkinicon64pxPng, nil
		} else if a.et.IsBlueprint() {
			return a.ui.sv.EveImage.InventoryTypeBPO(a.et.ID, 64)
		} else {
			return a.ui.sv.EveImage.InventoryTypeIcon(a.et.ID, 64)
		}
	})
	image.FillMode = canvas.ImageFillOriginal
	b := widget.NewButton("Show", func() {
		w := a.ui.app.NewWindow(a.makeTitle("Render"))
		i := newImageResourceAsync(resourceQuestionmarkSvg, func() (fyne.Resource, error) {
			return a.ui.sv.EveImage.InventoryTypeRender(a.et.ID, 512)
		})
		i.FillMode = canvas.ImageFillContain
		s := float32(512) / w.Canvas().Scale()
		w.Resize(fyne.Size{Width: s, Height: s})
		w.SetContent(i)
		w.Show()
	})
	if a.et.HasRender() {
		b.Enable()
	} else {
		b.Disable()
	}
	c := container.NewBorder(container.NewHBox(image, b), nil, nil, nil, tabs)
	return c
}

type row struct {
	label   string
	value   string
	isTitle bool
}

func (a *infoWindow) makeAttributesTab() fyne.CanvasObject {
	data := make([]row, 0)
	oo, err := a.ui.sv.EveUniverse.ListEveTypeDogmaAttributesForType(context.Background(), a.et.ID)
	if err != nil {
		panic(err)
	}
	for _, o := range oo {
		data = append(data, row{label: o.DogmaAttribute.DisplayName, value: humanize.Commaf(float64(o.Value))})
	}
	box := container.NewVBox()
	for _, r := range data {
		if r.isTitle {
			label := widget.NewLabel(r.label)
			label.TextStyle.Bold = true
			label.Importance = widget.HighImportance
			box.Add(container.NewHBox(label))
		} else {
			box.Add(container.NewHBox(widget.NewLabel(r.label), layout.NewSpacer(), widget.NewLabel(r.value)))
		}
	}
	return container.NewVScroll(box)
}
