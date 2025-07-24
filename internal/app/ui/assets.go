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
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
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
	isSingleton     bool
	itemID          int64
	location        *app.EveLocationShort
	locationDisplay []widget.RichTextSegment
	locationName    string
	price           optional.Optional[float64]
	priceDisplay    string
	quantity        int
	quantityDisplay string
	regionName      string
	searchTarget    string
	tags            set.Set[string]
	total           optional.Optional[float64]
	totalDisplay    string
	typeID          int32
	typeName        string
	typeNameDisplay string
}

func newAssetRow(ca *app.CharacterAsset, assetCollection assetcollection.AssetCollection, characterName func(int32) string) assetRow {
	r := assetRow{
		categoryName:    ca.Type.Group.Category.Name,
		characterID:     ca.CharacterID,
		characterName:   characterName(ca.CharacterID),
		groupID:         ca.Type.Group.ID,
		groupName:       ca.Type.Group.Name,
		isSingleton:     ca.IsSingleton,
		itemID:          ca.ItemID,
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
		r.location = location.ToShort()
		r.locationName = location.DisplayName()
		r.locationDisplay = location.DisplayRichText()
		if location.SolarSystem != nil {
			r.regionName = location.SolarSystem.Constellation.Region.Name
		}
	} else {
		r.locationDisplay = iwidget.RichTextSegmentsFromText("?")
	}
	if !ca.IsBlueprintCopy {
		r.price = ca.Price
	}
	r.priceDisplay = r.price.StringFunc("?", func(v float64) string {
		return ihumanize.Number(v, 1)
	})
	if !r.price.IsEmpty() {
		r.total.Set(ca.Price.ValueOrZero() * float64(ca.Quantity))
	}
	r.totalDisplay = r.total.StringFunc("?", func(v float64) string {
		return ihumanize.Number(v, 1)
	})
	return r
}

type assets struct {
	widget.BaseWidget

	onUpdate func(total string)

	body           fyne.CanvasObject
	columnSorter   *columnSorter
	entry          *widget.Entry
	found          *widget.Label
	rows           []assetRow
	rowsFiltered   []assetRow
	selectCategory *kxwidget.FilterChipSelect
	selectLocation *kxwidget.FilterChipSelect
	selectOwner    *kxwidget.FilterChipSelect
	selectRegion   *kxwidget.FilterChipSelect
	selectTag      *kxwidget.FilterChipSelect
	selectTotal    *kxwidget.FilterChipSelect
	sortButton     *sortButton
	total          *widget.Label
	u              *baseUI
}

func newAssets(u *baseUI) *assets {
	headers := []headerDef{
		{label: "Item", width: 300},
		{label: "Class", width: 200},
		{label: "Location", width: columnWidthLocation},
		{label: "Owner", width: columnWidthEntity},
		{label: "Qty.", width: 75},
		{label: "Total", width: 100},
	}
	a := &assets{
		columnSorter: newColumnSorter(headers),
		entry:        widget.NewEntry(),
		found:        widget.NewLabel(""),
		rowsFiltered: make([]assetRow, 0),
		total:        makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.entry.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
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
					return iwidget.RichTextSegmentsFromText(r.typeNameDisplay)
				case 1:
					return iwidget.RichTextSegmentsFromText(r.groupName)
				case 2:
					return r.locationDisplay
				case 3:
					return iwidget.RichTextSegmentsFromText(r.characterName)
				case 4:
					return iwidget.RichTextSegmentsFromText(r.quantityDisplay)
				case 5:
					return iwidget.RichTextSegmentsFromText(r.totalDisplay)
				}
				return iwidget.RichTextSegmentsFromText("?")
			},
			a.columnSorter, a.filterRows, func(_ int, r assetRow) {
				showAssetDetailWindow(u, r)
			})
	}

	a.selectCategory = kxwidget.NewFilterChipSelectWithSearch("Category", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectOwner = kxwidget.NewFilterChipSelect("Owner", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.selectRegion = kxwidget.NewFilterChipSelectWithSearch("Region", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)
	a.selectLocation = kxwidget.NewFilterChipSelectWithSearch("Location", []string{}, func(string) {
		a.filterRows(-1)
	}, a.u.window)

	a.selectTotal = kxwidget.NewFilterChipSelect("Total",
		[]string{
			assetsTotalYes,
			assetsTotalNo,
		},
		func(s string) {
			a.filterRows(-1)
		},
	)
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRows(-1)
	})
	a.sortButton = a.columnSorter.newSortButton(headers, func() {
		a.filterRows(-1)
	}, a.u.window)
	return a
}

func (a *assets) CreateRenderer() fyne.WidgetRenderer {
	filters := container.NewHBox(
		a.selectCategory,
		a.selectRegion,
		a.selectLocation,
		a.selectOwner,
		a.selectTotal,
		a.selectTag,
	)
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

func (a *assets) makeDataList() *iwidget.StripedList {
	p := theme.Padding()
	l := iwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Template", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			owner := widget.NewLabel("Template")
			location := iwidget.NewRichTextWithText("Template")
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
			box[1].(*iwidget.RichText).Set(r.locationDisplay)
			box[2].(*widget.Label).SetText(r.characterName)
			box[3].(*widget.Label).SetText(r.totalDisplay)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		showAssetDetailWindow(a.u, r)
	}
	return l
}

func (a *assets) focus() {
	a.u.MainWindow().Canvas().Focus(a.entry)
}

func (a *assets) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
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
	if x := a.selectLocation.Selected; x != "" {
		rows = xslices.Filter(rows, func(o assetRow) bool {
			return o.locationName == x
		})
	}
	if x := a.selectTotal.Selected; x != "" {
		rows = xslices.Filter(rows, func(r assetRow) bool {
			switch x {
			case assetsTotalYes:
				return !r.total.IsEmpty()
			case assetsTotalNo:
				return r.total.IsEmpty()
			}
			return false
		})
	}
	if x := a.selectTag.Selected; x != "" {
		rows = xslices.Filter(rows, func(r assetRow) bool {
			return r.tags.Contains(x)
		})
	}
	// search filter
	if search := strings.ToLower(a.entry.Text); search != "" {
		rows2 := make([]assetRow, 0)
		for _, r := range rows {
			var matches bool
			if search == "" {
				matches = true
			} else {
				matches = strings.Contains(r.searchTarget, search)
			}
			if matches {
				rows2 = append(rows2, r)
			}
		}
		rows = rows2
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
				x = cmp.Compare(a.total.ValueOrZero(), b.total.ValueOrZero())
			}
			if dir == sortAsc {
				return x
			} else {
				return -1 * x
			}
		})
	})
	// set data & refresh
	a.selectTag.SetOptions(slices.Sorted(set.Union(xslices.Map(rows, func(r assetRow) set.Set[string] {
		return r.tags
	})...).All()))
	a.selectCategory.SetOptions(xslices.Map(rows, func(o assetRow) string {
		return o.categoryName
	}))
	a.selectOwner.SetOptions(xslices.Map(rows, func(o assetRow) string {
		return o.characterName
	}))
	a.selectRegion.SetOptions(xslices.Map(rows, func(o assetRow) string {
		return o.regionName
	}))
	a.selectLocation.SetOptions(xslices.Map(rows, func(o assetRow) string {
		return o.locationName
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
	if a.onUpdate != nil {
		var s string
		if hasData && err == nil {
			var total float64
			for _, a := range assets {
				total += a.total.ValueOrZero()
			}
			s = ihumanize.Number(total, 1)
		}
		a.onUpdate(s)
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
	tagsPerCharacter := make(map[int32]set.Set[string])
	for _, c := range cc {
		tags, err := s.cs.ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			return nil, false, nil
		}
		tagsPerCharacter[c.ID] = set.Collect(xiter.MapSlice(tags, func(x *app.CharacterTag) string {
			return x.Name
		}))
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
		r := newAssetRow(ca, assetCollection, func(id int32) string {
			return characterNames[id]
		})
		r.searchTarget = strings.ToLower(r.typeNameDisplay)
		r.tags = tagsPerCharacter[ca.CharacterID]
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
		if a.u.scs.HasCharacterSection(c.ID, app.SectionCharacterAssets) {
			validCount++
		}
	}
	return validCount
}

// showAssetDetailWindow shows the details for a character assets in a new window.
func showAssetDetailWindow(u *baseUI, r assetRow) {
	w, ok := u.getOrCreateWindow(fmt.Sprintf("%d-%d", r.characterID, r.itemID), "Asset: Information", r.characterName)
	if !ok {
		w.Show()
		return
	}
	item := makeLinkLabelWithWrap(r.typeNameDisplay, func() {
		u.ShowTypeInfoWindowWithCharacter(r.typeID, r.characterID)
	})
	var location fyne.CanvasObject
	if r.location != nil {
		location = makeLocationLabel(r.location, u.ShowLocationInfoWindow)
	} else {
		location = widget.NewLabel("?")
	}
	fi := []*widget.FormItem{
		widget.NewFormItem("Owner", makeOwnerActionLabel(
			r.characterID,
			r.characterName,
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Item", item),
		widget.NewFormItem("Class", widget.NewLabel(r.groupName)),
		widget.NewFormItem("Location", location),
		widget.NewFormItem(
			"Price",
			widget.NewLabel(r.price.StringFunc("?", func(v float64) string {
				return formatISKAmount(v)
			})),
		),
		widget.NewFormItem("Quantity", widget.NewLabel(r.quantityDisplay)),
		widget.NewFormItem(
			"Total",
			widget.NewLabel(r.total.StringFunc("?", func(v float64) string {
				return formatISKAmount(v)
			})),
		),
	}

	f := widget.NewForm(fi...)
	f.Orientation = widget.Adaptive
	subTitle := fmt.Sprintf("Asset #%d", r.itemID)
	setDetailWindowWithSize(subTitle, fyne.NewSize(500, 450), f, w)
	w.Show()
}
