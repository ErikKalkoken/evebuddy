package assetui

import (
	"cmp"
	"context"
	"fmt"
	"iter"
	"log/slog"
	"slices"
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
	"github.com/ErikKalkoken/evebuddy/internal/app/awidget"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"

	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

const (
	assetsTotalYes = "Has total"
	assetsTotalNo  = "Has no total"
)

type assetRow struct {
	categoryName    string
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
	tags            set.Set[string]
	tagsDisplay     string
	total           optional.Optional[float64]
	totalDisplay    string
	typeID          int64
	typeName        string
	variant         app.InventoryTypeVariant
	state           string
}

func newCharacterAssetRow(ca *app.CharacterAsset, ac asset.Tree, characterName func(int64) string) assetRow {
	r := assetRow{
		categoryName: ca.Type.Group.Category.Name,
		groupID:      ca.Type.Group.ID,
		groupName:    ca.Type.Group.Name,
		isSingleton:  ca.IsSingleton,
		itemID:       ca.ItemID,
		typeID:       ca.Type.ID,
		typeName:     ca.Type.Name,
		name:         ca.DisplayName2(),
		variant:      ca.Variant(),
		owner: &app.EveEntity{
			ID:       ca.CharacterID,
			Name:     characterName(ca.CharacterID),
			Category: app.EveEntityCharacter,
		},
	}
	r.setQuantity(ca.IsSingleton, ca.Quantity)
	r.setLocation(ac, ca.ItemID)
	r.setLocationFlag(ac, ca.ItemID)
	r.setPrice(ca.Price, ca.Quantity, ca.IsBlueprintCopy)
	return r
}

func newCorporationAssetRow(ca *app.CorporationAsset, ac asset.Tree, corporationName string) assetRow {
	r := assetRow{
		categoryName: ca.Type.Group.Category.Name,
		groupID:      ca.Type.Group.ID,
		groupName:    ca.Type.Group.Name,
		isSingleton:  ca.IsSingleton,
		itemID:       ca.ItemID,
		typeID:       ca.Type.ID,
		typeName:     ca.Type.Name,
		name:         ca.DisplayName2(),
		variant:      ca.Variant(),
		owner: &app.EveEntity{
			ID:       ca.CorporationID,
			Name:     corporationName,
			Category: app.EveEntityCorporation,
		},
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
	r.location = el.ToShort()
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
		return humanize.FormatFloat(app.FloatFormat, v)
	})
}

type Search struct {
	widget.BaseWidget

	body           fyne.CanvasObject
	columnSorter   *xwidget.ColumnSorter[assetRow]
	corporation    atomic.Pointer[app.Corporation]
	footer         *widget.Label
	forCorporation bool // reports whether it runs in corporation mode
	rows           []assetRow
	rowsFiltered   []assetRow
	search         *widget.Entry
	selectCategory *kxwidget.FilterChipSelect
	selectGroup    *kxwidget.FilterChipSelect
	selectLocation *kxwidget.FilterChipSelect
	selectOwner    *kxwidget.FilterChipSelect
	selectRegion   *kxwidget.FilterChipSelect
	selectState    *kxwidget.FilterChipSelect
	selectTag      *kxwidget.FilterChipSelect
	selectTotal    *kxwidget.FilterChipSelect
	sortButton     *xwidget.SortButton
	top            *widget.Label
	u              ui
}

const (
	searchColItem = iota + 1
	searchColGroup
	searchColLocation
	searchColState
	searchColQuantity
	searchColTotal
	searchColOwner
	searchColTags
)

func NewSearchForAll(u ui) *Search {
	return newAssetSearch(u, false)
}

func NewSearchForCorporation(u ui) *Search {
	return newAssetSearch(u, true)
}

func newAssetSearch(u ui, forCorporation bool) *Search {
	corporationIcon := theme.NewThemedResource(icons.StarCircleOutlineSvg)
	cols := []xwidget.DataColumn[assetRow]{{
		ID:    searchColItem,
		Label: "Item",
		Width: 300,
		Sort: func(a, b assetRow) int {
			return strings.Compare(a.name, b.name)
		},
		Create: func() fyne.CanvasObject {
			icon := xwidget.NewImageFromResource(
				icons.BlankSvg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			name := widget.NewLabel("Template")
			name.Truncation = fyne.TextTruncateClip
			return container.NewBorder(nil, nil, icon, nil, name)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			border[0].(*widget.Label).SetText(r.typeName)
			x := border[1].(*canvas.Image)
			loadAssetIconAsync(u.EVEImage(), r.typeID, r.variant, func(r fyne.Resource) {
				x.Resource = r
				x.Refresh()
			})
		},
	}, {
		ID:    searchColGroup,
		Label: "Group",
		Width: 200,
		Sort: func(a, b assetRow) int {
			return strings.Compare(a.groupName, b.groupName)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.groupName)
		},
	}, {
		ID:    searchColLocation,
		Label: "Location",
		Width: app.ColumnWidthLocation,
		Sort: func(a, b assetRow) int {
			return strings.Compare(a.locationName, b.locationName)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.locationDisplay)
		},
	}, {
		ID:    searchColState,
		Label: "State",
		Width: 90,
		Sort: func(a, b assetRow) int {
			return strings.Compare(a.locationName, b.locationName)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.state)
		},
	}, {
		ID:    searchColQuantity,
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
		ID:    searchColTotal,
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
			ID:    searchColOwner,
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
			ID:    searchColTags,
			Label: "Tags",
			Width: app.ColumnWidthEntity,
			Update: func(r assetRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.tagsDisplay)
			},
		}})
	}
	columns := xwidget.NewDataColumns(cols)
	a := &Search{
		columnSorter:   xwidget.NewColumnSorter(columns, searchColItem, xwidget.SortAsc),
		forCorporation: forCorporation,
		footer:         awidget.NewLabelWithTruncation(""),
		search:         widget.NewEntry(),
		top:            awidget.NewLabelWithWrapping(""),
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
				ShowAssetDetailWindow(u, r)
			})
	}

	// filters
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRowsAsync(-1)
	})
	a.search.OnChanged = func(_ string) {
		a.filterRowsAsync(-1)
	}
	a.search.PlaceHolder = "Search items"
	a.selectCategory = kxwidget.NewFilterChipSelectWithSearch("Category", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectGroup = kxwidget.NewFilterChipSelectWithSearch("Group", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectOwner = kxwidget.NewFilterChipSelectWithSearch("Owner", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectRegion = kxwidget.NewFilterChipSelectWithSearch("Region", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectLocation = kxwidget.NewFilterChipSelectWithSearch("Location", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.selectState = kxwidget.NewFilterChipSelect("State", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectTotal = kxwidget.NewFilterChipSelect("Total",
		[]string{
			assetsTotalYes,
			assetsTotalNo,
		},
		func(_ string) {
			a.filterRowsAsync(-1)
		},
	)
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())

	// Signals
	if a.forCorporation {
		a.u.Signals().CurrentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			a.Update(ctx)
		})
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if a.corporation.Load().IDOrZero() != arg.CorporationID {
				return
			}
			if arg.Section != app.SectionCorporationAssets {
				return
			}
			a.Update(ctx)
		})
	} else {
		a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
			if arg.Section == app.SectionCharacterAssets {
				a.Update(ctx)
			}
		})
		a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
			a.Update(ctx)
		})
		a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			a.Update(ctx)
		})
		a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, _ struct{}) {
			a.Update(ctx)
		})
		a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
			if arg.Section == app.SectionCorporationAssets {
				a.Update(ctx)
			}
		})
	}
	a.u.Signals().EveUniverseSectionChanged.AddListener(func(ctx context.Context, arg app.EveUniverseSectionUpdated) {
		if arg.Section == app.SectionEveMarketPrices {
			a.Update(ctx)
		}
	})
	return a
}

func (a *Search) CreateRenderer() fyne.WidgetRenderer {
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
	topBox := container.NewVBox(a.top)
	if a.u.IsMobile() {
		filters.Add(a.sortButton)
		topBox.Add(a.search)
		topBox.Add(container.NewHScroll(filters))
	} else {
		topBox.Add(container.NewBorder(nil, nil, filters, nil, a.search))
	}
	c := container.NewBorder(topBox, a.footer, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *Search) makeDataList() *xwidget.StripedList {
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
				title = r.name
			} else {
				title = fmt.Sprintf("%s x%s", r.name, r.quantityDisplay)
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
		ShowAssetDetailWindow(a.u, r)
	}
	return l
}

func (a *Search) Focus() {
	a.u.MainWindow().Canvas().Focus(a.search)
}

func (a *Search) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	category := a.selectCategory.Selected
	group := a.selectGroup.Selected
	location := a.selectLocation.Selected
	owner := a.selectOwner.Selected
	region := a.selectRegion.Selected
	state := a.selectState.Selected
	tag := a.selectTag.Selected
	total := a.selectTotal.Selected
	search := strings.ToLower(a.search.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		if state != "" {
			rows = slices.DeleteFunc(rows, func(r assetRow) bool {
				return r.state != state
			})
		}
		if category != "" {
			rows = slices.DeleteFunc(rows, func(r assetRow) bool {
				return r.categoryName != category
			})
		}
		if group != "" {
			rows = slices.DeleteFunc(rows, func(r assetRow) bool {
				return r.groupName != group
			})
		}
		if owner != "" {
			rows = slices.DeleteFunc(rows, func(r assetRow) bool {
				return r.owner.Name != owner
			})
		}
		if region != "" {
			rows = slices.DeleteFunc(rows, func(r assetRow) bool {
				return r.regionName != region
			})
		}
		if location != "" {
			rows = slices.DeleteFunc(rows, func(r assetRow) bool {
				return r.locationName != location
			})
		}
		if total != "" {
			rows = slices.DeleteFunc(rows, func(r assetRow) bool {
				switch total {
				case assetsTotalYes:
					return r.total.IsEmpty()
				case assetsTotalNo:
					return !r.total.IsEmpty()
				}
				return true
			})
		}
		if tag != "" {
			rows = slices.DeleteFunc(rows, func(r assetRow) bool {
				return !r.tags.Contains(tag)
			})
		}
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
			value = optional.Sum(value, r.total)
		}
		if v, ok := value.Value(); ok {
			footer += fmt.Sprintf(" • %s ISK est. price", ihumanize.Comma(int(v)))
		}

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectCategory.SetOptions(categoryOptions)
			a.selectGroup.SetOptions(groupOptions)
			a.selectLocation.SetOptions(locationOptions)
			a.selectOwner.SetOptions(ownerOptions)
			a.selectRegion.SetOptions(regionOptions)
			a.selectState.SetOptions(stateOptions)
			a.selectTag.SetOptions(tagOptions)
			a.rowsFiltered = rows
			a.body.Refresh()
			switch x := a.body.(type) {
			case *widget.Table:
				x.ScrollToTop()
			}
		})
	}()
}

func (a *Search) Update(ctx context.Context) {
	reset := func() {
		fyne.Do(func() {
			a.rows = xslices.Reset(a.rows)
			a.filterRowsAsync(-1)
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
	if !a.forCorporation && a.characterCount() == 0 {
		reset()
		setTop("No characters", widget.LowImportance)
		return
	}
	var rows []assetRow
	var err error
	if a.forCorporation {
		rows, err = a.fetchRowsForCorporation(ctx)
	} else {
		rows, err = a.fetchRowsForAll(ctx)
	}
	if err != nil {
		slog.Error("Failed to refresh asset data", "err", err)
		reset()
		setTop("ERROR: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	fyne.Do(func() {
		a.top.Hide()
		a.rows = rows
		a.filterRowsAsync(-1)
	})
}

func (a *Search) fetchRowsForAll(ctx context.Context) ([]assetRow, error) {
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

func (a *Search) fetchRowsForCharacters(ctx context.Context) ([]assetRow, error) {
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
		r.searchTarget = strings.ToLower(r.name)
		r.tags = tagsPerCharacter[ca.CharacterID]
		r.tagsDisplay = strings.Join(slices.Sorted(r.tags.All()), ", ")
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *Search) fetchRowsForCorporations(ctx context.Context) ([]assetRow, error) {
	assets, err := a.u.Corporation().ListAllAssets(ctx)
	if err != nil {
		return nil, err
	}
	return a.fetchRowsForCorporations2(ctx, assets)
}

func (a *Search) fetchRowsForCorporation(ctx context.Context) ([]assetRow, error) {
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

func (a *Search) fetchRowsForCorporations2(ctx context.Context, assets []*app.CorporationAsset) ([]assetRow, error) {
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
		r.searchTarget = strings.ToLower(r.name)
		rows = append(rows, r)
		value += r.total.ValueOrZero()
	}
	return rows, nil
}

func (a *Search) characterCount() int {
	cc := a.u.StatusCache().ListCharacters()
	validCount := 0
	for _, c := range cc {
		if a.u.StatusCache().HasCharacterSection(c.ID, app.SectionCharacterAssets) {
			validCount++
		}
	}
	return validCount
}
