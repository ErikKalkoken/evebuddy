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
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type assetSearchRow struct {
	characterID     int32
	characterName   string
	groupID         int32
	groupName       string
	itemID          int64
	location        *app.EveLocation
	name            string
	price           float64
	priceDisplay    string
	quantity        int
	quantityDisplay string
	typeID          int32
	typeName        string
}

type OverviewAssets struct {
	widget.BaseWidget

	assets         []*assetSearchRow
	assetsFiltered []*assetSearchRow
	body           fyne.CanvasObject
	entry          *widget.Entry
	found          *widget.Label
	sortButton     *widget.Button
	sortedColumns  *sortedColumns
	total          *widget.Label
	u              *BaseUI
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
		sortedColumns:  newSortedColumns(len(headers)),
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

	if !a.u.isDesktop {
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
			switch a.sortedColumns.column(tci.Col) {
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
	a.sortButton = makeSortButton(headers, set.Of(0, 1, 2, 3, 4, 5), a.sortedColumns, func() {
		a.processData(-1)
	}, a.u.window)
	return a
}

func (a *OverviewAssets) CreateRenderer() fyne.WidgetRenderer {
	topBox := container.NewVBox(
		container.NewBorder(nil, nil, nil, a.found, a.total),
		a.entry,
	)
	if !a.u.isDesktop {
		topBox.Add(container.NewHBox(a.sortButton))
	}
	c := container.NewBorder(topBox, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *OverviewAssets) Focus() {
	a.u.MainWindow().Canvas().Focus(a.entry)
}

func (a *OverviewAssets) processData(sortCol int) {
	var dir sortDir
	if sortCol >= 0 {
		dir = a.sortedColumns.cycleColumn(sortCol)
	} else {
		sortCol, dir = a.sortedColumns.current()
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
	if sortCol >= 0 && dir != sortOff {
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
			if dir == sortAsc {
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
	a.entry.SetText("")
	a.processData(-1)
}

func (a *OverviewAssets) update() {
	var t string
	var i widget.Importance
	characterCount := a.characterCount()
	assets, hasData, err := a.loadData(a.u.services())
	if err != nil {
		slog.Error("Failed to refresh asset search data", "err", err)
		t = "ERROR: " + a.u.humanizeError(err)
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
		a.assetsFiltered = assets
		a.assets = assets
		a.body.Refresh()
	})
}

func (*OverviewAssets) loadData(s services) ([]*assetSearchRow, bool, error) {
	ctx := context.Background()
	cc, err := s.cs.ListCharactersShort(ctx)
	if err != nil {
		return nil, false, err
	}
	if len(cc) == 0 {
		return nil, false, nil
	}
	characterNames := make(map[int32]string)
	for _, o := range cc {
		characterNames[o.ID] = o.Name
	}
	assets, err := s.cs.ListAllAssets(ctx)
	if err != nil {
		return nil, false, err
	}
	locations, err := s.eus.ListLocations(ctx)
	if err != nil {
		return nil, false, err
	}
	assetCollection := assetcollection.New(assets, locations)
	rows := make([]*assetSearchRow, len(assets))
	for i, ca := range assets {
		r := &assetSearchRow{
			characterID:   ca.CharacterID,
			characterName: characterNames[ca.CharacterID],
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
		location, ok := assetCollection.AssetParentLocation(ca.ItemID)
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
	return rows, true, nil
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
	cc := a.u.scs.ListCharacters()
	validCount := 0
	for _, c := range cc {
		if a.u.scs.HasCharacterSection(c.ID, app.SectionAssets) {
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
