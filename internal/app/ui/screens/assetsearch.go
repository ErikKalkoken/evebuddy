package screens

import (
	"cmp"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/asset"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	assetSearchTotalYes = "Has total"
	assetSearchTotalNo  = "Has no total"
)

// Names of the asset search filters, used as labels on desktop and as option names on mobile.
const (
	assetSearchFilterCategory = "Category"
	assetSearchFilterGroup    = "Group"
	assetSearchFilterLocation = "Location"
	assetSearchFilterOwner    = "Owner"
	assetSearchFilterRegion   = "Region"
	assetSearchFilterState    = "State"
	assetSearchFilterTag      = "Tag"
	assetSearchFilterTotal    = "Total"
)

// assetSearchFilter is the selected value of each asset search filter. Empty means not filtered.
type assetSearchFilter struct {
	category string
	group    string
	location string
	owner    string
	region   string
	state    string
	tag      string
	total    string
}

// match reports whether row r passes all filters.
func (f assetSearchFilter) match(r assetRow) bool {
	switch {
	case f.category != "" && r.categoryName != f.category,
		f.group != "" && r.groupName != f.group,
		f.location != "" && r.locationName != f.location,
		f.owner != "" && r.owner.Name != f.owner,
		f.region != "" && r.regionName != f.region,
		f.state != "" && r.state != f.state,
		f.tag != "" && !r.tags.Contains(f.tag):
		return false
	}
	switch f.total {
	case assetSearchTotalYes:
		return !r.total.IsEmpty()
	case assetSearchTotalNo:
		return r.total.IsEmpty()
	}
	return true
}

type assetRow struct {
	categoryID      int64
	categoryName    string
	displayName     string
	groupID         int64
	groupName       string
	isSingleton     bool
	itemID          int64
	location        *app.EveLocationShort
	locationDisplay []widget.RichTextSegment
	locationFlag    app.LocationFlag
	locationName    string
	locationPath    []string
	name            string
	owner           *app.EveEntity
	price           optional.Optional[float64]
	priceDisplay    string
	quantity        int
	quantityDisplay string
	regionID        int64
	regionName      string
	searchTarget    string
	solarSystemID   int64
	solarSystemName string
	state           string
	tags            set.Set[string]
	tagsDisplay     string
	total           optional.Optional[float64]
	totalDisplay    string
	typeID          int64
	typeName        string
	variant         app.InventoryTypeVariant
}

func newCharacterAssetRow(ca *app.CharacterAsset, ac asset.Tree, characterName func(int64) string) assetRow {
	owner := &app.EveEntity{
		ID:       ca.CharacterID,
		Name:     characterName(ca.CharacterID),
		Category: app.EveEntityCharacter,
	}
	r := assetRow{
		categoryID:   ca.Type.Group.Category.ID,
		categoryName: ca.Type.Group.Category.Name,
		displayName:  ca.DisplayName2(),
		groupID:      ca.Type.Group.ID,
		groupName:    ca.Type.Group.Name,
		isSingleton:  ca.IsSingleton,
		itemID:       ca.ItemID,
		name:         ca.Name,
		owner:        owner,
		typeID:       ca.Type.ID,
		typeName:     ca.Type.Name,
		variant:      ca.Variant(),
	}
	r.setQuantity(ca.IsSingleton, ca.Quantity)
	r.setLocation(ac, ca.ItemID)
	r.setLocationFlag(ac, ca.ItemID)
	r.setPrice(ca.Price, ca.Quantity, ca.IsBlueprintCopy)
	return r
}

func newCorporationAssetRow(ca *app.CorporationAsset, ac asset.Tree, corporationName string) assetRow {
	owner := &app.EveEntity{
		ID:       ca.CorporationID,
		Name:     corporationName,
		Category: app.EveEntityCorporation,
	}
	r := assetRow{
		categoryID:   ca.Type.Group.Category.ID,
		categoryName: ca.Type.Group.Category.Name,
		displayName:  ca.DisplayName2(),
		groupID:      ca.Type.Group.ID,
		groupName:    ca.Type.Group.Name,
		isSingleton:  ca.IsSingleton,
		itemID:       ca.ItemID,
		name:         ca.Name,
		owner:        owner,
		typeID:       ca.Type.ID,
		typeName:     ca.Type.Name,
		variant:      ca.Variant(),
	}
	r.setQuantity(ca.IsSingleton, ca.Quantity)
	r.setLocation(ac, ca.ItemID)
	r.setLocationFlag(ac, ca.ItemID)
	r.setPrice(ca.Price, ca.Quantity, ca.IsBlueprintCopy)
	return r
}

func (r *assetRow) setLocationFlag(ac asset.Tree, itemID int64) {
	n, ok := ac.Node(itemID)
	if !ok {
		return
	}
	it, ok := n.Asset()
	if !ok {
		return
	}
	r.locationFlag = it.LocationFlag
}

func (r *assetRow) setLocation(ac asset.Tree, itemID int64) {
	ln, ok := ac.LocationForItem(itemID)
	if !ok {
		r.locationDisplay = xwidget.RichTextSegmentsFromText("?")
		return
	}
	el, ok := ln.Location()
	if !ok {
		r.locationDisplay = xwidget.RichTextSegmentsFromText("?")
		return
	}
	r.location = el.ToEveLocationShort()
	r.locationName = el.DisplayName()
	r.locationDisplay = el.DisplayRichText()
	n, ok := ac.Node(itemID)
	if ok {
		if p := n.Path(); len(p) > 0 {
			r.locationPath = xslices.Map(p[:len(p)-1], func(x *asset.Node) string {
				return x.String()
			})
			if len(p) > 1 {
				switch p[1].Category() {
				case asset.NodeAssetSafetyCharacter, asset.NodeAssetSafetyCorporation:
					r.state = "Asset Safety"
				case asset.NodeDeliveries:
					r.state = "Deliveries"
				case asset.NodeImpounded:
					r.state = "Impounded"
				case asset.NodeInSpace:
					r.state = "In Space"
				case asset.NodeItemHangar, asset.NodeShipHangar:
					r.state = "Personal"
				case asset.NodeOfficeFolder:
					r.state = "Office"
				default:
					r.state = "Other"
				}
			}
		}
	}
	if v, ok := el.SolarSystem.Value(); ok {
		r.solarSystemID = v.ID
		r.solarSystemName = v.Name
		r.regionName = v.Constellation.Region.Name
		r.regionID = v.Constellation.Region.ID
	}
}

func (r *assetRow) setQuantity(isSingleton bool, quantity int) {
	if isSingleton {
		r.quantityDisplay = "1*"
		r.quantity = 1
	} else {
		r.quantityDisplay = humanize.Comma(int64(quantity))
		r.quantity = quantity
	}
}

func (r *assetRow) setPrice(price optional.Optional[float64], quantity int, isBPC optional.Optional[bool]) {
	if !isBPC.ValueOrZero() {
		r.price = price
	}
	r.priceDisplay = r.price.StringFunc("?", func(v float64) string {
		return ihumanize.NumberF(v, 1)
	})
	if v, ok := r.price.Value(); ok {
		r.total.Set(v * float64(quantity))
	}
	r.totalDisplay = r.total.StringFunc("?", func(v float64) string {
		return humanize.FormatFloat(ui.FloatFormatISK, v)
	})
}

type AssetSearch struct {
	widget.BaseWidget

	body           fyne.CanvasObject
	columnSorter   *xwidget.ColumnSorter[assetRow]
	corporation    atomic.Pointer[app.Corporation]
	filterChip     *xwidget.FilterChipCompact // only on mobile
	filterRun      latestRun
	footer         *widget.Label
	forCorporation bool // reports whether it runs in corporation mode
	rows           []assetRow
	rowsFiltered   []assetRow
	searchEntry    *xwidget.SearchEntry
	selectCategory *kxwidget.FilterChipSelect // select chips only on desktop
	selectGroup    *kxwidget.FilterChipSelect
	selectLocation *kxwidget.FilterChipSelect
	selectOwner    *kxwidget.FilterChipSelect
	selectRegion   *kxwidget.FilterChipSelect
	selectState    *kxwidget.FilterChipSelect
	selectTag      *kxwidget.FilterChipSelect
	selectTotal    *kxwidget.FilterChipSelect
	sortChip       *kxwidget.SortChip
	top            *widget.Label
	u              baseUI
}

func NewAssetSearchForAll(u baseUI) *AssetSearch {
	return newAssetSearch(u, false)
}

func NewAssetSearchForCorporation(u baseUI) *AssetSearch {
	return newAssetSearch(u, true)
}

func newAssetSearch(u baseUI, forCorporation bool) *AssetSearch {
	corporationIcon := theme.NewThemedResource(icons.StarCircleOutlineSvg)
	cols := []xwidget.DataColumn[assetRow]{{
		Label: "Item",
		Width: 300,
		Sort: func(a, b assetRow) int {
			return strings.Compare(a.displayName, b.displayName)
		},
		Create: func() fyne.CanvasObject {
			icon := xwidget.NewImageFromResource(
				icons.BlankSvg,
				fyne.NewSquareSize(ui.IconUnitSize),
			)
			name := widget.NewLabel("Template")
			name.Truncation = fyne.TextTruncateClip
			return container.NewBorder(nil, nil, icon, nil, name)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			border[0].(*widget.Label).SetText(r.typeName)
			x := border[1].(*canvas.Image)
			u.EVEImage().AssetIconAsync(r.typeID, r.variant, ui.IconPixelSize, func(r fyne.Resource) {
				x.Resource = r
				x.Refresh()
			})
		},
	}, {
		Label: "Group",
		Width: 200,
		Sort: func(a, b assetRow) int {
			return strings.Compare(a.groupName, b.groupName)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.groupName)
		},
	}, {
		Label: "Location",
		Width: ui.ColumnWidthLocation,
		Sort: func(a, b assetRow) int {
			return strings.Compare(a.locationName, b.locationName)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.locationDisplay)
		},
	}, {
		Label: "State",
		Width: 90,
		Sort: func(a, b assetRow) int {
			return strings.Compare(a.locationName, b.locationName)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.state)
		},
	}, {
		Label: "Qty.",
		Width: 100,
		Sort: func(a, b assetRow) int {
			return cmp.Compare(a.quantity, b.quantity)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.quantityDisplay, widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			})
		},
	}, {
		Label: "Total",
		Width: 150,
		Sort: func(a, b assetRow) int {
			return optional.Compare(a.total, b.total)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.totalDisplay, widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			})
		},
	}}
	if !forCorporation {
		cols = slices.Concat(cols, []xwidget.DataColumn[assetRow]{{
			Label: "Owner",
			Width: 250,
			Sort: func(a, b assetRow) int {
				return xstrings.CompareIgnoreCase(a.owner.Name, b.owner.Name)
			},
			Create: func() fyne.CanvasObject {
				icon := widget.NewIcon(icons.BlankSvg)
				name := widget.NewLabel("Template")
				name.Truncation = fyne.TextTruncateClip
				return container.NewBorder(nil, nil, icon, nil, name)
			},
			Update: func(r assetRow, co fyne.CanvasObject) {
				border := co.(*fyne.Container).Objects
				border[0].(*widget.Label).SetText(r.owner.Name)
				icon := border[1].(*widget.Icon)
				if r.owner.IsCharacter() {
					icon.SetResource(theme.AccountIcon())
				} else {
					icon.SetResource(corporationIcon)
				}
			},
		}, {
			Label: "Tags",
			Width: ui.ColumnWidthEntity,
			Update: func(r assetRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.tagsDisplay)
			},
		}})
	}
	columns := xwidget.NewDataColumns(cols)
	a := &AssetSearch{
		columnSorter:   xwidget.NewColumnSorter(columns, "Item", xwidget.SortAsc),
		forCorporation: forCorporation,
		footer:         ui.NewLabelWithTruncation(""),
		top:            ui.NewLabelWithWrapping(""),
		u:              u,
	}
	a.ExtendBaseWidget(a)

	if a.u.IsMobile() {
		a.body = a.makeDataList()
	} else {
		a.body = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter, a.filterRowsAsync, func(_ int, r assetRow) {
				ShowAssetDetails(u, r)
			})
	}

	// filters
	placeholder := "Search items"
	if a.u.IsMobile() {
		placeholder = "Search" // shares the row with the chips
	}
	a.searchEntry = xwidget.NewSearchEntry(placeholder, func(_ string) {
		a.filterRowsAsync("")
	})

	if a.u.IsMobile() {
		a.filterChip = xwidget.NewFilterChipCompact(nil, func(map[string]string) {
			a.filterRowsAsync("")
		})
	} else {
		makeSelect := func(label string) *kxwidget.FilterChipSelect {
			return kxwidget.NewFilterChipSelect(label, []string{}, func(string) {
				a.filterRowsAsync("")
			})
		}
		makeSelectWithSearch := func(label string) *kxwidget.FilterChipSelect {
			return kxwidget.NewFilterChipSelectWithSearch(label, []string{}, func(string) {
				a.filterRowsAsync("")
			}, a.u.MainWindow())
		}
		a.selectCategory = makeSelectWithSearch(assetSearchFilterCategory)
		a.selectGroup = makeSelectWithSearch(assetSearchFilterGroup)
		a.selectOwner = makeSelectWithSearch(assetSearchFilterOwner)
		a.selectRegion = makeSelectWithSearch(assetSearchFilterRegion)
		a.selectLocation = makeSelectWithSearch(assetSearchFilterLocation)
		a.selectState = makeSelect(assetSearchFilterState)
		a.selectTotal = makeSelect(assetSearchFilterTotal)
		a.selectTotal.SetOptions([]string{assetSearchTotalYes, assetSearchTotalNo})
		a.selectTag = makeSelect(assetSearchFilterTag)
	}
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync("")
	})

	// Signals
	a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})

	if a.forCorporation {
		a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			fyne.Do(func() {
				a.searchEntry.ClearSilent()
				if a.filterChip != nil {
					a.filterChip.ResetSilent()
					return
				}
				a.selectCategory.Selected = ""
				a.selectGroup.Selected = ""
				a.selectLocation.Selected = ""
				a.selectOwner.Selected = ""
				a.selectRegion.Selected = ""
				a.selectState.Selected = ""
				a.selectTag.Selected = ""
				a.selectTotal.Selected = ""
			})
			a.update(ctx)
		})
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if a.corporation.Load().IDOrZero() != arg.CorporationID {
				return
			}
			if arg.Section != app.SectionCorporationAssets {
				return
			}
			a.update(ctx)
		})
	} else {
		a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
			if arg.Section == app.SectionCharacterAssets {
				a.update(ctx)
			}
		})
		a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
			a.update(ctx)
		})
		a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			a.update(ctx)
		})
		a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, _ struct{}) {
			a.update(ctx)
		})
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if arg.Section == app.SectionCorporationAssets {
				a.update(ctx)
			}
		})
	}
	a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		if arg.Section == app.SectionEveMarketPrices {
			a.update(ctx)
		}
	})
	return a
}

func (a *AssetSearch) CreateRenderer() fyne.WidgetRenderer {
	topBox := container.NewVBox(a.top)
	if a.u.IsMobile() {
		topBox.Add(container.NewBorder(nil, nil, nil, container.NewHBox(a.filterChip, a.sortChip), a.searchEntry))
	} else {
		filters := container.NewHBox(
			a.selectCategory,
			a.selectGroup,
			a.selectRegion,
			a.selectLocation,
			a.selectState,
			a.selectTotal,
		)
		if !a.forCorporation {
			filters.Add(a.selectTag)
			filters.Add(a.selectOwner)
		}
		topBox.Add(container.NewBorder(nil, nil, filters, nil, a.searchEntry))
	}
	c := container.NewBorder(topBox, a.footer, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

// currentFilter returns the selected filters: from the compact chip on mobile
// and from the filter chips on desktop.
func (a *AssetSearch) currentFilter() assetSearchFilter {
	if a.filterChip != nil {
		s := a.filterChip.Selected()
		return assetSearchFilter{
			category: s[assetSearchFilterCategory],
			group:    s[assetSearchFilterGroup],
			location: s[assetSearchFilterLocation],
			owner:    s[assetSearchFilterOwner],
			region:   s[assetSearchFilterRegion],
			state:    s[assetSearchFilterState],
			tag:      s[assetSearchFilterTag],
			total:    s[assetSearchFilterTotal],
		}
	}
	return assetSearchFilter{
		category: a.selectCategory.Selected,
		group:    a.selectGroup.Selected,
		location: a.selectLocation.Selected,
		owner:    a.selectOwner.Selected,
		region:   a.selectRegion.Selected,
		state:    a.selectState.Selected,
		tag:      a.selectTag.Selected,
		total:    a.selectTotal.Selected,
	}
}

func (a *AssetSearch) makeDataList() *xwidget.StripedList {
	p := theme.Padding()
	l := xwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Template", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			owner := widget.NewLabel("Template")
			if a.forCorporation {
				owner.Hide()
			}
			location := xwidget.NewRichTextWithText("Template")
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
				title = r.displayName
			} else {
				title = fmt.Sprintf("%s x%s", r.displayName, r.quantityDisplay)
			}
			box[0].(*widget.Label).SetText(title)
			box[1].(*xwidget.RichText).Set(r.locationDisplay)
			box[2].(*widget.Label).SetText(r.owner.Name)
			box[3].(*widget.Label).SetText(r.totalDisplay)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		ShowAssetDetails(a.u, r)
	}
	return l
}

func (a *AssetSearch) Focus() {
	a.u.MainWindow().Canvas().Focus(a.searchEntry)
}

// MoreItems returns the list of menu items for the overflow menu.
func (a *AssetSearch) MoreItems() []*fyne.MenuItem {
	return []*fyne.MenuItem{
		fyne.NewMenuItem("Copy assets to clipboard", a.consolidateToClipboard),
		fyne.NewMenuItem("Export assets as CSV", a.exportAsCSV),
	}
}

func (a *AssetSearch) consolidateToClipboard() {
	copyRowsToClipboard(a.u, "assets", a.rowsFiltered, consolidateAssetRows)
}

func consolidateAssetRows(rows []assetRow) (string, error) {
	type assetItem struct {
		name     string
		quantity int
	}
	quantities := make(map[int64]int)
	names := make(map[int64]string)
	for _, r := range rows {
		quantities[r.typeID] += r.quantity
		if _, found := names[r.typeID]; !found {
			names[r.typeID] = r.typeName
		}
	}
	var items []assetItem
	for id, name := range names {
		items = append(items, assetItem{
			name:     name,
			quantity: quantities[id],
		})
	}
	slices.SortFunc(items, func(a, b assetItem) int {
		return strings.Compare(a.name, b.name)
	})
	var b strings.Builder
	for _, it := range items {
		if _, err := fmt.Fprintf(&b, "%s %d\n", it.name, it.quantity); err != nil {
			return "", err
		}
	}
	return b.String(), nil
}

func (a *AssetSearch) exportAsCSV() {
	var filename string
	if a.forCorporation {
		filename = fmt.Sprintf("assets_%d.csv", a.corporation.Load().IDOrZero())
	} else {
		filename = "assets.csv"
	}
	exportRowsAsCSV(a.u, "assets", filename, a.rowsFiltered, func(w io.Writer, rows []assetRow) error {
		return writeAssetRowsToCSV(w, rows, a.forCorporation)
	})
}

func writeAssetRowsToCSV(w io.Writer, rows []assetRow, forCorporation bool) error {
	cw := csv.NewWriter(w)
	header := []string{
		"Item ID", "Type ID", "Type Name", "Item Name", "Group ID", "Group Name", "Category ID", "Category Name",
		"Location Name", "Location Flag", "State", "Quantity", "Is Singleton", "Variant",
		"Solar System ID", "Solar System Name", "Region ID", "Region Name", "Price", "Total",
		"Owner ID", "Owner Name",
	}
	if !forCorporation {
		header = append(header, "Tags")
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, r := range rows {
		var price string
		if v, ok := r.price.Value(); ok {
			price = strconv.FormatFloat(v, 'f', -1, 64)
		}
		var total string
		if v, ok := r.total.Value(); ok {
			total = strconv.FormatFloat(v, 'f', -1, 64)
		}
		var variant string
		if r.variant != app.VariantRegular {
			variant = r.variant.String()
		}
		record := []string{
			strconv.FormatInt(r.itemID, 10),
			strconv.FormatInt(r.typeID, 10),
			r.typeName,
			r.name,
			strconv.FormatInt(r.groupID, 10),
			r.groupName,
			strconv.FormatInt(r.categoryID, 10),
			r.categoryName,
			r.locationName,
			r.locationFlag.String(),
			r.state,
			strconv.Itoa(r.quantity),
			strconv.FormatBool(r.isSingleton),
			variant,
			strconv.FormatInt(r.solarSystemID, 10),
			r.solarSystemName,
			strconv.FormatInt(r.regionID, 10),
			r.regionName,
			price,
			total,
			strconv.FormatInt(r.owner.ID, 10),
			r.owner.Name,
		}
		if !forCorporation {
			record = append(record, r.tagsDisplay)
		}
		if err := cw.Write(record); err != nil {
			return err
		}

	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}
	return nil
}

func (a *AssetSearch) filterRowsAsync(sortCol string) {
	isLatest := a.filterRun.start()
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	filter := a.currentFilter()
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	runAsync(func() {
		rows = slices.DeleteFunc(rows, func(r assetRow) bool {
			return !filter.match(r)
		})
		// search filter
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r assetRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		// set data & refresh
		tagOptions := slices.Collect(xiter.Chain(xslices.Map(rows, func(r assetRow) iter.Seq[string] {
			return r.tags.All()
		})...))
		categoryOptions := xslices.Map(rows, func(r assetRow) string {
			return r.categoryName
		})
		groupOptions := xslices.Map(rows, func(r assetRow) string {
			return r.groupName
		})
		locationOptions := xslices.Map(rows, func(r assetRow) string {
			return r.locationName
		})
		ownerOptions := xslices.Map(rows, func(r assetRow) string {
			return r.owner.Name
		})
		regionOptions := xslices.Map(rows, func(r assetRow) string {
			return r.regionName
		})
		stateOptions := xslices.Map(rows, func(r assetRow) string {
			return r.state
		})

		footer := fmt.Sprintf("Showing %s / %s items", ihumanize.Comma(len(rows)), ihumanize.Comma(totalRows))
		var value optional.Optional[float64]
		for _, r := range rows {
			value = optional.SumNonEmpty(value, r.total)
		}
		if v, ok := value.Value(); ok {
			footer += fmt.Sprintf(" • %s ISK est. price", ihumanize.Comma(int(v)))
		}

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			if a.filterChip != nil {
				options := []xwidget.FilterOption{
					xwidget.NewFilterOptionMultiChoiceWithSearch(assetSearchFilterCategory, categoryOptions),
					xwidget.NewFilterOptionMultiChoiceWithSearch(assetSearchFilterGroup, groupOptions),
					xwidget.NewFilterOptionMultiChoiceWithSearch(assetSearchFilterRegion, regionOptions),
					xwidget.NewFilterOptionMultiChoiceWithSearch(assetSearchFilterLocation, locationOptions),
					xwidget.NewFilterOptionMultiChoice(assetSearchFilterState, stateOptions),
					xwidget.NewFilterOptionMultiChoice(assetSearchFilterTotal, []string{assetSearchTotalYes, assetSearchTotalNo}),
				}
				if !a.forCorporation {
					options = append(options,
						xwidget.NewFilterOptionMultiChoice(assetSearchFilterTag, tagOptions),
						xwidget.NewFilterOptionMultiChoiceWithSearch(assetSearchFilterOwner, ownerOptions),
					)
				}
				a.filterChip.SetOptions(options...)
			} else {
				a.selectCategory.SetOptions(categoryOptions)
				a.selectGroup.SetOptions(groupOptions)
				a.selectLocation.SetOptions(locationOptions)
				a.selectOwner.SetOptions(ownerOptions)
				a.selectRegion.SetOptions(regionOptions)
				a.selectState.SetOptions(stateOptions)
				a.selectTag.SetOptions(tagOptions)
			}
			a.rowsFiltered = rows
			a.body.Refresh()
			switch x := a.body.(type) {
			case *widget.Table:
				x.ScrollToTop()
			}
		})
	})
}

func (a *AssetSearch) update(ctx context.Context) {
	reset := func() {
		fyne.Do(func() {
			xslices.Clear(&a.rows)
			a.filterRowsAsync("")
		})
	}
	setTop := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.top.Text = s
			a.top.Importance = i
			a.top.Refresh()
			a.top.Show()
		})
	}
	if !a.forCorporation {
		n, err := a.characterCount(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Failed to refresh asset data", "err", err)
			reset()
			setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
			return
		}
		if n == 0 {
			reset()
			setTop("No characters", widget.LowImportance)
			return
		}
	}
	var rows []assetRow
	var err error
	if a.forCorporation {
		rows, err = a.fetchRowsForCorporation(ctx)
	} else {
		rows, err = a.fetchRowsForAll(ctx)
	}
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh asset data", "err", err)
		reset()
		setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	fyne.Do(func() {
		a.top.Hide()
		a.rows = rows
		a.filterRowsAsync("")
	})
}

func (a *AssetSearch) fetchRowsForAll(ctx context.Context) ([]assetRow, error) {
	r1, err := a.fetchRowsForCharacters(ctx)
	if err != nil {
		return nil, err
	}
	r2, err := a.fetchRowsForCorporations(ctx)
	if err != nil {
		return nil, err
	}
	return slices.Concat(r1, r2), nil
}

func (a *AssetSearch) fetchRowsForCharacters(ctx context.Context) ([]assetRow, error) {
	characters, err := a.u.Character().CharacterNames(ctx)
	if err != nil {
		return nil, err
	}
	if len(characters) == 0 {
		return nil, nil
	}
	tagsPerCharacter := make(map[int64]set.Set[string])
	for id := range characters {
		tags, err := a.u.Character().ListTagsForCharacter(ctx, id)
		if err != nil {
			return nil, nil
		}
		tagsPerCharacter[id] = tags
	}
	assets, err := a.u.Character().ListAllAssets(ctx)
	if err != nil {
		return nil, err
	}
	locations, err := a.u.EVEUniverse().ListLocations(ctx)
	if err != nil {
		return nil, err
	}
	ac := asset.NewFromCharacterAssets(assets, locations)
	var rows []assetRow
	for _, ca := range assets {
		r := newCharacterAssetRow(ca, ac, func(id int64) string {
			return characters[id]
		})
		r.searchTarget = strings.ToLower(r.displayName)
		r.tags = tagsPerCharacter[ca.CharacterID]
		r.tagsDisplay = strings.Join(slices.Sorted(r.tags.All()), ", ")
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *AssetSearch) fetchRowsForCorporations(ctx context.Context) ([]assetRow, error) {
	assets, err := a.u.Corporation().ListAllAssets(ctx)
	if err != nil {
		return nil, err
	}
	return a.fetchRowsForCorporations2(ctx, assets)
}

func (a *AssetSearch) fetchRowsForCorporation(ctx context.Context) ([]assetRow, error) {
	c := a.corporation.Load()
	if c == nil {
		return []assetRow{}, nil
	}
	assets, err := a.u.Corporation().ListAssets(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	return a.fetchRowsForCorporations2(ctx, assets)
}

func (a *AssetSearch) fetchRowsForCorporations2(ctx context.Context, assets []*app.CorporationAsset) ([]assetRow, error) {
	cc, err := a.u.Corporation().ListCorporationsShort(ctx)
	if err != nil {
		return nil, err
	}
	if len(cc) == 0 {
		return nil, nil
	}
	corporationNames := make(map[int64]string)
	for _, o := range cc {
		corporationNames[o.ID] = o.Name
	}
	locations, err := a.u.EVEUniverse().ListLocations(ctx)
	if err != nil {
		return nil, err
	}
	ac := asset.NewFromCorporationAssets(assets, locations)
	var rows []assetRow
	var value float64
	for _, ca := range assets {
		if ca.Type != nil && ca.Type.ID == app.EveTypeOffice {
			continue // filter out office item
		}
		r := newCorporationAssetRow(ca, ac, corporationNames[ca.CorporationID])
		r.searchTarget = strings.ToLower(r.displayName)
		rows = append(rows, r)
		value += r.total.ValueOrZero()
	}
	return rows, nil
}

func (a *AssetSearch) characterCount(ctx context.Context) (int, error) {
	cc, err := a.u.Character().ListCharacterIDs(ctx)
	if err != nil {
		return 0, err
	}
	validCount := 0
	for id := range cc.All() {
		hasSection, err := a.u.Character().HasSection(ctx, id, app.SectionCharacterAssets)
		if err != nil {
			return 0, err
		}
		if hasSection {
			validCount++
		}
	}
	return validCount, nil
}
