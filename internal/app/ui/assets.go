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

const (
	assetsTotalYes = "Has total"
	assetsTotalNo  = "Has no total"
)

type assetRow struct {
	categoryName    string
	characterID     int32
	characterName   string
	groupID         int32
	groupName       string
	hasTotal        bool
	isSingleton     bool
	itemID          int64
	locationDisplay []widget.RichTextSegment
	locationName    string
	quantity        int
	quantityDisplay string
	regionName      string
	searchTarget    string
	total           float64
	totalDisplay    string
	typeID          int32
	typeName        string
	typeNameDisplay string
}

type assets struct {
	widget.BaseWidget

	body           fyne.CanvasObject
	columnSorter   *columnSorter
	entry          *widget.Entry
	found          *widget.Label
	rows           []assetRow
	rowsFiltered   []assetRow
	selectCategory *iwidget.FilterChipSelect
	selectOwner    *iwidget.FilterChipSelect
	selectRegion   *iwidget.FilterChipSelect
	selectTotal    *iwidget.FilterChipSelect
	sortButton     *sortButton
	total          *widget.Label
	u              *BaseUI
}

func newAssets(u *BaseUI) *assets {
	headers := []headerDef{
		{Text: "Item", Width: 300},
		{Text: "Class", Width: 200},
		{Text: "Location", Width: 350},
		{Text: "Owner", Width: 200},
		{Text: "Qty.", Width: 75},
		{Text: "Total", Width: 100},
	}
	a := &assets{
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
	a.entry.PlaceHolder = "Search items and locations"
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
					return r.locationDisplay
				case 3:
					return iwidget.NewRichTextSegmentFromText(r.characterName)
				case 4:
					return iwidget.NewRichTextSegmentFromText(r.quantityDisplay)
				case 5:
					return iwidget.NewRichTextSegmentFromText(r.totalDisplay)
				}
				return iwidget.NewRichTextSegmentFromText("?")
			},
			a.columnSorter, a.filterRows, func(_ int, r assetRow) {
				a.u.ShowTypeInfoWindow(r.typeID)
			})
	}

	a.selectCategory = iwidget.NewFilterChipSelect("Category", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectOwner = iwidget.NewFilterChipSelect("Owner", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectRegion = iwidget.NewFilterChipSelect("Region", []string{}, func(string) {
		a.filterRows(-1)
	})

	a.selectTotal = iwidget.NewFilterChipSelect("Total",
		[]string{
			assetsTotalYes,
			assetsTotalNo,
		},
		func(s string) {
			a.filterRows(-1)
		},
	)

	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)
	return a
}

func (a *assets) CreateRenderer() fyne.WidgetRenderer {
	filters := container.NewHBox(a.selectCategory, a.selectRegion, a.selectOwner, a.selectTotal)
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

func (a *assets) makeDataList() *widget.List {
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
			iwidget.SetRichText(box[1].(*widget.RichText), r.locationDisplay...)
			box[2].(*widget.Label).SetText(r.characterName)
			box[3].(*widget.Label).SetText(r.totalDisplay)
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

func (a *assets) focus() {
	a.u.MainWindow().Canvas().Focus(a.entry)
}

func (a *assets) filterRows(sortCol int) {
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
				matches = strings.Contains(r.searchTarget, search)
			}
			if matches {
				rows = append(rows, r)
			}
		}
	}
	// other filters
	if x := a.selectCategory.Selected; x != "" {
		rows = xslices.Filter(rows, func(o assetRow) bool {
			return o.categoryName == x
		})
	}
	if x := a.selectOwner.Selected; x != "" {
		rows = xslices.Filter(rows, func(o assetRow) bool {
			return o.characterName == x
		})
	}
	if x := a.selectRegion.Selected; x != "" {
		rows = xslices.Filter(rows, func(o assetRow) bool {
			return o.regionName == x
		})
	}
	if x := a.selectTotal.Selected; x != "" {
		rows = xslices.Filter(rows, func(r assetRow) bool {
			switch x {
			case assetsTotalYes:
				return r.hasTotal
			case assetsTotalNo:
				return !r.hasTotal
			}
			return false
		})
	}
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
				x = strings.Compare(a.locationName, b.locationName)
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
	a.selectCategory.SetOptionsFromSeq(xiter.MapSlice(rows, func(o assetRow) string {
		return o.categoryName
	}))
	a.selectOwner.SetOptionsFromSeq(xiter.MapSlice(rows, func(o assetRow) string {
		return o.characterName
	}))
	a.selectRegion.SetOptionsFromSeq(xiter.MapSlice(rows, func(o assetRow) string {
		return o.regionName
	}))
	a.rowsFiltered = rows
	a.updateFoundInfo()
	a.body.Refresh()
	switch x := a.body.(type) {
	case *widget.Table:
		x.ScrollToTop()
	}
}

func (a *assets) resetSearch() {
	a.entry.SetText("")
	a.filterRows(-1)
}

func (a *assets) update() {
	var t string
	var i widget.Importance
	characterCount := a.characterCount()
	assets, hasData, err := a.fetchRows(a.u.services())
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
		t = fmt.Sprintf("%d characters • %s items", characterCount, ihumanize.Comma(len(assets)))
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

func (*assets) fetchRows(s services) ([]assetRow, bool, error) {
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
			categoryName:    ca.Type.Group.Category.Name,
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
		r.searchTarget = r.typeNameDisplay
		if ca.IsSingleton {
			r.quantityDisplay = "1*"
			r.quantity = 1
		} else {
			r.quantityDisplay = humanize.Comma(int64(ca.Quantity))
			r.quantity = int(ca.Quantity)
		}
		location, ok := assetCollection.AssetParentLocation(ca.ItemID)
		if ok {
			r.locationName = location.DisplayName()
			r.locationDisplay = location.DisplayRichText()
			r.searchTarget += " " + r.locationName
			if location.SolarSystem != nil {
				r.regionName = location.SolarSystem.Constellation.Region.Name
			}
		} else {
			r.locationDisplay = iwidget.NewRichTextSegmentFromText("?")
		}
		if ca.Price.IsEmpty() || ca.IsBlueprintCopy {
			r.totalDisplay = "?"
		} else {
			t := ca.Price.ValueOrZero() * float64(ca.Quantity)
			r.totalDisplay = ihumanize.Number(t, 1)
			r.hasTotal = true
		}
		r.searchTarget = strings.ToLower(r.searchTarget)
		rows[i] = r
	}
	return rows, true, nil
}

func (a *assets) updateFoundInfo() {
	if c := len(a.rowsFiltered); c < len(a.rows) {
		a.found.SetText(fmt.Sprintf("%s found", ihumanize.Comma(c)))
		a.found.Show()
	} else {
		a.found.Hide()
	}
}

func (a *assets) characterCount() int {
	cc := a.u.scs.ListCharacters()
	validCount := 0
	for _, c := range cc {
		if a.u.scs.HasCharacterSection(c.ID, app.SectionAssets) {
			validCount++
		}
	}
	return validCount
}
