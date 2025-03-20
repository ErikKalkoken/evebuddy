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
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// TODO: Mobile: Add column sort

type assetSortDir uint

const (
	sortOff assetSortDir = iota
	sortAsc
	sortDesc
)

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

type AllAssetSearch struct {
	widget.BaseWidget

	assetCollection assetcollection.AssetCollection
	assets          []*assetSearchRow
	assetsFiltered  []*assetSearchRow
	body            fyne.CanvasObject
	characterNames  map[int32]string
	colSort         []assetSortDir
	found           *widget.Label
	entry           *widget.Entry
	total           *widget.Label
	u               app.UI
}

func NewAssetSearch(u app.UI) *AllAssetSearch {
	a := &AllAssetSearch{
		assetsFiltered: make([]*assetSearchRow, 0),
		entry:          widget.NewEntry(),
		found:          widget.NewLabel(""),
		total:          MakeTopLabel(),
		u:              u,
	}
	a.ExtendBaseWidget(a)
	a.entry.ActionItem = iwidget.NewIconButton(theme.CancelIcon(), func() {
		a.resetSearch()
	})
	a.entry.OnChanged = func(s string) {
		a.processData(-1)
	}
	a.entry.PlaceHolder = "Search assets"
	a.total.TextStyle.Bold = true
	a.found.Hide()
	var headers = []iwidget.HeaderDef{
		{Text: "Item", Width: 300},
		{Text: "Class", Width: 200},
		{Text: "Location", Width: 350},
		{Text: "Owner", Width: 200},
		{Text: "Qty.", Width: 75},
		{Text: "Price", Width: 100},
	}
	makeDataLabel := func(col int, r *assetSearchRow) (string, fyne.TextAlign, widget.Importance) {
		var a fyne.TextAlign
		var i widget.Importance
		var t string
		switch col {
		case 0:
			t = r.name
		case 1:
			t = r.groupName
		case 2:
			t = r.locationName
		case 3:
			t = r.characterName
		case 4:
			t = r.quantityDisplay
			a = fyne.TextAlignTrailing
		case 5:
			t = r.priceDisplay
			a = fyne.TextAlignTrailing
		}
		return t, a, i
	}
	if a.u.IsMobile() {
		a.body = iwidget.MakeDataTableForMobile(headers, &a.assetsFiltered, makeDataLabel, func(r *assetSearchRow) {
			a.u.ShowTypeInfoWindow(r.typeID)
		})
	} else {
		// can't use helper here, because we also need sort
		a.body = a.makeTable(headers, makeDataLabel, func(col int, r *assetSearchRow) {
			switch col {
			case 0:
				a.u.ShowInfoWindow(app.EveEntityInventoryType, r.typeID)
			case 2:
				a.u.ShowLocationInfoWindow(r.locationID)
			case 3:
				a.u.ShowInfoWindow(app.EveEntityCharacter, r.characterID)
			}
		})
	}
	return a
}

func (a *AllAssetSearch) CreateRenderer() fyne.WidgetRenderer {
	topBox := container.NewVBox(
		container.NewBorder(nil, nil, nil, a.found, a.total),
		a.entry,
		widget.NewSeparator(),
	)
	c := container.NewBorder(topBox, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *AllAssetSearch) Focus() {
	a.u.MainWindow().Canvas().Focus(a.entry)
}

func (a *AllAssetSearch) makeTable(
	headers []iwidget.HeaderDef,
	makeDataLabel func(int, *assetSearchRow) (string, fyne.TextAlign, widget.Importance),
	onSelected func(int, *assetSearchRow),
) *widget.Table {
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
			l.Text, l.Alignment, l.Importance = makeDataLabel(tci.Col, r)
			l.Truncation = fyne.TextTruncateClip
			l.Refresh()
		},
	)
	t.ShowHeaderRow = true
	iconSortAsc := theme.NewPrimaryThemedResource(icons.SortAscendingSvg)
	iconSortDesc := theme.NewPrimaryThemedResource(icons.SortDescendingSvg)
	iconSortOff := theme.NewThemedResource(icons.SortSvg)
	t.CreateHeader = func() fyne.CanvasObject {
		b := widget.NewButtonWithIcon("", iconSortOff, func() {})
		return container.NewBorder(nil, nil, nil, b, widget.NewLabel("Template"))
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		h := headers[tci.Col]
		row := co.(*fyne.Container).Objects
		label := row[0].(*widget.Label)
		label.SetText(h.Text)
		button := row[1].(*widget.Button)
		switch a.colSort[tci.Col] {
		case sortOff:
			button.SetIcon(iconSortOff)
		case sortAsc:
			button.SetIcon(iconSortAsc)
		case sortDesc:
			button.SetIcon(iconSortDesc)
		}
		button.OnTapped = func() {
			a.processData(tci.Col)
		}
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.Width)
	}
	t.OnSelected = func(id widget.TableCellID) {
		defer t.UnselectAll()
		if id.Row >= len(a.assetsFiltered) || id.Row < 0 {
			return
		}
		r := a.assetsFiltered[id.Row]
		onSelected(id.Col, r)
	}
	return t
}

func (a *AllAssetSearch) processData(sortCol int) {
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
	search := a.entry.Text
	for _, r := range a.assets {
		var matches bool
		if search == "" {
			matches = true
		}
		for _, cell := range []string{r.typeName, r.groupName, r.locationName, r.characterName} {
			if search != "" {
				matches = matches || strings.Contains(strings.ToLower(cell), search)
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
				x = cmp.Compare(a.groupName, b.groupName)
			case 2:
				x = cmp.Compare(a.locationName, b.locationName)
			case 3:
				x = cmp.Compare(a.characterName, b.characterName)
			case 4:
				x = cmp.Compare(a.quantity, b.quantity)
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
	a.body.Refresh()
	switch x := a.body.(type) {
	case *widget.Table:
		x.ScrollToTop()
	}
}

func (a *AllAssetSearch) resetSearch() {
	for i := range a.colSort {
		a.colSort[i] = sortOff
	}
	a.entry.SetText("")
	a.processData(-1)
}

func (a *AllAssetSearch) Update() {
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
	a.body.Refresh()
}

func (a *AllAssetSearch) loadData() (bool, error) {
	ctx := context.TODO()
	cc, err := a.u.CharacterService().ListCharactersShort(ctx)
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
	assets, err := a.u.CharacterService().ListAllCharacterAssets(ctx)
	if err != nil {
		return false, err
	}
	locations, err := a.u.EveUniverseService().ListLocations(ctx)
	if err != nil {
		return false, err
	}
	a.assetCollection = assetcollection.New(assets, locations)
	rows := make([]*assetSearchRow, len(assets))
	for i, ca := range assets {
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
			r.quantityDisplay = "1*"
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
		rows[i] = r
	}
	a.assetsFiltered = rows
	a.assets = rows
	a.updateFoundInfo()
	return true, nil
}

func (a *AllAssetSearch) updateFoundInfo() {
	if c := len(a.assetsFiltered); c < len(a.assets) {
		a.found.SetText(fmt.Sprintf("%d found", c))
		a.found.Show()
	} else {
		a.found.Hide()
	}
}

func (a *AllAssetSearch) characterCount() int {
	cc := a.u.StatusCacheService().ListCharacters()
	validCount := 0
	for _, c := range cc {
		if a.u.StatusCacheService().CharacterSectionExists(c.ID, app.SectionAssets) {
			validCount++
		}
	}
	return validCount
}

func (a *AllAssetSearch) makeTopText(c int) (string, widget.Importance) {
	it := humanize.Comma(int64(len(a.assets)))
	text := fmt.Sprintf("%d characters • %s items", c, it)
	return text, widget.MediumImportance
}
