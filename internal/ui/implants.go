package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

const implantsUpdateTicker = 10 * time.Second

// implantsArea is the UI area that shows the skillqueue
type implantsArea struct {
	content   *fyne.Container
	implants  binding.UntypedList
	errorText binding.String
	top       *widget.Label
	ui        *ui
}

func (u *ui) NewImplantsArea() *implantsArea {
	a := implantsArea{
		implants:  binding.NewUntypedList(),
		errorText: binding.NewString(),
		top:       widget.NewLabel(""),
		ui:        u,
	}
	a.top.TextStyle.Bold = true
	list := widget.NewListWithData(
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

	list.OnSelected = func(id widget.ListItemID) {
		implant, err := getFromBoundUntypedList[*model.CharacterImplant](a.implants, id)
		if err != nil {
			slog.Error("failed to access implant item in list", "err", err)
			return
		}
		d := makeImplantDetailDialog(implant.EveType.Name, implant.EveType.DescriptionPlain(), a.ui.window)
		d.SetOnClosed(func() {
			list.UnselectAll()
		})
		d.Show()
	}

	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, list)
	return &a
}

func (a *implantsArea) Refresh() {
	err := a.updateData()
	if err != nil {
		slog.Error("failed to update skillqueue items for character", "characterID", a.ui.CurrentCharID(), "err", err)
		return
	}
	t, i := a.makeTopText()
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
}

func (a *implantsArea) updateData() error {
	if err := a.errorText.Set(""); err != nil {
		return err
	}
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		err := a.implants.Set(make([]any, 0))
		if err != nil {
			return err
		}
	}
	implants, err := a.ui.service.ListCharacterImplants(characterID)
	if err != nil {
		return err
	}
	items := make([]any, len(implants))
	for i, o := range implants {
		items[i] = o
	}
	if err := a.implants.Set(items); err != nil {
		panic(err)
	}
	return nil
}

func (a *implantsArea) makeTopText() (string, widget.Importance) {
	errorText, err := a.errorText.Get()
	if err != nil {
		panic(err)
	}
	if errorText != "" {
		return errorText, widget.DangerImportance
	}
	hasData, err := a.ui.service.CharacterSectionWasUpdated(a.ui.CurrentCharID(), model.CharacterSectionImplants)
	if err != nil {
		return "ERROR", widget.DangerImportance
	}
	if !hasData {
		return "No data", widget.LowImportance
	}
	return fmt.Sprintf("%d implants", a.implants.Length()), widget.MediumImportance
}

func (a *implantsArea) StartUpdateTicker() {
	ticker := time.NewTicker(implantsUpdateTicker)
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

func (a *implantsArea) MaybeUpdateAndRefresh(characterID int32) {
	changed, err := a.ui.service.UpdateCharacterSectionIfExpired(characterID, model.CharacterSectionImplants)
	if err != nil {
		slog.Error("Failed to update implants", "character", characterID, "err", err)
		return
	}
	if changed && characterID == a.ui.CurrentCharID() {
		a.Refresh()
	}
}

func makeImplantDetailDialog(name, description string, window fyne.Window) *dialog.CustomDialog {
	label := widget.NewLabel(description)
	label.Wrapping = fyne.TextWrapWord
	x := container.NewVScroll(label)
	x.SetMinSize(fyne.Size{Width: 600, Height: 250})
	d := dialog.NewCustom(name, "OK", x, window)
	return d
}
