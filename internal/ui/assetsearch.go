package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/dustin/go-humanize"
)

// assetSearchArea is the UI area that shows the skillqueue
type assetSearchArea struct {
	content        *fyne.Container
	assets         []*model.CharacterAsset
	assetTree      map[int64]AssetNode
	assetLocations map[int64]int64
	assetTable     *widget.Table
	assetData      binding.UntypedList
	searchBox      *widget.Entry
	locations      map[int64]*model.EveLocation
	characterNames map[int32]string
	top            *widget.Label
	ui             *ui
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
	sb.OnChanged = func(search string) {
		if len(search) == 1 {
			return
		}
		xx := make([]*model.CharacterAsset, 0)
		search2 := strings.ToLower(search)
		for _, o := range a.assets {
			if strings.Contains(strings.ToLower(o.EveType.Name), search2) {
				xx = append(xx, o)
			}
		}
		a.assetData.Set(copyToUntypedSlice(xx))
		a.assetTable.Refresh()
		a.assetTable.ScrollToTop()
	}
	return sb
}

func (a *assetSearchArea) makeAssetsTable() *widget.Table {
	var headers = []struct {
		text  string
		width float32
	}{
		{"Name", 250},
		{"Location", 350},
		{"ID", 250},
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
			ca, err := getItemUntypedList[*model.CharacterAsset](a.assetData, tci.Row)
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
				label.Text = ca.EveType.Name
			case 1:
				var t string
				locationID, ok := a.assetLocations[ca.ItemID]
				if !ok {
					t = fmt.Sprintf("asset location not found for: %d", ca.ItemID)
				} else {
					x2, ok := a.locations[locationID]
					if !ok {
						t = fmt.Sprintf("location not found: %d", locationID)
					} else {
						t = x2.NamePlus()
					}
				}
				label.Text = t
			case 2:
				label.Text = fmt.Sprint(ca.ItemID)
			case 3:
				label.Text = a.characterNames[ca.CharacterID]
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
		// o, err := getItemUntypedList[*model.CharacterAsset](a.assetsData, tci.Row)
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
	if err := a.updateData(); err != nil {
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

func (a *assetSearchArea) updateData() error {
	if !a.ui.hasCharacter() {
		oo := make([]*model.CharacterAsset, 0)
		a.assetData.Set(copyToUntypedSlice(oo))
		a.searchBox.SetText("")
		return nil
	}
	ctx := context.Background()
	ca, err := a.ui.sv.Characters.ListAllCharacterAssets(ctx)
	if err != nil {
		return err
	}
	a.assets = ca
	a.assetTree = NewAssetTree(ca)
	a.assetLocations = CompileAssetParentLocations(a.assetTree)
	a.assetData.Set(copyToUntypedSlice(ca))

	el, err := a.ui.sv.EveUniverse.ListEveLocations(ctx)
	if err != nil {
		return err
	}
	m := make(map[int64]*model.EveLocation)
	for _, o := range el {
		m[o.ID] = o
	}
	a.locations = m

	cc, err := a.ui.sv.Characters.ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	m2 := make(map[int32]string)
	for _, o := range cc {
		m2[o.ID] = o.Name
	}
	a.characterNames = m2
	return nil
}

func (a *assetSearchArea) makeTopText() (string, widget.Importance) {
	text := fmt.Sprintf("%s total assets", humanize.Commaf(float64(len(a.assets))))
	return text, widget.MediumImportance
}
