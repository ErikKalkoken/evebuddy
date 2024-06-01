package ui

import (
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
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

			q, err := convertDataItem[*model.CharacterImplant](di)
			if err != nil {
				slog.Error("failed to render row in implants table", "err", err)
				label.Text = "failed to render"
				label.Importance = widget.DangerImportance
				label.Refresh()
				return
			}
			label.SetText(fmt.Sprintf("%s\nSlot %d", q.EveType.Name, q.SlotNum))

			refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
				return a.ui.imageManager.InventoryTypeIcon(q.EveType.ID, 64)
			})
		})

	l.OnSelected = func(id widget.ListItemID) {
		implant, err := getItemUntypedList[*model.CharacterImplant](a.implants, id)
		if err != nil {
			t := "Failed to select implant"
			slog.Error(t, "err", err)
			a.ui.statusBarArea.SetError(t)
			return
		}
		d := makeTypeDetailDialog(implant.EveType.Name, implant.EveType.DescriptionPlain(), a.ui.window)
		d.SetOnClosed(func() {
			l.UnselectAll()
		})
		d.Show()
	}
	return l
}

func (a *implantsArea) refresh() {
	t, i, err := func() (string, widget.Importance, error) {
		if err := a.updateData(); err != nil {
			return "", 0, err
		}
		return a.makeTopText()
	}()
	if err != nil {
		slog.Error("Failed to refresh implants UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
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
	implants, err := a.ui.service.ListCharacterImplants(a.ui.currentCharID())
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

func (a *implantsArea) makeTopText() (string, widget.Importance, error) {
	hasData, err := a.ui.service.CharacterSectionWasUpdated(a.ui.currentCharID(), model.CharacterSectionImplants)
	if err != nil {
		return "", 0, err
	}
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	return fmt.Sprintf("%d implants", a.implants.Length()), widget.MediumImportance, nil
}

func makeTypeDetailDialog(title, description string, window fyne.Window) *dialog.CustomDialog {
	label := widget.NewLabel(description)
	label.Wrapping = fyne.TextWrapWord
	x := container.NewVScroll(label)
	x.SetMinSize(fyne.Size{Width: 600, Height: 250})
	d := dialog.NewCustom(title, "OK", x, window)
	return d
}
