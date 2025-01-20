package desktop

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
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
	assets          []*assetSearchRow
	assetCollection assetcollection.AssetCollection
	assetTable      *widget.Table
	assetsFiltered  []*assetSearchRow
	colSort         []assetSortDir
	characterNames  map[int32]string
	content         *fyne.Container
	iconSortAsc     fyne.Resource
	iconSortDesc    fyne.Resource
	iconSortOff     fyne.Resource
	found           *widget.Label
	colSearch       []string
	searchBoxes     []*widget.Entry
	total           *widget.Label
	u               *DesktopUI
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

func (u *DesktopUI) newAssetSearchArea() *assetSearchArea {
	a := &assetSearchArea{
		u:              u,
		assetsFiltered: make([]*assetSearchRow, 0),
		iconSortAsc:    theme.NewThemedResource(ui.IconSortAscendingSvg),
		iconSortDesc:   theme.NewThemedResource(ui.IconSortDescendingSvg),
		iconSortOff:    theme.NewThemedResource(ui.IconSortSvg),
		total:          widget.NewLabel(""),
		found:          widget.NewLabel(""),
	}
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

func (a *assetSearchArea) newAssetSearchRow(ca *app.CharacterAsset) *assetSearchRow {
	r := &assetSearchRow{
		characterID:   ca.CharacterID,
		characterName: a.characterNames[ca.CharacterID],
		groupID:       ca.EveType.Group.ID,
		groupName:     ca.EveType.Group.Name,
		itemID:        ca.ItemID,
		name:          ca.DisplayName2(),
		price:         ca.Price.ValueOrZero(),
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
	location, ok := a.assetCollection.AssetParentLocation(ca.ItemID)
	if !ok {
		r.locationName = "?"
	} else {
		r.locationID = location.ID
		r.locationName = location.DisplayName()
	}
	var price string
	if ca.Price.IsEmpty() || ca.IsBlueprintCopy {
		price = "?"
	} else {
		t := ca.Price.ValueOrZero() * float64(ca.Quantity)
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
		{"Qty.", 75},
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
			return len(a.assetsFiltered), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			if tci.Row >= len(a.assetsFiltered) || tci.Row < 0 {
				return
			}
			r := a.assetsFiltered[tci.Row]
			l := co.(*widget.Label)
			l.Truncation = fyne.TextTruncateClip
			l.Alignment = fyne.TextAlignLeading
			var t string
			switch tci.Col {
			case 0:
				t = r.name
			case 1:
				t = r.quantityDisplay
				l.Alignment = fyne.TextAlignTrailing
			case 2:
				t = r.groupName
			case 3:
				t = r.locationName
			case 4:
				t = r.characterName
			case 5:
				t = r.priceDisplay
				l.Alignment = fyne.TextAlignTrailing
			}
			l.Text = t
			l.Refresh()
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
		row := co.(*fyne.Container).Objects
		sb := row[0].(*widget.Entry)
		label := row[1].(*widget.Label)
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
		button := row[2].(*widget.Button)
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
		if tci.Row >= len(a.assetsFiltered) || tci.Row < 0 {
			return
		}
		r := a.assetsFiltered[tci.Row]
		switch tci.Col {
		case 0:
			a.u.showTypeInfoWindow(r.typeID, a.u.CharacterID(), descriptionTab)
		case 3:
			a.u.showLocationInfoWindow(r.locationID)
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
	a.assetsFiltered = rows
	a.updateFoundInfo()
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
	characterCount := a.characterCount()
	hasData, err := a.loadData()
	if err != nil {
		slog.Error("Failed to refresh asset search data", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else if !hasData {
		t = "No data"
		i = widget.LowImportance
	} else if characterCount == 0 {
		t = "No characters"
		i = widget.LowImportance
	} else {
		t, i = a.makeTopText(characterCount)
	}
	a.total.Text = t
	a.total.Importance = i
	a.total.Refresh()
	a.assetTable.Refresh()
}

func (a *assetSearchArea) loadData() (bool, error) {
	ctx := context.TODO()
	cc, err := a.u.CharacterService.ListCharactersShort(ctx)
	if err != nil {
		return false, err
	}
	if len(cc) == 0 {
		return false, nil
	}
	m2 := make(map[int32]string)
	for _, o := range cc {
		m2[o.ID] = o.Name
	}
	a.characterNames = m2
	assets, err := a.u.CharacterService.ListAllCharacterAssets(ctx)
	if err != nil {
		return false, err
	}
	locations, err := a.u.EveUniverseService.ListEveLocations(ctx)
	if err != nil {
		return false, err
	}
	a.assetCollection = assetcollection.New(assets, locations)
	rows := make([]*assetSearchRow, len(assets))
	for i, ca := range assets {
		rows[i] = a.newAssetSearchRow(ca)
	}
	a.assetsFiltered = rows
	a.assets = rows
	a.updateFoundInfo()
	return true, nil
}

func (a *assetSearchArea) updateFoundInfo() {
	if c := len(a.assetsFiltered); c < len(a.assets) {
		a.found.SetText(fmt.Sprintf("%d found", c))
		a.found.Show()
	} else {
		a.found.Hide()
	}
}

func (a *assetSearchArea) characterCount() int {
	cc := a.u.StatusCacheService.ListCharacters()
	validCount := 0
	for _, c := range cc {
		if a.u.StatusCacheService.CharacterSectionExists(c.ID, app.SectionAssets) {
			validCount++
		}
	}
	return validCount
}

func (a *assetSearchArea) makeTopText(c int) (string, widget.Importance) {
	it := humanize.Comma(int64(len(a.assets)))
	text := fmt.Sprintf("%d characters • %s items", c, it)
	return text, widget.MediumImportance
}
