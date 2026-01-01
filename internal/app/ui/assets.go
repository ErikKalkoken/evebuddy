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
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
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
	regionID        int32
	regionName      string
	searchTarget    string
	tags            set.Set[string]
	total           optional.Optional[float64]
	totalDisplay    string
	typeID          int32
	typeName        string
	typeNameDisplay string
	variant         app.InventoryTypeVariant
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
		variant:         ca.Variant(),
	}
	if ca.IsSingleton {
		r.quantityDisplay = "1*"
		r.quantity = 1
	} else {
		r.quantityDisplay = humanize.Comma(int64(ca.Quantity))
		r.quantity = int(ca.Quantity)
	}
	ln, ok := assetCollection.AssetLocation(ca.ItemID)
	if ok {
		r.location = ln.Location.ToShort()
		r.locationName = ln.Location.DisplayName()
		r.locationDisplay = ln.Location.DisplayRichText()
		if ln.Location.SolarSystem != nil {
			r.regionName = ln.Location.SolarSystem.Constellation.Region.Name
			r.regionID = ln.Location.SolarSystem.Constellation.Region.ID
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
		return humanize.FormatFloat(app.FloatFormat, v)
	})
	return r
}

type assets struct {
	widget.BaseWidget

	onUpdate func(total string)

	body           fyne.CanvasObject
	columnSorter   *iwidget.ColumnSorter
	found          *widget.Label
	rows           []assetRow
	rowsFiltered   []assetRow
	search         *widget.Entry
	selectCategory *kxwidget.FilterChipSelect
	selectGroup    *kxwidget.FilterChipSelect
	selectLocation *kxwidget.FilterChipSelect
	selectOwner    *kxwidget.FilterChipSelect
	selectRegion   *kxwidget.FilterChipSelect
	selectTag      *kxwidget.FilterChipSelect
	selectTotal    *kxwidget.FilterChipSelect
	sortButton     *iwidget.SortButton
	total          *widget.Label
	u              *baseUI
}

const (
	assetsColItem     = 0
	assetsColGroup    = 1
	assetsColLocation = 2
	assetsColOwner    = 3
	assetsColQuantity = 4
	assetsColTotal    = 5
)

func newAssets(u *baseUI) *assets {
	headers := iwidget.NewDataTableDef([]iwidget.ColumnDef{{
		Col:   assetsColItem,
		Label: "Item",
		Width: 300,
	}, {
		Col:   assetsColGroup,
		Label: "Group",
		Width: 200,
	}, {
		Col:   assetsColLocation,
		Label: "Location",
		Width: columnWidthLocation,
	}, {
		Col:   assetsColOwner,
		Label: "Owner",
		Width: columnWidthEntity,
	}, {
		Col:   assetsColQuantity,
		Label: "Qty.",
		Width: 100,
	}, {
		Col:   assetsColTotal,
		Label: "Total",
		Width: 150,
	}})
	a := &assets{
		columnSorter: headers.NewColumnSorter(assetsColItem, iwidget.SortAsc),
		found:        widget.NewLabel(""),
		rowsFiltered: make([]assetRow, 0),
		search:       widget.NewEntry(),
		total:        makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRows(-1)
	})
	a.search.OnChanged = func(s string) {
		a.filterRows(-1)
	}
	a.search.PlaceHolder = "Search items"
	a.found.Hide()

	if !a.u.isDesktop {
		a.body = a.makeDataList()
	} else {
		a.body = iwidget.MakeDataTable(headers, &a.rowsFiltered,
			func(col int, r assetRow) []widget.RichTextSegment {
				switch col {
				case assetsColItem:
					return iwidget.RichTextSegmentsFromText(r.typeNameDisplay)
				case assetsColGroup:
					return iwidget.RichTextSegmentsFromText(r.groupName)
				case assetsColLocation:
					return r.locationDisplay
				case assetsColOwner:
					return iwidget.RichTextSegmentsFromText(r.characterName)
				case assetsColQuantity:
					return iwidget.RichTextSegmentsFromText(r.quantityDisplay, widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
					})
				case assetsColTotal:
					return iwidget.RichTextSegmentsFromText(r.totalDisplay, widget.RichTextStyle{
						Alignment: fyne.TextAlignTrailing,
					})
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
	a.selectGroup = kxwidget.NewFilterChipSelectWithSearch("Group", []string{}, func(string) {
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
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRows(-1)
	}, a.u.window)

	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		if arg.section == app.SectionCharacterAssets {
			a.update()
		}
	})
	a.u.generalSectionChanged.AddListener(func(_ context.Context, arg generalSectionUpdated) {
		if arg.section == app.SectionEveMarketPrices {
			a.update()
		}
	})
	return a
}

func (a *assets) CreateRenderer() fyne.WidgetRenderer {
	filters := container.NewHBox(
		a.selectCategory,
		a.selectGroup,
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
		a.search,
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
	a.u.MainWindow().Canvas().Focus(a.search)
}

func (a *assets) filterRows(sortCol int) {
	rows := slices.Clone(a.rows)
	// other filters
	if x := a.selectCategory.Selected; x != "" {
		rows = xslices.Filter(rows, func(o assetRow) bool {
			return o.categoryName == x
		})
	}
	if x := a.selectGroup.Selected; x != "" {
		rows = xslices.Filter(rows, func(o assetRow) bool {
			return o.groupName == x
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
	if search := strings.ToLower(a.search.Text); search != "" {
		rows = slices.DeleteFunc(rows, func(r assetRow) bool {
			return !strings.Contains(r.searchTarget, search)
		})
	}
	// sort
	a.columnSorter.Sort(sortCol, func(sortCol int, dir iwidget.SortDir) {
		slices.SortFunc(rows, func(a, b assetRow) int {
			var x int
			switch sortCol {
			case assetsColItem:
				x = strings.Compare(a.typeNameDisplay, b.typeNameDisplay)
			case assetsColGroup:
				x = strings.Compare(a.groupName, b.groupName)
			case assetsColLocation:
				x = strings.Compare(a.locationName, b.locationName)
			case assetsColOwner:
				x = xstrings.CompareIgnoreCase(a.characterName, b.characterName)
			case assetsColQuantity:
				x = cmp.Compare(a.quantity, b.quantity)
			case assetsColTotal:
				x = cmp.Compare(a.total.ValueOrZero(), b.total.ValueOrZero())
			}
			if dir == iwidget.SortAsc {
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
	a.selectGroup.SetOptions(xslices.Map(rows, func(o assetRow) string {
		return o.groupName
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

func (a *assets) update() {
	var t string
	var i widget.Importance
	characterCount := a.characterCount()
	assets, quantity, total, err := a.fetchRows(a.u.services())
	if err != nil {
		slog.Error("Failed to refresh asset search data", "err", err)
		t = "ERROR: " + a.u.humanizeError(err)
		i = widget.DangerImportance
	} else if characterCount == 0 {
		t = "No characters"
		i = widget.LowImportance
	} else {
		t = fmt.Sprintf("%d characters • %s items", characterCount, ihumanize.Comma(quantity))
	}
	if a.onUpdate != nil {
		var s string
		if err == nil {
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

func (*assets) fetchRows(s services) ([]assetRow, int, float64, error) {
	ctx := context.Background()
	cc, err := s.cs.ListCharactersShort(ctx)
	if err != nil {
		return nil, 0, 0, err
	}
	if len(cc) == 0 {
		return nil, 0, 0, nil
	}
	characterNames := make(map[int32]string)
	for _, o := range cc {
		characterNames[o.ID] = o.Name
	}
	tagsPerCharacter := make(map[int32]set.Set[string])
	for _, c := range cc {
		tags, err := s.cs.ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			return nil, 0, 0, nil
		}
		tagsPerCharacter[c.ID] = tags
	}
	assets, err := s.cs.ListAllAssets(ctx)
	if err != nil {
		return nil, 0, 0, err
	}
	locations, err := s.eus.ListLocations(ctx)
	if err != nil {
		return nil, 0, 0, err
	}
	assetCollection := assetcollection.New(assets, locations)
	rows := make([]assetRow, len(assets))
	var totalQuantity int
	var totalPrice float64
	for i, ca := range assets {
		r := newAssetRow(ca, assetCollection, func(id int32) string {
			return characterNames[id]
		})
		r.searchTarget = strings.ToLower(r.typeNameDisplay)
		r.tags = tagsPerCharacter[ca.CharacterID]
		rows[i] = r
		totalQuantity += r.quantity
		totalPrice += r.total.ValueOrZero()
	}
	return rows, totalQuantity, totalPrice, nil
}

func (a *assets) updateFoundInfo() {
	if len(a.rowsFiltered) < len(a.rows) {
		var quantity int
		for _, r := range a.rowsFiltered {
			quantity += r.quantity
		}
		s := fmt.Sprintf("%s found", ihumanize.Comma(quantity))
		a.found.SetText(s)
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
	w, created := u.getOrCreateWindow(
		fmt.Sprintf("asset-%d-%d", r.characterID, r.itemID),
		"Asset: Information",
		r.characterName,
	)
	if !created {
		w.Show()
		return
	}
	item := makeLinkLabelWithWrap(r.typeNameDisplay, func() {
		u.ShowTypeInfoWindowWithCharacter(r.typeID, r.characterID)
	})
	var location, region fyne.CanvasObject
	if r.location != nil {
		location = makeLocationLabel(r.location, u.ShowLocationInfoWindow)
		region = makeLinkLabel(r.regionName, func() {
			u.ShowInfoWindow(app.EveEntityRegion, r.regionID)
		})
	} else {
		location = widget.NewLabel("?")
		region = widget.NewLabel("?")
	}
	items := []*widget.FormItem{
		widget.NewFormItem("Owner", makeCharacterActionLabel(
			r.characterID,
			r.characterName,
			u.ShowEveEntityInfoWindow,
		)),
		widget.NewFormItem("Item", item),
		widget.NewFormItem("Group", widget.NewLabel(r.groupName)),
		widget.NewFormItem("Category", widget.NewLabel(r.categoryName)),
		widget.NewFormItem("Location", location),
		widget.NewFormItem("Region", region),
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
	if u.IsDeveloperMode() {
		items = append(items, widget.NewFormItem("Item ID", u.makeCopyToClipboardLabel(fmt.Sprint(r.itemID))))
	}

	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	subTitle := fmt.Sprintf("Asset #%d", r.itemID)
	setDetailWindow(detailWindowParams{
		content: f,
		imageAction: func() {
			u.ShowTypeInfoWindow(r.typeID)
		},
		imageLoader: func() (fyne.Resource, error) {
			switch r.variant {
			case app.VariantSKIN:
				return u.eis.InventoryTypeSKIN(r.typeID, app.IconPixelSize)
			case app.VariantBPO:
				return u.eis.InventoryTypeBPO(r.typeID, app.IconPixelSize)
			case app.VariantBPC:
				return u.eis.InventoryTypeBPC(r.typeID, app.IconPixelSize)
			default:
				return u.eis.InventoryTypeIcon(r.typeID, app.IconPixelSize)
			}
		},
		minSize: fyne.NewSize(500, 450),
		title:   subTitle,
		window:  w,
	})
	w.Show()
}
