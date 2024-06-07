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
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

// assetSearchArea is the UI area that shows the skillqueue
type assetSearchArea struct {
	content    *fyne.Container
	assets     []*model.CharacterAsset
	assetTable *widget.Table
	assetData  binding.UntypedList
	searchBox  *widget.Entry
	top        *widget.Label
	ui         *ui
}

func (u *ui) newAssetSearchArea() *assetSearchArea {
	a := &assetSearchArea{
		ui:        u,
		assetData: binding.NewUntypedList(),
		top:       widget.NewLabel(""),
	}
	a.top.TextStyle.Bold = true
	a.searchBox = a.makeSearchBox()
	a.assetTable = a.makeAssetsTable()
	topBox := container.NewVBox(a.top, widget.NewSeparator(), a.searchBox)
	a.content = container.NewBorder(topBox, nil, nil, nil, a.assetTable)
	return a
}

func (a *assetSearchArea) makeSearchBox() *widget.Entry {
	sb := widget.NewEntry()
	sb.SetPlaceHolder("Filter by item type")
	sb.OnChanged = func(s string) {
		// if len(s) == 1 {
		// 	return
		// }
		// if err := a.updateAssets(); err != nil {
		// 	t := "Failed to update asset search box"
		// 	slog.Error(t, "err", err)
		// 	a.ui.statusBarArea.SetError(t)
		// }
		// a.assetTable.Refresh()
		// a.assetTable.ScrollToTop()
	}
	return sb
}

func (a *assetSearchArea) makeAssetsTable() *widget.Table {
	var headers = []struct {
		text  string
		width float32
	}{
		{"Name", 250},
		{"Location", 250},
		{"Character", 250},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return a.assetData.Length(), len(headers)
		},
		func() fyne.CanvasObject {
			icon := canvas.NewImageFromResource(resourceCharacterplaceholder32Jpeg)
			icon.FillMode = canvas.ImageFillOriginal
			icon.Hide()
			label := widget.NewLabel("Template")
			return container.NewHBox(icon, label)
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			// icon := row.Objects[0].(*canvas.Image)
			label := row.Objects[1].(*widget.Label)
			ca, err := getItemUntypedList[*model.CharacterSearchAsset](a.assetData, tci.Row)
			if err != nil {
				slog.Error("Failed to render asset item in UI", "err", err)
				label.SetText("ERROR")
				return
			}
			switch tci.Col {
			case 0:
				// refreshImageResourceAsync(icon, func() (fyne.Resource, error) {
				// 	return a.ui.sv.EveImage.InventoryTypeIcon(o.Type.ID, defaultIconSize)
				// })
				// icon.Show()
				label.Text = ca.Asset.EveType.Name
			case 1:
				label.Text = entityNameOrFallback(ca.Location, "?")
			case 2:
				label.Text = ca.Character.Name
			}
			label.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		co.(*widget.Label).SetText(s.text)
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.width)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
		// o, err := getItemUntypedList[*model.CharacterSearchAsset](a.assetsData, tci.Row)
		// if err != nil {
		// 	slog.Error("Failed to select asset", "err", err)
		// 	return
		// }
		// a.ui.showTypeInfoWindow(o.Type.ID, a.ui.characterID())
	}
	return t
}

func (a *assetSearchArea) refresh() {
	var t string
	var i widget.Importance
	if err := a.updateAssets(); err != nil {
		slog.Error("Failed to refresh ships UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeTopText()
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.assetTable.Refresh()
	// if enabled {
	// 	a.searchBox.Enable()
	// } else {
	// 	a.searchBox.Disable()
	// }
}

func (a *assetSearchArea) updateAssets() error {
	if !a.ui.hasCharacter() {
		oo := make([]*model.CharacterSearchAsset, 0)
		a.assetData.Set(copyToUntypedSlice(oo))
		a.searchBox.SetText("")
		return nil
	}
	oo, err := a.ui.sv.Characters.ListAllCharacterAssets(context.Background())
	if err != nil {
		return err
	}
	a.assets = oo
	return nil
}

func (a *assetSearchArea) makeTopText() (string, widget.Importance) {
	text := fmt.Sprintf("%d total assets", len(a.assets))
	return text, widget.MediumImportance
}
