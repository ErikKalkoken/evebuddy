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
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

const (
	selectOwnerAny    = "Any owner"
	selectLocationAny = "Any location"
)

type assetRow struct {
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

	body           fyne.CanvasObject
	columnSorter   *columnSorter
	entry          *widget.Entry
	found          *widget.Label
	rows           []assetRow
	rowsFiltered   []assetRow
	selectLocation *widget.Select
	selectOwner    *widget.Select
	sortButton     *widget.Button
	total          *widget.Label
	u              *BaseUI
}

func NewOverviewAssets(u *BaseUI) *OverviewAssets {
	headers := []headerDef{
		{Text: "Item", Width: 300},
		{Text: "Class", Width: 200},
		{Text: "Location", Width: 350},
		{Text: "Owner", Width: 200},
		{Text: "Qty.", Width: 75},
		{Text: "Price", Width: 100},
	}
	a := &OverviewAssets{
		entry:        widget.NewEntry(),
		found:        widget.NewLabel(""),
		rowsFiltered: make([]assetRow, 0),
		columnSorter: newColumnSorter(len(headers)),
		total:        makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.entry.ActionItem = iwidget.NewIconButton(theme.CancelIcon(), func() {
		a.resetSearch()
	})
	a.entry.OnChanged = func(s string) {
		a.filterRows(-1)
	}
	a.entry.PlaceHolder = "Search assets"
	a.found.Hide()

	makeCell := func(col int, r assetRow) []widget.RichTextSegment {
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
		a.body = makeDataList(headers, &a.rowsFiltered, makeCell, func(r assetRow) {
			a.u.ShowTypeInfoWindow(r.typeID)
		})
	} else {
		a.body = makeDataTableWithSort(headers, &a.rowsFiltered, makeCell, a.columnSorter, a.filterRows, func(_ int, r assetRow) {
			a.u.ShowTypeInfoWindow(r.typeID)
		})
	}

	a.selectOwner = widget.NewSelect([]string{selectOwnerAny}, func(s string) {
		a.filterRows(-1)
	})
	a.selectOwner.Selected = selectOwnerAny

	a.selectLocation = widget.NewSelect([]string{selectLocationAny}, func(s string) {
		a.filterRows(-1)
	})
	a.selectLocation.Selected = selectLocationAny

	a.sortButton = makeSortButton(headers, set.Of(0, 1, 2, 3, 4, 5), a.columnSorter, func() {
		a.filterRows(-1)
	}, a.u.window)
	return a
}

func (a *OverviewAssets) CreateRenderer() fyne.WidgetRenderer {
	config := container.NewHBox(a.selectLocation, a.selectOwner)
	topBox := container.NewVBox(
		container.NewBorder(nil, nil, nil, a.found, a.total),
		a.entry,
		config,
	)
	if !a.u.isDesktop {
		config.Add(container.NewHBox(a.sortButton))
	}
	c := container.NewBorder(topBox, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *OverviewAssets) Focus() {
	a.u.MainWindow().Canvas().Focus(a.entry)
}

func (a *OverviewAssets) filterRows(sortCol int) {
	// search filter
	rows := make([]assetRow, 0)
	if search := strings.ToLower(a.entry.Text); search == "" {
		rows = slices.Clone(a.rows)
	} else {
		for _, r := range a.rows {
			var matches bool
			if search == "" {
				matches = true
			}
			for _, cell := range []string{r.typeName, r.groupName} {
				if search != "" {
					matches = matches || strings.Contains(strings.ToLower(cell), search)
				}
			}
			if matches {
				rows = append(rows, r)
			}
		}
	}
	// other filter
	if x := a.selectLocation.Selected; x != selectLocationAny {
		rows = xslices.Filter(rows, func(o assetRow) bool {
			return o.location.DisplayName() == x
		})
	}
	if x := a.selectOwner.Selected; x != selectOwnerAny {
		rows = xslices.Filter(rows, func(o assetRow) bool {
			return o.characterName == x
		})
	}
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b assetRow) int {
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
	})
	locations := slices.Concat(
		[]string{selectLocationAny},
		slices.Sorted(set.Collect(xiter.MapSlice(rows, func(o assetRow) string {
			return o.location.DisplayName()
		})).All()),
	)
	owners := slices.Concat(
		[]string{selectOwnerAny},
		slices.Sorted(set.Collect(xiter.MapSlice(rows, func(o assetRow) string {
			return o.characterName
		})).All()),
	)
	a.selectLocation.SetOptions(locations)
	a.selectOwner.SetOptions(owners)
	a.rowsFiltered = rows
	a.updateFoundInfo()
	a.body.Refresh()
	switch x := a.body.(type) {
	case *widget.Table:
		x.ScrollToTop()
	}
}

func (a *OverviewAssets) resetSearch() {
	a.entry.SetText("")
	a.filterRows(-1)
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
		a.rowsFiltered = assets
		a.rows = assets
		a.body.Refresh()
		a.filterRows(-1)
	})
}

func (*OverviewAssets) loadData(s services) ([]assetRow, bool, error) {
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
	rows := make([]assetRow, len(assets))
	for i, ca := range assets {
		r := assetRow{
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
	if c := len(a.rowsFiltered); c < len(a.rows) {
		a.found.SetText(fmt.Sprintf("%s found", ihumanize.Comma(c)))
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
	it := humanize.Comma(int64(len(a.rows)))
	text := fmt.Sprintf("%d characters • %s items", c, it)
	return text, widget.MediumImportance
}
