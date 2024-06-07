package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service/character"
	"github.com/dustin/go-humanize"
)

// assetSearchArea is the UI area that shows the skillqueue
type assetSearchArea struct {
	assets          []*assetSearchRow
	assetTree       map[int64]character.AssetNode
	assetLocations  map[int64]int64
	assetTable      *widget.Table
	assetData       binding.UntypedList
	characterNames  map[int32]string
	content         *fyne.Container
	locations       map[int64]*model.EveLocation
	found           *widget.Label
	searchCharacter string
	searchGroup     string
	searchLocation  string
	searchType      string
	total           *widget.Label
	ui              *ui
}

func (u *ui) newAssetSearchArea() *assetSearchArea {
	a := &assetSearchArea{
		ui:        u,
		assetData: binding.NewUntypedList(),
		total:     widget.NewLabel(""),
		found:     widget.NewLabel(""),
	}
	a.assetData.AddListener(binding.NewDataListener(func() {
		t := a.assetData.Length()
		if t != len(a.assets) {
			a.found.SetText(fmt.Sprintf("%d found", t))
			a.found.Show()
		} else {
			a.found.Hide()
		}
	}))
	a.total.TextStyle.Bold = true
	a.found.Hide()
	a.assetTable = a.makeAssetsTable()
	topBox := container.NewVBox(container.NewHBox(a.total, a.found), widget.NewSeparator())
	a.content = container.NewBorder(topBox, nil, nil, nil, a.assetTable)
	return a
}

type assetSearchRow struct {
	characterID   int32
	characterName string
	groupID       int32
	groupName     string
	itemID        int64
	locationID    int64
	locationName  string
	name          string
	quantity      string
	typeID        int32
	typeName      string
}

func (a *assetSearchArea) newAssetSearchRow(ca *model.CharacterAsset) *assetSearchRow {
	r := &assetSearchRow{
		characterID:   ca.CharacterID,
		characterName: a.characterNames[ca.CharacterID],
		groupID:       ca.EveType.Group.ID,
		groupName:     ca.EveType.Group.Name,
		itemID:        ca.ItemID,
		name:          ca.DisplayName2(),
		typeID:        ca.EveType.ID,
		typeName:      ca.EveType.Name,
	}
	if ca.IsSingleton {
		r.quantity = ""
	} else {
		r.quantity = humanize.Comma(int64(ca.Quantity))
	}
	locationID, ok := a.assetLocations[ca.ItemID]
	var t string
	if !ok {
		t = "?"
	} else {
		x2, ok := a.locations[locationID]
		if !ok {
			t = "?"
		} else {
			t = x2.NamePlus()
		}
	}
	r.locationID = locationID
	r.locationName = t
	return r
}

func (a *assetSearchArea) makeAssetsTable() *widget.Table {
	var headers = []struct {
		text  string
		width float32
	}{
		{"Name", 350},
		{"Quantity", 75},
		{"Group", 250},
		{"Location", 350},
		{"Character", 200},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return a.assetData.Length(), len(headers)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("Template")
			label.Truncation = fyne.TextTruncateEllipsis
			return label
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			label := co.(*widget.Label)
			r, err := getItemUntypedList[*assetSearchRow](a.assetData, tci.Row)
			if err != nil {
				slog.Error("Failed to render asset item in UI", "err", err)
				label.SetText("ERROR")
				return
			}
			var t string
			var ta fyne.TextAlign
			switch tci.Col {
			case 0:
				t = r.name
			case 1:
				t = r.quantity
				ta = fyne.TextAlignTrailing
			case 2:
				t = r.groupName
			case 3:
				t = r.locationName
			case 4:
				t = r.characterName
			}
			label.Text = t
			label.Alignment = ta
			label.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.CreateHeader = func() fyne.CanvasObject {
		return container.NewBorder(nil, nil, widget.NewLabel("Template"), nil, widget.NewEntry())
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		row := co.(*fyne.Container)
		label := row.Objects[1].(*widget.Label)
		sb := row.Objects[0].(*widget.Entry)
		switch tci.Col {
		case 0:
			label.Hide()
			sb.SetPlaceHolder(s.text)
			sb.OnChanged = func(search string) {
				if len(search) == 1 {
					return
				}
				a.searchType = strings.ToLower(search)
				a.filterData()
			}
			sb.Show()
		case 2:
			label.Hide()
			sb.SetPlaceHolder(s.text)
			sb.OnChanged = func(search string) {
				if len(search) == 1 {
					return
				}
				a.searchGroup = strings.ToLower(search)
				a.filterData()
			}
			sb.Show()
		case 3:
			label.Hide()
			sb.SetPlaceHolder(s.text)
			sb.OnChanged = func(search string) {
				if len(search) == 1 {
					return
				}
				a.searchLocation = strings.ToLower(search)
				a.filterData()
			}
			sb.Show()
		case 4:
			label.Hide()
			sb.SetPlaceHolder(s.text)
			sb.OnChanged = func(search string) {
				if len(search) == 1 {
					return
				}
				a.searchCharacter = strings.ToLower(search)
				a.filterData()
			}
			sb.Show()
		default:
			label.SetText(s.text)
			label.Show()
			sb.Hide()
		}
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.width)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
		o, err := getItemUntypedList[*assetSearchRow](a.assetData, tci.Row)
		if err != nil {
			slog.Error("Failed to select asset", "err", err)
			return
		}
		switch tci.Col {
		case 0:
			a.ui.showTypeInfoWindow(o.typeID, a.ui.characterID())
		case 3:
			a.ui.showLocationInfoWindow(o.locationID)
		}
	}
	return t
}

func (a *assetSearchArea) filterData() {
	rows := make([]*assetSearchRow, 0)
	for _, r := range a.assets {
		matches := true
		if a.searchType != "" {
			matches = matches && strings.Contains(strings.ToLower(r.typeName), a.searchType)
		}
		if a.searchGroup != "" {
			matches = matches && strings.Contains(strings.ToLower(r.groupName), a.searchGroup)
		}
		if a.searchLocation != "" {
			matches = matches && strings.Contains(strings.ToLower(r.locationName), a.searchLocation)
		}
		if a.searchCharacter != "" {
			matches = matches && strings.Contains(strings.ToLower(r.characterName), a.searchCharacter)
		}
		if matches {
			rows = append(rows, r)
		}
	}
	a.assetData.Set(copyToUntypedSlice(rows))
	a.assetTable.Refresh()
	a.assetTable.ScrollToTop()
}

func (a *assetSearchArea) refresh() {
	var t string
	var i widget.Importance
	if err := a.updateData(); err != nil {
		slog.Error("Failed to refresh asset search data", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeTopText()
	}
	a.total.Text = t
	a.total.Importance = i
	a.total.Refresh()
	a.assetTable.Refresh()
}

func (a *assetSearchArea) updateData() error {
	if !a.ui.hasCharacter() {
		oo := make([]*model.CharacterAsset, 0)
		a.assetData.Set(copyToUntypedSlice(oo))
		// a.searchBox.SetText("")
		return nil
	}
	ctx := context.Background()
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
	assets, err := a.ui.sv.Characters.ListAllCharacterAssets(ctx)
	if err != nil {
		return err
	}
	a.assetTree = character.NewAssetTree(assets)
	a.assetLocations = character.CompileAssetParentLocations(a.assetTree)
	rows := make([]*assetSearchRow, len(assets))
	for i, ca := range assets {
		rows[i] = a.newAssetSearchRow(ca)
	}
	a.assetData.Set(copyToUntypedSlice(rows))
	a.assets = rows
	return nil
}

func (a *assetSearchArea) makeTopText() (string, widget.Importance) {
	text := fmt.Sprintf("%s total items", humanize.Comma(int64(len(a.assets))))
	return text, widget.MediumImportance
}
