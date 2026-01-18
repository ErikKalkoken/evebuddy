package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/asset"
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
	groupID         int32
	groupName       string
	isSingleton     bool
	itemID          int64
	location        *app.EveLocationShort
	locationDisplay []widget.RichTextSegment
	locationName    string
	owner           *app.EveEntity
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

func newCharacterAssetRow(ca *app.CharacterAsset, assetCollection asset.Collection, characterName func(int32) string) assetRow {
	r := assetRow{
		categoryName:    ca.Type.Group.Category.Name,
		groupID:         ca.Type.Group.ID,
		groupName:       ca.Type.Group.Name,
		isSingleton:     ca.IsSingleton,
		itemID:          ca.ItemID,
		typeID:          ca.Type.ID,
		typeName:        ca.Type.Name,
		typeNameDisplay: ca.DisplayName2(),
		variant:         ca.Variant(),
		owner: &app.EveEntity{
			ID:       ca.CharacterID,
			Name:     characterName(ca.CharacterID),
			Category: app.EveEntityCharacter,
		},
	}
	r.setQuantity(ca.IsSingleton, ca.Quantity)
	r.setLocation(assetCollection, ca.ItemID)
	r.setPrice(ca.Price, ca.Quantity, ca.IsBlueprintCopy)
	return r
}

func newCorporationAssetRow(ca *app.CorporationAsset, assetCollection asset.Collection, corporationName string) assetRow {
	r := assetRow{
		categoryName:    ca.Type.Group.Category.Name,
		groupID:         ca.Type.Group.ID,
		groupName:       ca.Type.Group.Name,
		isSingleton:     ca.IsSingleton,
		itemID:          ca.ItemID,
		typeID:          ca.Type.ID,
		typeName:        ca.Type.Name,
		typeNameDisplay: ca.DisplayName2(),
		variant:         ca.Variant(),
		owner: &app.EveEntity{
			ID:       ca.CorporationID,
			Name:     corporationName,
			Category: app.EveEntityCorporation,
		},
	}
	r.setQuantity(ca.IsSingleton, ca.Quantity)
	r.setLocation(assetCollection, ca.ItemID)
	r.setPrice(ca.Price, ca.Quantity, ca.IsBlueprintCopy)
	return r
}

func (r *assetRow) setLocation(assetCollection asset.Collection, itemID int64) {
	ln, ok := assetCollection.AssetLocation(itemID)
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

func (r *assetRow) setPrice(price optional.Optional[float64], quantity int, isBPC bool) {
	if !isBPC {
		r.price = price
	}
	r.priceDisplay = r.price.StringFunc("?", func(v float64) string {
		return ihumanize.NumberF(v, 1)
	})
	if !r.price.IsEmpty() {
		r.total.Set(price.ValueOrZero() * float64(quantity))
	}
	r.totalDisplay = r.total.StringFunc("?", func(v float64) string {
		return humanize.FormatFloat(app.FloatFormat, v)
	})
}

type assets struct {
	widget.BaseWidget

	onUpdate func(int, string)

	body           fyne.CanvasObject
	columnSorter   *iwidget.ColumnSorter
	corporation    atomic.Pointer[app.Corporation]
	forCorporation bool // reports whether it runs in corporation mode
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
	top            *widget.Label
	u              *baseUI
}

const (
	assetsColItem     = 0
	assetsColGroup    = 1
	assetsColLocation = 2
	assetsColQuantity = 3
	assetsColTotal    = 4
	assetsColOwner    = 5
)

func newAssetsForCharacters(u *baseUI) *assets {
	return newAssets(u, false)
}

func newAssetsForCorporation(u *baseUI) *assets {
	return newAssets(u, true)
}

func newAssets(u *baseUI, forCorporation bool) *assets {
	headers := iwidget.NewDataTableDef([]iwidget.ColumnDef{
		{
			Col:   assetsColItem,
			Label: "Item",
			Width: 300,
		},
		{
			Col:   assetsColGroup,
			Label: "Group",
			Width: 200,
		},
		{
			Col:   assetsColLocation,
			Label: "Location",
			Width: columnWidthLocation,
		},
		{
			Col:   assetsColQuantity,
			Label: "Qty.",
			Width: 100,
		},
		{
			Col:   assetsColTotal,
			Label: "Total",
			Width: 150,
		},
		{
			Col:   assetsColOwner,
			Label: "Owner",
			Width: columnWidthEntity,
		},
	})
	a := &assets{
		columnSorter:   headers.NewColumnSorter(assetsColItem, iwidget.SortAsc),
		forCorporation: forCorporation,
		found:          widget.NewLabel(""),
		rowsFiltered:   make([]assetRow, 0),
		search:         widget.NewEntry(),
		top:            makeTopLabel(),
		u:              u,
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

	if a.u.isMobile {
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
					return iwidget.RichTextSegmentsFromText(r.owner.Name)
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

	// Signals
	if a.forCorporation {
		a.u.currentCorporationExchanged.AddListener(func(_ context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			a.update()
		})
		a.u.corporationSectionChanged.AddListener(func(_ context.Context, arg corporationSectionUpdated) {
			if corporationIDOrZero(a.corporation.Load()) != arg.corporationID {
				return
			}
			if arg.section != app.SectionCorporationAssets {
				return
			}
			a.update()
		})
	} else {
		a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
			if arg.section == app.SectionCharacterAssets {
				a.update()
			}
		})
		a.u.characterAdded.AddListener(func(_ context.Context, _ *app.Character) {
			a.update()
		})
		a.u.characterRemoved.AddListener(func(_ context.Context, _ *app.EntityShort[int32]) {
			a.update()
		})
		a.u.tagsChanged.AddListener(func(ctx context.Context, s struct{}) {
			a.update()
		})
	}
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
		a.selectTotal,
	)
	if !a.forCorporation {
		filters.Add(a.selectTag)
		filters.Add(a.selectOwner)
	}
	if a.u.isMobile {
		filters.Add(container.NewHBox(a.sortButton))
	}
	topBox := container.NewVBox(
		container.NewBorder(nil, nil, nil, a.found, a.top),
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
			if a.forCorporation {
				owner.Hide()
			}
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
		rows = xslices.Filter(rows, func(r assetRow) bool {
			return r.owner.Name == x
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
				x = xstrings.CompareIgnoreCase(a.owner.Name, b.owner.Name)
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
	a.selectCategory.SetOptions(xslices.Map(rows, func(r assetRow) string {
		return r.categoryName
	}))
	a.selectGroup.SetOptions(xslices.Map(rows, func(r assetRow) string {
		return r.groupName
	}))
	a.selectOwner.SetOptions(xslices.Map(rows, func(r assetRow) string {
		return r.owner.Name
	}))
	a.selectRegion.SetOptions(xslices.Map(rows, func(r assetRow) string {
		return r.regionName
	}))
	a.selectLocation.SetOptions(xslices.Map(rows, func(r assetRow) string {
		return r.locationName
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
	clear := func() {
		if a.onUpdate != nil {
			a.onUpdate(0, "")
		}
		fyne.Do(func() {
			a.found.Hide()
			r := []assetRow{}
			a.rows = r
			a.rowsFiltered = r
			a.body.Refresh()
		})
	}
	setTop := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.top.Text = s
			a.top.Importance = i
			a.top.Refresh()
		})
	}
	if !a.forCorporation && a.characterCount() == 0 {
		clear()
		setTop("No characters", widget.LowImportance)
		return
	}
	var rows []assetRow
	var quantity int
	var err error
	var value float64
	ctx := context.Background()
	if a.forCorporation {
		rows, quantity, value, err = a.fetchRowsForCorporation(ctx)
	} else {
		rows, quantity, value, err = a.fetchRowsForCharacters(ctx)
	}
	if err != nil {
		slog.Error("Failed to refresh asset data", "err", err)
		clear()
		setTop("ERROR: "+a.u.humanizeError(err), widget.DangerImportance)
		return
	}
	top := fmt.Sprintf("%s items - %s ISK est.", ihumanize.Number(quantity, 1), ihumanize.NumberF(value, 1))
	setTop(top, widget.MediumImportance)
	if a.onUpdate != nil {
		a.onUpdate(quantity, top)
	}
	fyne.Do(func() {
		a.updateFoundInfo()
	})
	fyne.Do(func() {
		a.rowsFiltered = rows
		a.rows = rows
		a.filterRows(-1)
	})
}

func (a *assets) fetchRowsForCharacters(ctx context.Context) ([]assetRow, int, float64, error) {
	cc, err := a.u.cs.ListCharactersShort(ctx)
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
		tags, err := a.u.cs.ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			return nil, 0, 0, nil
		}
		tagsPerCharacter[c.ID] = tags
	}
	assets, err := a.u.cs.ListAllAssets(ctx)
	if err != nil {
		return nil, 0, 0, err
	}
	locations, err := a.u.eus.ListLocations(ctx)
	if err != nil {
		return nil, 0, 0, err
	}
	ac := asset.New(asset.ItemsFromCharacterAssets(assets), locations)
	rows := make([]assetRow, len(assets))
	var quantity int
	var total float64
	for i, ca := range assets {
		r := newCharacterAssetRow(ca, ac, func(id int32) string {
			return characterNames[id]
		})
		r.searchTarget = strings.ToLower(r.typeNameDisplay)
		r.tags = tagsPerCharacter[ca.CharacterID]
		rows[i] = r
		quantity += r.quantity
		total += r.total.ValueOrZero()
	}
	return rows, quantity, total, nil
}

func (a *assets) fetchRowsForCorporation(ctx context.Context) ([]assetRow, int, float64, error) {
	c := a.corporation.Load()
	if c == nil {
		return []assetRow{}, 0, 0, nil
	}
	assets, err := a.u.rs.ListAssets(ctx, c.ID)
	if err != nil {
		return nil, 0, 0, err
	}
	locations, err := a.u.eus.ListLocations(ctx)
	if err != nil {
		return nil, 0, 0, err
	}
	ac := asset.New(asset.ItemsFromCorporationAssets(assets), locations)
	rows := make([]assetRow, len(assets))
	var quantity int
	var total float64
	for i, ca := range assets {
		r := newCorporationAssetRow(ca, ac, c.EveCorporation.Name)
		r.searchTarget = strings.ToLower(r.typeNameDisplay)
		rows[i] = r
		quantity += r.quantity
		total += r.total.ValueOrZero()
	}
	return rows, quantity, total, nil
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
		fmt.Sprintf("asset-%d-%d", r.owner.ID, r.itemID),
		"Asset: Information",
		r.owner.Name,
	)
	if !created {
		w.Show()
		return
	}
	item := makeLinkLabelWithWrap(r.typeNameDisplay, func() {
		if r.owner.IsCharacter() {
			u.ShowTypeInfoWindowWithCharacter(r.typeID, r.owner.ID)
		} else {
			u.ShowTypeInfoWindow(r.typeID)
		}
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
			r.owner.ID,
			r.owner.Name,
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
