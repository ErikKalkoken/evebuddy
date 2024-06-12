package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/assettree"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/dustin/go-humanize"
)

type assetSortDir uint

const (
	sortOff assetSortDir = iota
	sortAsc
	sortDesc
)

// assetSearchArea is the UI area that shows the skillqueue
type assetSearchArea struct {
	assets         []*assetSearchRow
	assetTree      assettree.AssetTree
	assetTable     *widget.Table
	assetData      binding.UntypedList
	colSort        []assetSortDir
	characterNames map[int32]string
	content        *fyne.Container
	iconSortAsc    fyne.Resource
	iconSortDesc   fyne.Resource
	iconSortOff    fyne.Resource
	found          *widget.Label
	colSearch      []string
	searchBoxes    []*widget.Entry
	total          *widget.Label
	ui             *ui
}

type assetSearchRow struct {
	characterID     int32
	characterName   string
	groupID         int32
	groupName       string
	itemID          int64
	locationID      int64
	locationName    string
	name            string
	quantity        int
	quantityDisplay string
	price           float64
	priceDisplay    string
	typeID          int32
	typeName        string
}

func (u *ui) newAssetSearchArea() *assetSearchArea {
	a := &assetSearchArea{
		ui:           u,
		assetData:    binding.NewUntypedList(),
		iconSortAsc:  theme.MoveUpIcon(),
		iconSortDesc: theme.MoveDownIcon(),
		iconSortOff:  theme.NewThemedResource(resourceBlankSvg),
		total:        widget.NewLabel(""),
		found:        widget.NewLabel(""),
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
	b := widget.NewButton("Reset", func() {
		a.resetSearch()
	})
	topBox := container.NewVBox(container.NewHBox(a.total, a.found, layout.NewSpacer(), b), widget.NewSeparator())
	a.content = container.NewBorder(topBox, nil, nil, nil, a.assetTable)
	return a
}

func (a *assetSearchArea) newAssetSearchRow(ca *model.CharacterAsset) *assetSearchRow {
	r := &assetSearchRow{
		characterID:   ca.CharacterID,
		characterName: a.characterNames[ca.CharacterID],
		groupID:       ca.EveType.Group.ID,
		groupName:     ca.EveType.Group.Name,
		itemID:        ca.ItemID,
		name:          ca.DisplayName2(),
		price:         ca.Price.Float64,
		typeID:        ca.EveType.ID,
		typeName:      ca.EveType.Name,
	}
	if ca.IsSingleton {
		r.quantityDisplay = ""
		r.quantity = 1
	} else {
		r.quantityDisplay = humanize.Comma(int64(ca.Quantity))
		r.quantity = int(ca.Quantity)
	}
	location, ok := a.assetTree.AssetParentLocation(ca.ItemID)
	var t string
	if !ok {
		t = "?"
	} else {
		t = location.DisplayName()
	}
	r.locationID = location.ID
	r.locationName = t
	var price string
	if !ca.Price.Valid || ca.IsBlueprintCopy {
		price = "?"
	} else {
		t := ca.Price.Float64 * float64(ca.Quantity)
		price = ihumanize.Number(t, 1)
	}
	r.priceDisplay = price
	return r
}

func (a *assetSearchArea) makeAssetsTable() *widget.Table {
	var headers = []struct {
		text  string
		width float32
	}{
		{"Name", 300},
		{"Quantity", 75},
		{"Group", 200},
		{"Location", 350},
		{"Character", 200},
		{"Price", 100},
	}
	a.colSearch = make([]string, len(headers))
	a.colSort = make([]assetSortDir, len(headers))
	for i := range headers {
		a.colSort[i] = sortOff
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
				t = r.quantityDisplay
				ta = fyne.TextAlignTrailing
			case 2:
				t = r.groupName
			case 3:
				t = r.locationName
			case 4:
				t = r.characterName
			case 5:
				t = r.priceDisplay
			}
			label.Text = t
			label.Alignment = ta
			label.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.CreateHeader = func() fyne.CanvasObject {
		sb := widget.NewEntry()
		a.searchBoxes = append(a.searchBoxes, sb)
		b := widget.NewButtonWithIcon("", a.iconSortOff, func() {})
		return container.NewBorder(nil, nil, widget.NewLabel("Template"), b, sb)
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		row := co.(*fyne.Container)
		sb := row.Objects[0].(*widget.Entry)
		label := row.Objects[1].(*widget.Label)
		switch tci.Col {
		case 0, 2, 3, 4:
			label.Hide()
			sb.SetPlaceHolder(s.text)
			sb.OnChanged = func(search string) {
				if len(search) == 1 {
					return
				}
				a.colSearch[tci.Col] = strings.ToLower(search)
				a.processData(-1)
			}
			sb.Show()
		default:
			label.SetText(s.text)
			label.Show()
			sb.Hide()
		}
		button := row.Objects[2].(*widget.Button)
		switch a.colSort[tci.Col] {
		case sortOff:
			button.SetIcon(a.iconSortOff)
		case sortAsc:
			button.SetIcon(a.iconSortAsc)
		case sortDesc:
			button.SetIcon(a.iconSortDesc)
		}
		button.OnTapped = func() {
			a.processData(tci.Col)
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

func (a *assetSearchArea) processData(sortCol int) {
	var order assetSortDir
	if sortCol >= 0 {
		order = a.colSort[sortCol]
		order++
		if order > sortDesc {
			order = sortOff
		}
		for i := range a.colSort {
			a.colSort[i] = sortOff
		}
		a.colSort[sortCol] = order
	} else {
		for i := range a.colSort {
			if a.colSort[i] != sortOff {
				order = a.colSort[i]
				sortCol = i
				break
			}
		}
	}
	rows := make([]*assetSearchRow, 0)
	for _, r := range a.assets {
		matches := true
		for i, search := range a.colSearch {
			var cell string
			switch i {
			case 0:
				cell = r.typeName
			case 2:
				cell = r.groupName
			case 3:
				cell = r.locationName
			case 4:
				cell = r.characterName
			}
			if search != "" {
				matches = matches && strings.Contains(strings.ToLower(cell), search)
			}
		}
		if matches {
			rows = append(rows, r)
		}
	}
	if sortCol >= 0 && order != sortOff {
		slices.SortFunc(rows, func(a, b *assetSearchRow) int {
			var x int
			switch sortCol {
			case 0:
				x = cmp.Compare(a.name, b.name)
			case 1:
				x = cmp.Compare(a.quantity, b.quantity)
			case 2:
				x = cmp.Compare(a.groupName, b.groupName)
			case 3:
				x = cmp.Compare(a.locationName, b.locationName)
			case 4:
				x = cmp.Compare(a.characterName, b.characterName)
			case 5:
				x = cmp.Compare(a.price, b.price)
			}
			if order == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	}
	a.assetData.Set(copyToUntypedSlice(rows))
	a.assetTable.Refresh()
	a.assetTable.ScrollToTop()
}

func (a *assetSearchArea) resetSearch() {
	for i := range a.colSort {
		a.colSort[i] = sortOff
	}
	for _, w := range a.searchBoxes {
		w.SetText("")
	}
	a.processData(-1)
}

func (a *assetSearchArea) refresh() {
	var t string
	var i widget.Importance
	if err := a.loadData(); err != nil {
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

func (a *assetSearchArea) loadData() error {
	if !a.ui.hasCharacter() {
		oo := make([]*model.CharacterAsset, 0)
		a.assetData.Set(copyToUntypedSlice(oo))
		return nil
	}
	ctx := context.Background()
	cc, err := a.ui.sv.Character.ListCharactersShort(ctx)
	if err != nil {
		return err
	}
	m2 := make(map[int32]string)
	for _, o := range cc {
		m2[o.ID] = o.Name
	}
	a.characterNames = m2
	assets, err := a.ui.sv.Character.ListAllCharacterAssets(ctx)
	if err != nil {
		return err
	}
	locations, err := a.ui.sv.EveUniverse.ListEveLocations(ctx)
	if err != nil {
		return err
	}
	a.assetTree = assettree.New(assets, locations)
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
