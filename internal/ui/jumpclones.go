package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/model"
)

const jumpClonesUpdateTicker = 10 * time.Second

// jumpClonesArea is the UI area that shows the skillqueue
type jumpClonesArea struct {
	content *fyne.Container
	list    *fyne.Container
	top     *widget.Label
	ui      *ui
}

func (u *ui) NewJumpClonesArea() *jumpClonesArea {
	a := jumpClonesArea{
		list: container.NewVBox(),
		top:  widget.NewLabel(""),
		ui:   u,
	}
	a.top.TextStyle.Bold = true

	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, container.NewVScroll(a.list))
	return &a
}

func (a *jumpClonesArea) Redraw() {
	a.list.RemoveAll()
	characterID := a.ui.CurrentCharID()
	if characterID == 0 {
		return
	}
	clones, err := a.ui.service.ListCharacterJumpClones(characterID)
	if err != nil {
		panic(err)
	}
	for _, c := range clones {
		row := container.NewVBox()
		row.Add(widget.NewLabel(c.Location.Name))
		if len(c.Implants) == 0 {
			x := widget.NewLabel("No implants")
			x.Importance = widget.LowImportance
			row.Add(x)
		} else {
			// data := make([]string, 0)
			// for _, i := range c.Implants {
			// 	data = append(data, i.EveType.Name)
			// }
			// implants := widget.NewList(
			// 	func() int {
			// 		return len(data)
			// 	},
			// 	func() fyne.CanvasObject {
			// 		return widget.NewLabel("Placeholder")
			// 	},
			// 	func(id widget.ListItemID, co fyne.CanvasObject) {
			// 		co.(*widget.Label).SetText(data[id])
			// 	},
			// )
			implants := container.NewVBox()
			for _, i := range c.Implants {
				icon := newImageResourceAsync(theme.AccountIcon(), func() (fyne.Resource, error) {
					return a.ui.imageManager.InventoryTypeIcon(i.EveType.ID, defaultIconSize)
				})
				icon.FillMode = canvas.ImageFillOriginal
				name := canvas.NewText(i.EveType.Name, theme.ForegroundColor())
				implants.Add(container.NewHBox(icon, name))
			}
			acc := widget.NewAccordion(
				widget.NewAccordionItem(
					fmt.Sprintf("%d implants", len(c.Implants)),
					implants,
				))
			row.Add(acc)
		}
		a.list.Add(row)
		a.list.Add(widget.NewSeparator())
	}
	a.list.Refresh()
	a.top.SetText(fmt.Sprintf("%d clones", len(clones)))
}

// func (a *jumpClonesArea) makeTopText() (string, widget.Importance) {
// 	errorText, err := a.errorText.Get()
// 	if err != nil {
// 		panic(err)
// 	}
// 	if errorText != "" {
// 		return errorText, widget.DangerImportance
// 	}
// 	hasData, err := a.ui.service.CharacterSectionWasUpdated(a.ui.CurrentCharID(), model.CharacterSectionJumpClones)
// 	if err != nil {
// 		return "ERROR", widget.DangerImportance
// 	}
// 	if !hasData {
// 		return "No data", widget.LowImportance
// 	}
// 	return fmt.Sprintf("%d clones", a.clones.Length()), widget.MediumImportance
// }

func (a *jumpClonesArea) StartUpdateTicker() {
	ticker := time.NewTicker(jumpClonesUpdateTicker)
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

func (a *jumpClonesArea) MaybeUpdateAndRefresh(characterID int32) {
	changed, err := a.ui.service.UpdateCharacterSectionIfExpired(characterID, model.CharacterSectionJumpClones)
	if err != nil {
		slog.Error("Failed to update jump clones", "character", characterID, "err", err)
		return
	}
	if changed && characterID == a.ui.CurrentCharID() {
		a.Redraw()
	}
}
