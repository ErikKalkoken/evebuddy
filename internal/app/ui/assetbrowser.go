package ui

import (
	"context"
	"fmt"
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

	"github.com/dustin/go-humanize"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/asset"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type assetFilter uint

const (
	assetNoFilter = iota
	assetCorpOther
	assetDeliveries
	assetImpounded
	assetInSpace
	assetOffice
	assetPersonalAssets
	assetSafety
)

// assetBrowser shows the attributes for the current character.
type assetBrowser struct {
	widget.BaseWidget

	Navigation *assetBrowserNavigation
	Selected   *assetBrowserContainer

	at             asset.Tree
	character      atomic.Pointer[app.Character]
	corporation    atomic.Pointer[app.Corporation]
	forCorporation bool
	u              *baseUI
}

func newCharacterAssetBrowser(u *baseUI) *assetBrowser {
	return newAssetBrowser(u, false)
}

func newCorporationAssetBrowser(u *baseUI) *assetBrowser {
	return newAssetBrowser(u, true)
}

func newAssetBrowser(u *baseUI, forCorporation bool) *assetBrowser {
	a := &assetBrowser{
		forCorporation: forCorporation,
		u:              u,
	}
	a.ExtendBaseWidget(a)

	a.Navigation = newAssetBrowserNavigation(a)
	a.Selected = newAssetBrowserContainer(a)

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
			if arg.section == app.SectionCorporationAssets {
				a.update(ctx)
			}
		})
	} else {
		a.u.currentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
			a.character.Store(c)
			a.update(ctx)
		})
		a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
			if characterIDOrZero(a.character.Load()) != arg.characterID {
				return
			}
			if arg.section == app.SectionCharacterAssets {
				a.update(ctx)
			}
		})
	}
	return a
}

func (a *assetBrowser) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewHSplit(a.Navigation, a.Selected)
	c.SetOffset(0.33)
	return widget.NewSimpleRenderer(c)
}

func (a *assetBrowser) update(ctx context.Context) {
	clear := func() {
		fyne.Do(func() {
			a.Navigation.clear()
			a.Selected.clear()
		})
	}
	setTop := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.Navigation.setTop(s, i)
		})
	}
	reportError := func(err error) {
		slog.Error("Failed to update asset browser", "error", err)
		setTop(a.u.humanizeError(err), widget.DangerImportance)
	}
	el, err := a.u.eus.ListLocations(ctx)
	if err != nil {
		reportError(err)
		return
	}
	var at asset.Tree
	if a.forCorporation {
		corporationID := corporationIDOrZero(a.corporation.Load())
		if corporationID == 0 {
			clear()
			return
		}
		hasData := a.u.scs.HasCorporationSection(corporationID, app.SectionCorporationAssets)
		if !hasData {
			clear()
			setTop("Waiting for data to be loaded...", widget.WarningImportance)
			return
		}
		assets, err := a.u.rs.ListAssets(ctx, corporationID)
		if err != nil {
			reportError(err)
			return
		}
		at = asset.NewFromCorporationAssets(assets, el)
	} else {
		characterID := characterIDOrZero(a.character.Load())
		if characterID == 0 {
			clear()
			return
		}
		hasData := a.u.scs.HasCharacterSection(characterID, app.SectionCharacterAssets)
		if !hasData {
			clear()
			setTop("Waiting for data to be loaded...", widget.WarningImportance)
			return
		}
		assets, err := a.u.cs.ListAssets(ctx, characterID)
		if err != nil {
			reportError(err)
			return
		}
		at = asset.NewFromCharacterAssets(assets, el)
	}
	fyne.DoAndWait(func() {
		a.at = at
	})
	a.Navigation.update(ctx, at.Locations())
}

const (
	assetCategoryAll        = "All"
	assetCategoryDeliveries = "Deliveries"
	assetCategoryImpounded  = "Impounded"
	assetCategoryInSpace    = "In Space"
	assetCategoryOffice     = "Office"
	assetCategoryPersonal   = "Personal"
	assetCategorySafety     = "Safety"
	assetCategoryOther      = "Other"
)

type assetContainerNode struct {
	node       *asset.Node
	itemCount  optional.Optional[int]
	searchText string
}

func (n assetContainerNode) String() string {
	if n.node == nil {
		return "?"
	}
	return n.node.String()
}

type filteredTree struct {
	td         iwidget.TreeData[assetContainerNode]
	nodeLookup map[*asset.Node]*assetContainerNode
}

type assetBrowserNavigation struct {
	widget.BaseWidget

	OnSelected func()

	ab             *assetBrowser
	collapseAll    *ttwidget.Button
	filteredTrees  map[assetFilter]filteredTree
	filters        []assetFilter
	locations      *iwidget.Tree[assetContainerNode]
	search         *widget.Entry
	selectCategory *kxwidget.FilterChipSelect
	top            *widget.Label
}

func newAssetBrowserNavigation(ab *assetBrowser) *assetBrowserNavigation {
	a := &assetBrowserNavigation{
		ab:            ab,
		filteredTrees: make(map[assetFilter]filteredTree),
		filters:       make([]assetFilter, 0),
		search:        widget.NewEntry(),
		top:           makeTopLabel(),
	}
	a.ExtendBaseWidget(a)

	a.locations = iwidget.NewTree(
		func(_ bool) fyne.CanvasObject {
			count := widget.NewLabel("99.999.999")
			name := widget.NewLabel("Template")
			name.Truncation = fyne.TextTruncateEllipsis
			return container.NewBorder(
				nil,
				nil,
				nil,
				count,
				name,
			)
		},
		func(cn *assetContainerNode, _ bool, co fyne.CanvasObject) {
			b := co.(*fyne.Container).Objects
			b[0].(*widget.Label).SetText(cn.node.String())
			var s string
			if !cn.itemCount.IsEmpty() {
				s = humanize.Comma(int64(cn.itemCount.ValueOrZero()))
			}
			b[1].(*widget.Label).SetText(s)
		},
	)
	a.locations.OnSelectedNode = func(cn *assetContainerNode) {
		a.ab.Selected.set(cn)
		if a.OnSelected != nil {
			a.OnSelected()
		}
		if ab.u.isMobile {
			a.locations.UnselectAll()
		}
	}
	if a.ab.forCorporation {
		a.filters = []assetFilter{
			assetOffice,
			assetImpounded,
			assetDeliveries,
			assetInSpace,
			assetSafety,
			assetCorpOther,
			assetNoFilter,
		}
		a.selectCategory = kxwidget.NewFilterChipSelect("", []string{
			assetCategoryOffice,
			assetCategoryImpounded,
			assetCategoryDeliveries,
			assetCategoryInSpace,
			assetCategorySafety,
			assetCategoryOther,
			assetCategoryAll,
		}, func(string) {
			a.filterLocations()
		})
		a.selectCategory.Selected = assetCategoryOffice
		a.selectCategory.SortDisabled = true
	} else {
		a.filters = []assetFilter{
			assetPersonalAssets,
			assetDeliveries,
			assetInSpace,
			assetSafety,
			assetNoFilter,
		}
		a.selectCategory = kxwidget.NewFilterChipSelect("", []string{
			assetCategoryPersonal,
			assetCategoryDeliveries,
			assetCategoryInSpace,
			assetCategorySafety,
			assetCategoryAll,
		}, func(string) {
			a.filterLocations()
		})
		a.selectCategory.Selected = assetCategoryPersonal
		a.selectCategory.SortDisabled = true
	}
	a.collapseAll = ttwidget.NewButtonWithIcon("", theme.NewThemedResource(icons.CollapseAllSvg), func() {
		a.locations.CloseAllBranches()
	})
	a.collapseAll.SetToolTip("Collapse branches")

	a.search.OnChanged = func(s string) {
		a.filterLocations()
	}
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterLocations()
	})
	a.search.PlaceHolder = "Search locations"
	return a
}

func (a *assetBrowserNavigation) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(
		container.NewBorder(
			container.NewHBox(a.selectCategory, layout.NewSpacer(), a.collapseAll),
			nil,
			nil,
			nil,
			a.search,
		),
		a.top,
		nil,
		nil,
		a.locations,
	))
}

func (a *assetBrowserNavigation) clear() {
	a.locations.Clear()
	a.locations.UnselectAll()
	a.top.SetText("")
}

func (a *assetBrowserNavigation) update(ctx context.Context, trees []*asset.Node) {
	filteredTrees := make(map[assetFilter]filteredTree)
	for _, f := range a.filters {
		td := generateTreeData(trees, f, a.ab.forCorporation)
		lookup := make(map[*asset.Node]*assetContainerNode)
		td.Walk(nil, func(n *assetContainerNode) bool {
			lookup[n.node] = n
			return true
		})
		filteredTrees[f] = filteredTree{
			td:         td,
			nodeLookup: lookup,
		}
	}
	fyne.Do(func() {
		a.filteredTrees = filteredTrees
		a.filterLocations()
	})
}

func generateTreeData(trees []*asset.Node, filter assetFilter, isCorporation bool) iwidget.TreeData[assetContainerNode] {
	var td iwidget.TreeData[assetContainerNode]

	addNodes(&td, nil, trees, filter, isCorporation)
	updateItemCounts(td)
	return td
}

func addNodes(td *iwidget.TreeData[assetContainerNode], parent *assetContainerNode, nodes []*asset.Node, filter assetFilter, isCorporation bool) {
	slices.SortFunc(nodes, func(a, b *asset.Node) int {
		return strings.Compare(a.String(), b.String())
	})

	isExcluded := func(category asset.NodeCategory) bool {
		switch filter {
		case assetCorpOther:
			switch category {
			case
				asset.NodeAssetSafetyCorporation,
				asset.NodeDeliveries,
				asset.NodeImpounded,
				asset.NodeInSpace,
				asset.NodeOfficeFolder:
				return true
			default:
				return false
			}
		case assetDeliveries:
			return category != asset.NodeDeliveries
		case assetImpounded:
			return category != asset.NodeImpounded
		case assetInSpace:
			return category != asset.NodeInSpace
		case assetOffice:
			return category != asset.NodeOfficeFolder

		case assetPersonalAssets:
			switch category {
			case asset.NodeAssetSafetyCharacter, asset.NodeDeliveries, asset.NodeInSpace:
				return true
			default:
				return false
			}
		case assetSafety:
			if isCorporation {
				return category != asset.NodeAssetSafetyCorporation
			} else {
				return category != asset.NodeAssetSafetyCharacter
			}
		}
		return false
	}

	for _, n := range nodes {
		if !n.IsContainer() {
			continue
		}
		children := n.Children()
		if n.AncestorCount() == 0 {
			var remaining int
			for _, c := range children {
				if !isExcluded(c.Category()) {
					remaining++
				}
			}
			if remaining == 0 {
				continue
			}
		}
		if n.AncestorCount() == 1 && isExcluded(n.Category()) {
			continue
		}

		var hasContainerChildren bool
		for _, n := range children {
			if n.IsContainer() {
				hasContainerChildren = true
				break
			}
		}

		cn := &assetContainerNode{
			node:       n,
			searchText: strings.ToLower(n.String()),
		}
		err := td.Add(parent, cn, hasContainerChildren)
		if err != nil {
			slog.Error("Failed to add node", "ID", n.ID(), "Name", n.String(), "error", err)
			return
		}
		if len(children) > 0 {
			addNodes(td, cn, children, filter, isCorporation)
		}
	}
}

func updateItemCounts(td iwidget.TreeData[assetContainerNode]) {
	td.Walk(nil, func(n *assetContainerNode) bool {
		if k := n.node.ChildrenCount(); k > 0 && !n.node.IsShip() {
			n.itemCount.Set(k)
		}
		return true
	})
	for _, location := range td.Children(nil) {
		location.itemCount.Clear()
		for _, top := range td.Children(location) {
			top.itemCount.Clear()
			switch top.node.Category() {
			case asset.NodeOfficeFolder, asset.NodeAssetSafetyCharacter:
				for _, n1 := range td.Children(top) {
					n1.itemCount = optional.FromIntegerWithZero(len(n1.node.Children()))
					top.itemCount = optional.Sum(top.itemCount, n1.itemCount)
				}
			case asset.NodeAssetSafetyCorporation, asset.NodeImpounded:
				for _, n1 := range td.Children(top) {
					n1.itemCount.Clear()
					for _, n2 := range td.Children(n1) {
						n2.itemCount = optional.FromIntegerWithZero(len(n2.node.Children()))
						n1.itemCount = optional.Sum(n1.itemCount, n2.itemCount)
					}
					top.itemCount = optional.Sum(top.itemCount, n1.itemCount)
				}
			default:
				top.itemCount = optional.FromIntegerWithZero(len(top.node.Children()))
			}
			location.itemCount = optional.Sum(location.itemCount, top.itemCount)
		}
	}
}

var assetFilterLookup = map[string]assetFilter{
	assetCategoryAll:        assetNoFilter,
	assetCategoryDeliveries: assetDeliveries,
	assetCategoryImpounded:  assetImpounded,
	assetCategoryInSpace:    assetInSpace,
	assetCategoryOffice:     assetOffice,
	assetCategoryOther:      assetCorpOther,
	assetCategoryPersonal:   assetPersonalAssets,
	assetCategorySafety:     assetSafety,
}

func (a *assetBrowserNavigation) filterLocations() {
	filter := assetFilterLookup[a.selectCategory.Selected]
	var td iwidget.TreeData[assetContainerNode]
	ft := a.filteredTrees[filter]
	count, _ := ft.td.ChildrenCount(nil)
	top := fmt.Sprintf("%d locations", count)
	search := strings.ToLower(a.search.Text)

	go func() {
		if len(search) > 1 {
			td = ft.td.Clone()
			td.DeleteChildrenFunc(nil, func(n *assetContainerNode) bool {
				return !strings.Contains(n.searchText, search)
			})
		} else {
			td = ft.td
		}
		fyne.Do(func() {
			a.locations.UnselectAll()
			a.locations.CloseAllBranches()
			a.locations.Set(td)
			a.ab.Selected.clear()
			a.setTop(top, widget.MediumImportance)
		})
	}()
}

func (a *assetBrowserNavigation) nodeLookup(n *asset.Node) (*assetContainerNode, bool) {
	filter := assetFilterLookup[a.selectCategory.Selected]
	ft, ok := a.filteredTrees[filter]
	if !ok {
		ft = a.filteredTrees[assetNoFilter]
	}
	cn, ok := ft.nodeLookup[n]
	if !ok {
		return nil, false
	}
	return cn, true
}

func (a *assetBrowserNavigation) setTop(s string, i widget.Importance) {
	a.top.Text = s
	a.top.Importance = i
	a.top.Refresh()
}

func (a *assetBrowserNavigation) selectContainer(cn *assetContainerNode) {
	a.locations.UnselectAll()
	if !a.ab.u.isMobile {
		a.locations.SelectNode(cn)
	}
	for _, cn2 := range a.locations.Data().Path(nil, cn) {
		a.locations.OpenBranchNode(cn2)
	}
	a.locations.ScrollToNode(cn)
}

type containerItem struct {
	node       *asset.Node
	searchText string
}

type assetBrowserContainer struct {
	widget.BaseWidget

	ab            *assetBrowser
	bottom        *widget.Label
	grid          *widget.GridWrap
	items         []containerItem
	itemsFiltered []containerItem
	location      *assetBrowserLocation
	search        *widget.Entry
}

func newAssetBrowserContainer(ab *assetBrowser) *assetBrowserContainer {
	a := &assetBrowserContainer{
		ab:            ab,
		bottom:        widget.NewLabel(""),
		items:         make([]containerItem, 0),
		itemsFiltered: make([]containerItem, 0),
		search:        widget.NewEntry(),
	}
	a.ExtendBaseWidget(a)
	a.grid = a.makeAssetGrid()
	a.location = newAssetBrowserLocation(a)

	a.search.OnChanged = func(s string) {
		a.filterItems()
	}
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterItems()
	})
	a.search.PlaceHolder = "Search items"
	a.search.Hide()
	return a
}

func (a *assetBrowserContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(
		container.NewBorder(
			a.location,
			nil,
			nil,
			nil,
			a.search,
		),
		a.bottom,
		nil,
		nil,
		a.grid,
	))
}

func (a *assetBrowserContainer) makeAssetGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.itemsFiltered)
		},
		func() fyne.CanvasObject {
			return newAssetIcon(a.ab.u.eis)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.itemsFiltered) {
				return
			}
			it := a.itemsFiltered[id]
			item := co.(*assetItem)
			item.Set(it.node)
		},
	)
	g.OnSelected = func(id widget.GridWrapItemID) {
		defer g.UnselectAll()
		if id >= len(a.itemsFiltered) {
			return
		}
		it := a.itemsFiltered[id]
		n := it.node
		if n.IsContainer() {
			cn, ok := a.ab.Navigation.nodeLookup(n)
			if !ok {
				return
			}
			a.ab.Navigation.selectContainer(cn)
			a.set(cn)
		} else {
			a.showNodeInfo(n)
		}
	}
	return g
}

func (a *assetBrowserContainer) set(cn *assetContainerNode) {
	var nodes []*asset.Node
	if cn.node.AncestorCount() == 0 {
		// ensuring the location container shows the same items like the nav tree
		children := a.ab.Navigation.locations.Data().Children(cn)
		for _, n := range children {
			nodes = append(nodes, n.node)
		}
	} else {
		nodes = cn.node.Children()
	}
	go func() {
		items := make([]containerItem, 0)
		for _, n := range nodes {
			var s string
			if an, ok := n.Asset(); ok {
				s = an.DisplayName2()
			} else if el, ok := n.Location(); ok {
				s = el.DisplayName()
			} else {
				s = n.String()
			}
			items = append(items, containerItem{
				node:       n,
				searchText: strings.ToLower(s),
			})
		}
		fyne.Do(func() {
			a.items = items
			a.location.set(cn)
			a.search.Show()
			a.filterItems()
		})
	}()
}

func (a *assetBrowserContainer) filterItems() {
	items := slices.Clone(a.items)
	search := strings.ToLower(a.search.Text)
	go func() {
		if len(search) > 1 {
			items = slices.DeleteFunc(items, func(ci containerItem) bool {
				return !strings.Contains(ci.searchText, search)
			})
		}
		sortName := func(it containerItem) string {
			n := it.node
			switch n.Category() {
			case asset.NodeLocation:
				el, ok := n.Location()
				if !ok {
					return "?"
				}
				return el.DisplayName()
			case asset.NodeAsset:
				n, ok := n.Asset()
				if !ok {
					return "?"
				}
				return fmt.Sprintf("%s-%s", n.TypeName(), n.Name)
			}
			return n.Category().String()
		}
		slices.SortFunc(items, func(a, b containerItem) int {
			return strings.Compare(sortName(a), sortName(b))
		})

		var itemCount int64
		var value optional.Optional[float64]
		for _, it := range items {
			itemCount++
			if as, ok := it.node.Asset(); ok {
				value = optional.Sum(value, optional.New(as.Price.ValueOrZero()*float64(as.Quantity)))
			}
		}
		bottom := fmt.Sprintf("%s items", humanize.Comma(itemCount))
		if !value.IsEmpty() {
			bottom += fmt.Sprintf(" • %s ISK est. price", ihumanize.Comma(int(value.ValueOrZero())))
		}

		fyne.Do(func() {
			a.itemsFiltered = items
			a.grid.Refresh()
			a.bottom.SetText(bottom)
		})
	}()
}

func (a *assetBrowserContainer) clear() {
	a.location.clear()
	a.search.Hide()
	a.bottom.SetText("")
	a.items = make([]containerItem, 0)
	a.filterItems()
}

func (a *assetBrowserContainer) showNodeInfo(n *asset.Node) {
	if a.ab.forCorporation {
		ca, ok := n.CorporationAsset()
		if !ok {
			return
		}
		name := corporationNameOrZero(a.ab.corporation.Load())
		showAssetDetailWindow(a.ab.u, newCorporationAssetRow(ca, a.ab.at, name))
		return
	}
	ca, ok := n.CharacterAsset()
	if !ok {
		return
	}
	showAssetDetailWindow(a.ab.u, newCharacterAssetRow(ca, a.ab.at, a.ab.u.scs.CharacterName))
}

type assetBrowserLocation struct {
	widget.BaseWidget

	breadcrumbs *fyne.Container
	info        *iwidget.TappableIcon
	selected    *assetBrowserContainer
}

func newAssetBrowserLocation(selected *assetBrowserContainer) *assetBrowserLocation {
	a := &assetBrowserLocation{
		breadcrumbs: container.New(layout.NewRowWrapLayoutWithCustomPadding(0, 0)),
		info:        iwidget.NewTappableIcon(theme.InfoIcon(), nil),
		selected:    selected,
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *assetBrowserLocation) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, nil, a.info, a.breadcrumbs)
	return widget.NewSimpleRenderer(c)
}

func (a *assetBrowserLocation) clear() {
	a.breadcrumbs.RemoveAll()
	a.info.Hide()
}

func (a *assetBrowserLocation) set(cn *assetContainerNode) {
	nodeName := func(n *asset.Node) string {
		if an, ok := n.Asset(); ok {
			return an.DisplayName3()
		}
		return n.String()
	}
	a.breadcrumbs.RemoveAll()
	p := theme.Padding()

	node := cn.node
	if path := node.Path(); len(path) > 0 {
		for _, n := range path[:len(path)-1] {
			l := widget.NewHyperlink(nodeName(n), nil)
			l.OnTapped = func() {
				cn, ok := a.selected.ab.Navigation.nodeLookup(n)
				if !ok {
					return
				}
				a.selected.ab.Navigation.selectContainer(cn)
				a.selected.set(cn)
			}
			a.breadcrumbs.Add(l)
			x := container.New(layout.NewCustomPaddedLayout(0, 0, -2*p, -2*p), widget.NewLabel("＞"))
			a.breadcrumbs.Add(x)
		}
	}

	a.breadcrumbs.Add(widget.NewLabel(nodeName(node)))

	switch node.Category() {
	case asset.NodeLocation:
		el, ok := node.Location()
		if !ok {
			return
		}
		if el.Variant() == app.EveLocationUnknown {
			return
		}
		a.info.OnTapped = func() {
			a.selected.ab.u.ShowLocationInfoWindow(el.ID)
		}
		a.info.Show()
	case asset.NodeAsset:
		a.info.OnTapped = func() {
			a.selected.showNodeInfo(node)
		}
		a.info.Show()
	default:
		a.info.Hide()
	}
}

const (
	colorAssetQuantityBadgeBackground = theme.ColorNameMenuBackground
	labelMaxCharacters                = 10
	sizeLabelText                     = 12
	typeIconSize                      = 55
)

// assetItem represents an asset shown with an icon and label.
type assetItem struct {
	widget.BaseWidget

	badge *assetQuantityBadge
	eis   assetIconEIS
	icon  *canvas.Image
	label *assetLabel
}

func newAssetIcon(eis assetIconEIS) *assetItem {
	icon := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(typeIconSize))
	w := &assetItem{
		icon:  icon,
		label: newAssetLabel(),
		eis:   eis,
		badge: newAssetQuantityBadge(),
	}
	w.badge.Hide()
	w.ExtendBaseWidget(w)
	return w
}

func (w *assetItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewPadded(container.New(layout.NewCustomPaddedVBoxLayout(0),
		container.New(&bottomRightLayout{}, w.icon, w.badge),
		w.label,
	))
	return widget.NewSimpleRenderer(c)
}

func (w *assetItem) Set(n *asset.Node) {
	defer w.Refresh()
	showFolder := func(w *assetItem) {
		fyne.Do(func() {
			w.icon.Resource = theme.FolderIcon()
			w.icon.Refresh()
			w.badge.Hide()
		})
	}
	as, ok := n.Asset()
	if !ok {
		w.label.SetText(n.String())
		showFolder(w)
		return
	}
	var name string
	if as.Name != "" {
		name = as.Name
	} else {
		name = as.TypeName()
	}
	w.label.SetText(name)
	if n.Category() == asset.NodeOfficeFolder {
		showFolder(w)
		return
	}
	if !as.IsSingleton {
		w.badge.SetQuantity(int(as.Quantity))
		w.badge.Show()
	} else {
		w.badge.Hide()
	}

	loadAssetIconAsync(w.eis, w.icon, as.Type.ID, as.Variant())
}

type assetLabel struct {
	widget.BaseWidget

	label1 *canvas.Text
	label2 *canvas.Text
}

func newAssetLabel() *assetLabel {
	l1 := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	l1.TextSize = theme.CaptionTextSize()
	l2 := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	l2.TextSize = l1.TextSize
	w := &assetLabel{label1: l1, label2: l2}
	w.ExtendBaseWidget(w)
	return w
}

func (w *assetLabel) SetText(s string) {
	l1, l2 := splitLines(s, labelMaxCharacters)
	w.label1.Text = l1
	w.label2.Text = l2
	w.label1.Refresh()
	w.label2.Refresh()
}

func (w *assetLabel) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.label1.Color = th.Color(theme.ColorNameForeground, v)
	w.label1.Refresh()
	w.label2.Color = th.Color(theme.ColorNameForeground, v)
	w.label2.Refresh()
	w.BaseWidget.Refresh()
}

func (w *assetLabel) CreateRenderer() fyne.WidgetRenderer {
	customVBox := layout.NewCustomPaddedVBoxLayout(0)
	customHBox := layout.NewCustomPaddedHBoxLayout(0)
	c := container.New(
		customVBox,
		container.New(customHBox, layout.NewSpacer(), w.label1, layout.NewSpacer()),
		container.New(customHBox, layout.NewSpacer(), w.label2, layout.NewSpacer()),
	)
	return widget.NewSimpleRenderer(c)
}

// splitLines will split a strings into 2 lines while ensuring no line is longer then maxLine characters.
//
// When possible it will wrap on spaces.
func splitLines(s string, maxLine int) (string, string) {
	if len(s) < maxLine {
		return s, ""
	}
	if len(s) > 2*maxLine {
		s = s[:2*maxLine]
	}
	ll := make([]string, 2)
	p := strings.Split(s, " ")
	if len(p) == 1 {
		// wrapping on spaces failed
		ll[0] = s[:min(len(s), maxLine)]
		if len(s) > maxLine {
			ll[1] = s[maxLine:min(len(s), 2*maxLine)]
		}
		return ll[0], ll[1]
	}
	var l int
	ll[l] = p[0]
	for _, x := range p[1:] {
		if len(ll[l]+x)+1 > maxLine {
			if l == 1 {
				remaining := max(0, maxLine-len(ll[l])-1)
				if remaining > 0 {
					ll[l] += " " + x[:remaining]
				}
				break
			}
			l++
			ll[l] += x
			continue
		}
		ll[l] += " " + x
	}
	return ll[0], ll[1]
}

type assetQuantityBadge struct {
	widget.BaseWidget

	quantity *canvas.Text
	bg       *canvas.Rectangle
}

func newAssetQuantityBadge() *assetQuantityBadge {
	q := canvas.NewText("", theme.Color(theme.ColorNameForeground))
	q.TextSize = sizeLabelText
	w := &assetQuantityBadge{
		quantity: q,
		bg:       canvas.NewRectangle(theme.Color(colorAssetQuantityBadgeBackground)),
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *assetQuantityBadge) SetQuantity(q int) {
	w.quantity.Text = humanize.Comma(int64(q))
	w.quantity.Refresh()
}

func (w *assetQuantityBadge) Refresh() {
	th := w.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	w.quantity.Color = th.Color(theme.ColorNameForeground, v)
	w.quantity.Refresh()
	w.bg.FillColor = th.Color(colorAssetQuantityBadgeBackground, v)
	w.bg.Refresh()
	w.BaseWidget.Refresh()
}

func (w *assetQuantityBadge) CreateRenderer() fyne.WidgetRenderer {
	p := theme.Padding()
	bgPadding := layout.NewCustomPaddedLayout(0, 0, p, p)
	customPadding := layout.NewCustomPaddedLayout(p/2, p/2, p/2, p/2)
	c := container.New(customPadding, container.NewStack(
		w.bg,
		container.New(bgPadding, w.quantity),
	))
	return widget.NewSimpleRenderer(c)
}
