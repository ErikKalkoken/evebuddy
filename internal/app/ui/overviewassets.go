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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type assetRow struct {
	characterID     int32
	characterName   string
	groupID         int32
	groupName       string
	isSingleton     bool
	itemID          int64
	location        *app.EveLocation
	priceDisplay    string
	quantity        int
	quantityDisplay string
	total           float64
	typeID          int32
	typeName        string
	typeNameDisplay string
}

func (r assetRow) SolarSystemName() string {
	if r.location == nil || r.location.SolarSystem == nil {
		return ""
	}
	return r.location.SolarSystem.Name
}

type overviewAssets struct {
	widget.BaseWidget

	body              fyne.CanvasObject
	columnSorter      *columnSorter
	entry             *widget.Entry
	found             *widget.Label
	rows              []assetRow
	rowsFiltered      []assetRow
	selectSolarSystem *selectFilter
	selectOwner       *selectFilter
	sortButton        *sortButton
	total             *widget.Label
	u                 *BaseUI
}

func newOverviewAssets(u *BaseUI) *overviewAssets {
	headers := []headerDef{
		{Text: "Item", Width: 300},
		{Text: "Class", Width: 200},
		{Text: "Location", Width: 350},
		{Text: "Owner", Width: 200},
		{Text: "Qty.", Width: 75},
		{Text: "Total", Width: 100},
	}
	a := &overviewAssets{
		entry:        widget.NewEntry(),
		found:        widget.NewLabel(""),
		rowsFiltered: make([]assetRow, 0),
		columnSorter: newColumnSorter(headers),
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
	a.entry.PlaceHolder = "Search items"
	a.found.Hide()

	if !a.u.isDesktop {
		a.body = a.makeDataList()
	} else {
		a.body = makeDataTable(headers, &a.rowsFiltered,
			func(col int, r assetRow) []widget.RichTextSegment {
				switch col {
				case 0:
					return iwidget.NewRichTextSegmentFromText(r.typeNameDisplay)
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
			},
			a.columnSorter, a.filterRows, func(_ int, r assetRow) {
				a.u.ShowTypeInfoWindow(r.typeID)
			})
	}

	a.selectOwner = newSelectFilter("Any owner", func() {
		a.filterRows(-1)
	})

	a.selectSolarSystem = newSelectFilter("Any system", func() {
		a.filterRows(-1)
	})

	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)
	return a
}

func (a *overviewAssets) CreateRenderer() fyne.WidgetRenderer {
	filters := container.NewHBox(a.selectSolarSystem, a.selectOwner)
	if !a.u.isDesktop {
		filters.Add(container.NewHBox(a.sortButton))
	}
	topBox := container.NewVBox(
		container.NewBorder(nil, nil, nil, a.found, a.total),
		a.entry,
		container.NewHScroll(filters),
	)
	c := container.NewBorder(topBox, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *overviewAssets) makeDataList() *widget.List {
	p := theme.Padding()
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Template", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			owner := widget.NewLabel("Template")
			location := widget.NewRichTextWithText("Template")
			price := widget.NewLabel("Template")
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				title,
				location,
				owner,
				price,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			box := co.(*fyne.Container).Objects
			var title string
			if r.isSingleton {
				title = r.typeNameDisplay
			} else {
				title = fmt.Sprintf("%s x%s", r.typeNameDisplay, r.quantityDisplay)
			}
			box[0].(*widget.Label).SetText(title)
			iwidget.SetRichText(box[1].(*widget.RichText), r.location.DisplayRichText()...)
			box[2].(*widget.Label).SetText(r.characterName)
			box[3].(*widget.Label).SetText(r.priceDisplay)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		a.u.ShowTypeInfoWindow(a.rowsFiltered[id].typeID)
	}
	return l
}

func (a *overviewAssets) focus() {
	a.u.MainWindow().Canvas().Focus(a.entry)
}

func (a *overviewAssets) filterRows(sortCol int) {
	// search filter
	rows := make([]assetRow, 0)
	if search := strings.ToLower(a.entry.Text); search == "" {
		rows = slices.Clone(a.rows)
	} else {
		for _, r := range a.rows {
			var matches bool
			if search == "" {
				matches = true
			} else {
				matches = strings.Contains(strings.ToLower(r.typeNameDisplay), search)
			}
			if matches {
				rows = append(rows, r)
			}
		}
	}
	// other filter
	a.selectSolarSystem.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(o assetRow) bool {
			return o.SolarSystemName() == selected
		})
	})
	a.selectOwner.applyFilter(func(selected string) {
		rows = xslices.Filter(rows, func(o assetRow) bool {
			return o.characterName == selected
		})
	})
	// sort
	a.columnSorter.sort(sortCol, func(sortCol int, dir sortDir) {
		slices.SortFunc(rows, func(a, b assetRow) int {
			var x int
			switch sortCol {
			case 0:
				x = cmp.Compare(a.typeNameDisplay, b.typeNameDisplay)
			case 1:
				x = cmp.Compare(a.groupName, b.groupName)
			case 2:
				x = cmp.Compare(a.location.DisplayName(), b.location.DisplayName())
			case 3:
				x = cmp.Compare(a.characterName, b.characterName)
			case 4:
				x = cmp.Compare(a.quantity, b.quantity)
			case 5:
				x = cmp.Compare(a.total, b.total)
			}
			if dir == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	a.selectSolarSystem.setOptions(xiter.MapSlice(rows, func(o assetRow) string {
		return o.SolarSystemName()
	}))
	a.selectOwner.setOptions(xiter.MapSlice(rows, func(o assetRow) string {
		return o.characterName
	}))
	a.rowsFiltered = rows
	a.updateFoundInfo()
	a.body.Refresh()
	switch x := a.body.(type) {
	case *widget.Table:
		x.ScrollToTop()
	}
}

func (a *overviewAssets) resetSearch() {
	a.entry.SetText("")
	a.filterRows(-1)
}

func (a *overviewAssets) update() {
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

func (*overviewAssets) loadData(s services) ([]assetRow, bool, error) {
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
			characterID:     ca.CharacterID,
			characterName:   characterNames[ca.CharacterID],
			groupID:         ca.Type.Group.ID,
			groupName:       ca.Type.Group.Name,
			isSingleton:     ca.IsSingleton,
			itemID:          ca.ItemID,
			total:           ca.Price.ValueOrZero(),
			typeID:          ca.Type.ID,
			typeName:        ca.Type.Name,
			typeNameDisplay: ca.DisplayName2(),
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

func (a *overviewAssets) updateFoundInfo() {
	if c := len(a.rowsFiltered); c < len(a.rows) {
		a.found.SetText(fmt.Sprintf("%s found", ihumanize.Comma(c)))
		a.found.Show()
	} else {
		a.found.Hide()
	}
}

func (a *overviewAssets) characterCount() int {
	cc := a.u.scs.ListCharacters()
	validCount := 0
	for _, c := range cc {
		if a.u.scs.HasCharacterSection(c.ID, app.SectionAssets) {
			validCount++
		}
	}
	return validCount
}

func (a *overviewAssets) makeTopText(c int) (string, widget.Importance) {
	it := humanize.Comma(int64(len(a.rows)))
	text := fmt.Sprintf("%d characters • %s items", c, it)
	return text, widget.MediumImportance
}
