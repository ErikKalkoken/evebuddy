package ui

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
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xsync"
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
		r.locationDisplay = iwidget.RichTextSegmentsFromText("?")
		return
	}
	el, ok := ln.Location()
	if !ok {
		r.locationDisplay = iwidget.RichTextSegmentsFromText("?")
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

type assetSearch struct {
	widget.BaseWidget

	body           fyne.CanvasObject
	columnSorter   *iwidget.ColumnSorter[assetRow]
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
	sortButton     *iwidget.SortButton
	top            *widget.Label
	u              *baseUI
}

const (
	assetsColItem = iota + 1
	assetsColGroup
	assetsColLocation
	assetsColState
	assetsColQuantity
	assetsColTotal
	assetsColOwner
	assetsColTags
)

func newCombinedAssetSearch(u *baseUI) *assetSearch {
	return newAssetSearch(u, false)
}

func newAssetSearchForCorporation(u *baseUI) *assetSearch {
	return newAssetSearch(u, true)
}

func newAssetSearch(u *baseUI, forCorporation bool) *assetSearch {
	corporationIcon := theme.NewThemedResource(icons.StarCircleOutlineSvg)
	cols := []iwidget.DataColumn[assetRow]{{
		ID:    assetsColItem,
		Label: "Item",
		Width: 300,
		Sort: func(a, b assetRow) int {
			return strings.Compare(a.name, b.name)
		},
		Create: func() fyne.CanvasObject {
			icon := iwidget.NewImageFromResource(
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
			loadAssetIconAsync(u.eis, x, r.typeID, r.variant)
		},
	}, {
		ID:    assetsColGroup,
		Label: "Group",
		Width: 200,
		Sort: func(a, b assetRow) int {
			return strings.Compare(a.groupName, b.groupName)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).SetWithText(r.groupName)
		},
	}, {
		ID:    assetsColLocation,
		Label: "Location",
		Width: columnWidthLocation,
		Sort: func(a, b assetRow) int {
			return strings.Compare(a.locationName, b.locationName)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).Set(r.locationDisplay)
		},
	}, {
		ID:    assetsColState,
		Label: "State",
		Width: 90,
		Sort: func(a, b assetRow) int {
			return strings.Compare(a.locationName, b.locationName)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).SetWithText(r.state)
		},
	}, {
		ID:    assetsColQuantity,
		Label: "Qty.",
		Width: 100,
		Sort: func(a, b assetRow) int {
			return cmp.Compare(a.quantity, b.quantity)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).SetWithText(r.quantityDisplay, widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			})
		},
	}, {
		ID:    assetsColTotal,
		Label: "Total",
		Width: 150,
		Sort: func(a, b assetRow) int {
			return optional.Compare(a.total, b.total)
		},
		Update: func(r assetRow, co fyne.CanvasObject) {
			co.(*iwidget.RichText).SetWithText(r.totalDisplay, widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			})
		},
	}}
	if !forCorporation {
		cols = slices.Concat(cols, []iwidget.DataColumn[assetRow]{{
			ID:    assetsColOwner,
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
			ID:    assetsColTags,
			Label: "Tags",
			Width: columnWidthEntity,
			Update: func(r assetRow, co fyne.CanvasObject) {
				co.(*iwidget.RichText).SetWithText(r.tagsDisplay)
			},
		}})
	}
	columns := iwidget.NewDataColumns(cols)
	a := &assetSearch{
		columnSorter:   iwidget.NewColumnSorter(columns, assetsColItem, iwidget.SortAsc),
		forCorporation: forCorporation,
		footer:         newLabelWithTruncation(),
		rowsFiltered:   make([]assetRow, 0),
		search:         widget.NewEntry(),
		top:            newLabelWithWrapping(),
		u:              u,
	}
	a.ExtendBaseWidget(a)

	if a.u.isMobile {
		a.body = a.makeDataList()
	} else {
		a.body = iwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := iwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter, a.filterRowsAsync, func(_ int, r assetRow) {
				showAssetDetailWindow(u, r)
			})
	}

	// filters
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRowsAsync(-1)
	})
	a.search.OnChanged = func(s string) {
		a.filterRowsAsync(-1)
	}
	a.search.PlaceHolder = "Search items"
	a.selectCategory = kxwidget.NewFilterChipSelectWithSearch("Category", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.window)
	a.selectGroup = kxwidget.NewFilterChipSelectWithSearch("Group", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.window)
	a.selectOwner = kxwidget.NewFilterChipSelectWithSearch("Owner", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.window)
	a.selectRegion = kxwidget.NewFilterChipSelectWithSearch("Region", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.window)
	a.selectLocation = kxwidget.NewFilterChipSelectWithSearch("Location", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.window)
	a.selectState = kxwidget.NewFilterChipSelect("State", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.selectTotal = kxwidget.NewFilterChipSelect("Total",
		[]string{
			assetsTotalYes,
			assetsTotalNo,
		},
		func(s string) {
			a.filterRowsAsync(-1)
		},
	)
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync(-1)
	}, a.u.window)

	// Signals
	if a.forCorporation {
		a.u.currentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			a.update(ctx)
		})
		a.u.corporationSectionChanged.AddListener(func(ctx context.Context, arg corporationSectionUpdated) {
			if corporationIDOrZero(a.corporation.Load()) != arg.corporationID {
				return
			}
			if arg.section != app.SectionCorporationAssets {
				return
			}
			a.update(ctx)
		})
	} else {
		a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
			if arg.section == app.SectionCharacterAssets {
				a.update(ctx)
			}
		})
		a.u.characterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
			a.update(ctx)
		})
		a.u.characterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			a.update(ctx)
		})
		a.u.tagsChanged.AddListener(func(ctx context.Context, s struct{}) {
			a.update(ctx)
		})
		a.u.corporationSectionChanged.AddListener(func(ctx context.Context, arg corporationSectionUpdated) {
			if arg.section == app.SectionCorporationAssets {
				a.update(ctx)
			}
		})
	}
	a.u.generalSectionChanged.AddListener(func(ctx context.Context, arg generalSectionUpdated) {
		if arg.section == app.SectionEveMarketPrices {
			a.update(ctx)
		}
	})
	return a
}

func (a *assetSearch) CreateRenderer() fyne.WidgetRenderer {
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
	if a.u.isMobile {
		filters.Add(a.sortButton)
		topBox.Add(a.search)
		topBox.Add(container.NewHScroll(filters))
	} else {
		topBox.Add(container.NewBorder(nil, nil, filters, nil, a.search))
	}
	c := container.NewBorder(topBox, a.footer, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *assetSearch) makeDataList() *iwidget.StripedList {
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
				title = r.name
			} else {
				title = fmt.Sprintf("%s x%s", r.name, r.quantityDisplay)
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

func (a *assetSearch) focus() {
	a.u.MainWindow().Canvas().Focus(a.search)
}

func (a *assetSearch) filterRowsAsync(sortCol int) {
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

func (a *assetSearch) update(ctx context.Context) {
	clear := func() {
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
		clear()
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
		clear()
		setTop("ERROR: "+a.u.humanizeError(err), widget.DangerImportance)
		return
	}
	fyne.Do(func() {
		a.top.Hide()
		a.rows = rows
		a.filterRowsAsync(-1)
	})
}

func (a *assetSearch) fetchRowsForAll(ctx context.Context) ([]assetRow, error) {
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

func (a *assetSearch) fetchRowsForCharacters(ctx context.Context) ([]assetRow, error) {
	cc, err := a.u.cs.ListCharactersShort(ctx)
	if err != nil {
		return nil, err
	}
	if len(cc) == 0 {
		return nil, nil
	}
	characterNames := make(map[int64]string)
	for _, o := range cc {
		characterNames[o.ID] = o.Name
	}
	tagsPerCharacter := make(map[int64]set.Set[string])
	for _, c := range cc {
		tags, err := a.u.cs.ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			return nil, nil
		}
		tagsPerCharacter[c.ID] = tags
	}
	assets, err := a.u.cs.ListAllAssets(ctx)
	if err != nil {
		return nil, err
	}
	locations, err := a.u.eus.ListLocations(ctx)
	if err != nil {
		return nil, err
	}
	ac := asset.NewFromCharacterAssets(assets, locations)
	var rows []assetRow
	for _, ca := range assets {
		r := newCharacterAssetRow(ca, ac, func(id int64) string {
			return characterNames[id]
		})
		r.searchTarget = strings.ToLower(r.name)
		r.tags = tagsPerCharacter[ca.CharacterID]
		r.tagsDisplay = strings.Join(slices.Sorted(r.tags.All()), ", ")
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *assetSearch) fetchRowsForCorporations(ctx context.Context) ([]assetRow, error) {
	assets, err := a.u.rs.ListAllAssets(ctx)
	if err != nil {
		return nil, err
	}
	return a.fetchRowsForCorporations2(ctx, assets)
}

func (a *assetSearch) fetchRowsForCorporation(ctx context.Context) ([]assetRow, error) {
	c := a.corporation.Load()
	if c == nil {
		return []assetRow{}, nil
	}
	assets, err := a.u.rs.ListAssets(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	return a.fetchRowsForCorporations2(ctx, assets)
}

func (a *assetSearch) fetchRowsForCorporations2(ctx context.Context, assets []*app.CorporationAsset) ([]assetRow, error) {
	cc, err := a.u.rs.ListCorporationsShort(ctx)
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
	locations, err := a.u.eus.ListLocations(ctx)
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

func (a *assetSearch) characterCount() int {
	cc := a.u.scs.ListCharacters()
	validCount := 0
	for _, c := range cc {
		if a.u.scs.HasCharacterSection(c.ID, app.SectionCharacterAssets) {
			validCount++
		}
	}
	return validCount
}

type assetIconEIS interface {
	InventoryTypeBPC(id int64, size int) (fyne.Resource, error)
	InventoryTypeBPO(id int64, size int) (fyne.Resource, error)
	InventoryTypeIcon(id int64, size int) (fyne.Resource, error)
	InventoryTypeSKIN(id int64, size int) (fyne.Resource, error)
}

// assetIconCache caches the images for asset icons.
var assetIconCache xsync.Map[string, fyne.Resource]

func loadAssetIconAsync(eis assetIconEIS, icon *canvas.Image, typeID int64, variant app.InventoryTypeVariant) {
	key := fmt.Sprintf("%d-%d", typeID, variant)
	iwidget.LoadResourceAsyncWithCache(
		icons.BlankSvg,
		func() (fyne.Resource, bool) {
			return assetIconCache.Load(key)
		},
		func(r fyne.Resource) {
			icon.Resource = r
			icon.Refresh()
		},
		func() (fyne.Resource, error) {
			switch variant {
			case app.VariantBPO:
				return eis.InventoryTypeBPO(typeID, app.IconPixelSize)
			case app.VariantBPC:
				return eis.InventoryTypeBPC(typeID, app.IconPixelSize)
			case app.VariantSKIN:
				return eis.InventoryTypeSKIN(typeID, app.IconPixelSize)
			default:
				return eis.InventoryTypeIcon(typeID, app.IconPixelSize)
			}
		},
		func(r fyne.Resource) {
			assetIconCache.Store(key, r)
		},
	)
}

// showAssetDetailWindow shows the details for an assets in a new window.
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
	item := makeLinkLabelWithWrap(r.typeName, func() {
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

	var p string
	if len(r.locationPath) > 0 {
		p = strings.Join(r.locationPath, " / ")
	} else {
		p = "-"
	}
	path := widget.NewLabel(p)
	path.Wrapping = fyne.TextWrapWord

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
		widget.NewFormItem("Path", path),
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
		items = slices.Concat(items, []*widget.FormItem{
			widget.NewFormItem("Location Flag", widget.NewLabel(r.locationFlag.String())),
			widget.NewFormItem("Item ID", u.makeCopyToClipboardLabel(fmt.Sprint(r.itemID))),
		})
	}

	f := widget.NewForm(items...)
	f.Orientation = widget.Adaptive
	setDetailWindow(detailWindowParams{
		content: f,
		imageAction: func() {
			u.ShowTypeInfoWindow(r.typeID)
		},
		imageLoader: func(setter func(r fyne.Resource)) {
			switch r.variant {
			case app.VariantSKIN:
				u.eis.InventoryTypeSKINAsync(r.typeID, app.IconPixelSize, setter)
			case app.VariantBPO:
				u.eis.InventoryTypeBPOAsync(r.typeID, app.IconPixelSize, setter)
			case app.VariantBPC:
				u.eis.InventoryTypeBPCAsync(r.typeID, app.IconPixelSize, setter)
			default:
				u.eis.InventoryTypeIconAsync(r.typeID, app.IconPixelSize, setter)
			}
		},
		minSize: fyne.NewSize(500, 450),
		title:   r.name,
		window:  w,
	})
	w.Show()
}
