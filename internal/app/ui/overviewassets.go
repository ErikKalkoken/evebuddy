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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// TODO: Mobile: Add column sort

type sortDir uint

const (
	sortOff sortDir = iota
	sortAsc
	sortDesc
)

type assetSearchRow struct {
	characterID     int32
	characterName   string
	groupID         int32
	groupName       string
	itemID          int64
	location        *app.EveLocation
	name            string
	quantity        int
	quantityDisplay string
	price           float64
	priceDisplay    string
	typeID          int32
	typeName        string
}

type OverviewAssets struct {
	widget.BaseWidget

	assetCollection assetcollection.AssetCollection
	assets          []*assetSearchRow
	assetsFiltered  []*assetSearchRow
	body            fyne.CanvasObject
	characterNames  map[int32]string
	colSort         []sortDir
	found           *widget.Label
	entry           *widget.Entry
	total           *widget.Label
	u               *BaseUI
}

func NewOverviewAssets(u *BaseUI) *OverviewAssets {
	headers := []iwidget.HeaderDef{
		{Text: "Item", Width: 300},
		{Text: "Class", Width: 200},
		{Text: "Location", Width: 350},
		{Text: "Owner", Width: 200},
		{Text: "Qty.", Width: 75},
		{Text: "Price", Width: 100},
	}
	a := &OverviewAssets{
		colSort:        make([]sortDir, len(headers)),
		assetsFiltered: make([]*assetSearchRow, 0),
		entry:          widget.NewEntry(),
		found:          widget.NewLabel(""),
		total:          appwidget.MakeTopLabel(),
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
	a.found.Hide()

	makeCell := func(col int, r *assetSearchRow) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(r.name)
		case 1:
			return iwidget.NewRichTextSegmentFromText(r.groupName)
		case 2:
			if r.location != nil {
				return r.location.DisplayRichText()
			}
		case 3:
			return iwidget.NewRichTextSegmentFromText(r.characterName)
		case 4:
			return iwidget.NewRichTextSegmentFromText(r.quantityDisplay)
		case 5:
			return iwidget.NewRichTextSegmentFromText(r.priceDisplay)
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	if a.u.IsMobile() {
		a.body = iwidget.MakeDataTableForMobile(headers, &a.assetsFiltered, makeCell, func(r *assetSearchRow) {
			a.u.ShowTypeInfoWindow(r.typeID)
		})
	} else {
		t := iwidget.MakeDataTableForDesktop(headers, &a.assetsFiltered, makeCell, func(col int, r *assetSearchRow) {
			switch col {
			case 0:
				a.u.ShowInfoWindow(app.EveEntityInventoryType, r.typeID)
			case 2:
				if r.location != nil {
					a.u.ShowLocationInfoWindow(r.location.ID)
				}
			case 3:
				a.u.ShowInfoWindow(app.EveEntityCharacter, r.characterID)
			}
		})
		iconSortAsc := theme.NewPrimaryThemedResource(icons.SortAscendingSvg)
		iconSortDesc := theme.NewPrimaryThemedResource(icons.SortDescendingSvg)
		iconSortOff := theme.NewThemedResource(icons.SortSvg)
		t.CreateHeader = func() fyne.CanvasObject {
			icon := widget.NewIcon(iconSortOff)
			label := kxwidget.NewTappableLabel("XXX", nil)
			return container.NewBorder(nil, nil, nil, icon, label)
		}
		t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
			h := headers[tci.Col]
			row := co.(*fyne.Container).Objects
			label := row[0].(*kxwidget.TappableLabel)
			label.OnTapped = func() {
				a.processData(tci.Col)
			}
			label.SetText(h.Text)
			icon := row[1].(*widget.Icon)
			switch a.colSort[tci.Col] {
			case sortOff:
				icon.SetResource(iconSortOff)
			case sortAsc:
				icon.SetResource(iconSortAsc)
			case sortDesc:
				icon.SetResource(iconSortDesc)
			}
		}
		a.body = t
	}
	return a
}

func (a *OverviewAssets) CreateRenderer() fyne.WidgetRenderer {
	topBox := container.NewVBox(
		container.NewBorder(nil, nil, nil, a.found, a.total),
		a.entry,
	)
	c := container.NewBorder(topBox, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *OverviewAssets) Focus() {
	a.u.MainWindow().Canvas().Focus(a.entry)
}

func (a *OverviewAssets) processData(sortCol int) {
	var order sortDir
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
	search := strings.ToLower(a.entry.Text)
	for _, r := range a.assets {
		var matches bool
		if search == "" {
			matches = true
		}
		for _, cell := range []string{r.typeName, r.groupName, r.location.DisplayName(), r.characterName} {
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
				x = cmp.Compare(a.location.DisplayName(), b.location.DisplayName())
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

func (a *OverviewAssets) resetSearch() {
	for i := range a.colSort {
		a.colSort[i] = sortOff
	}
	a.entry.SetText("")
	a.processData(-1)
}

func (a *OverviewAssets) update() {
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
	fyne.Do(func() {
		a.updateFoundInfo()
		a.total.Text = t
		a.total.Importance = i
		a.total.Refresh()
	})
	fyne.Do(func() {
		a.body.Refresh()
	})
}

func (a *OverviewAssets) loadData() (bool, error) {
	ctx := context.Background()
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
	assets, err := a.u.CharacterService().ListAllAssets(ctx)
	if err != nil {
		return false, err
	}
	locations, err := a.u.eus.ListLocations(ctx)
	if err != nil {
		return false, err
	}
	a.assetCollection = assetcollection.New(assets, locations)
	rows := make([]*assetSearchRow, len(assets))
	for i, ca := range assets {
		r := &assetSearchRow{
			characterID:   ca.CharacterID,
			characterName: a.characterNames[ca.CharacterID],
			groupID:       ca.Type.Group.ID,
			groupName:     ca.Type.Group.Name,
			itemID:        ca.ItemID,
			name:          ca.DisplayName2(),
			price:         ca.Price.ValueOrZero(),
			typeID:        ca.Type.ID,
			typeName:      ca.Type.Name,
		}
		if ca.IsSingleton {
			r.quantityDisplay = "1*"
			r.quantity = 1
		} else {
			r.quantityDisplay = humanize.Comma(int64(ca.Quantity))
			r.quantity = int(ca.Quantity)
		}
		location, ok := a.assetCollection.AssetParentLocation(ca.ItemID)
		if ok {
			r.location = location
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
	return true, nil
}

func (a *OverviewAssets) updateFoundInfo() {
	if c := len(a.assetsFiltered); c < len(a.assets) {
		a.found.SetText(fmt.Sprintf("%d found", c))
		a.found.Show()
	} else {
		a.found.Hide()
	}
}

func (a *OverviewAssets) characterCount() int {
	cc := a.u.StatusCacheService().ListCharacters()
	validCount := 0
	for _, c := range cc {
		if a.u.StatusCacheService().CharacterSectionExists(c.ID, app.SectionAssets) {
			validCount++
		}
	}
	return validCount
}

func (a *OverviewAssets) makeTopText(c int) (string, widget.Importance) {
	it := humanize.Comma(int64(len(a.assets)))
	text := fmt.Sprintf("%d characters • %s items", c, it)
	return text, widget.MediumImportance
}
