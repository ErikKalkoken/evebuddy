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
	Selected   *assetBrowserSelected

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
	a.Selected = newAssetBrowserSelected(a)

	// Signals
	if a.forCorporation {
		a.u.currentCorporationExchanged.AddListener(func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			a.updateAsync()
		})
		a.u.corporationSectionChanged.AddListener(func(ctx context.Context, arg corporationSectionUpdated) {
			if corporationIDOrZero(a.corporation.Load()) != arg.corporationID {
				return
			}
			if arg.section == app.SectionCorporationAssets {
				a.updateAsync()
			}
		})
	} else {
		a.u.currentCharacterExchanged.AddListener(func(_ context.Context, c *app.Character) {
			a.character.Store(c)
			a.updateAsync()
		})
		a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
			if characterIDOrZero(a.character.Load()) != arg.characterID {
				return
			}
			if arg.section == app.SectionCharacterAssets {
				a.updateAsync()
			}
		})
	}
	return a
}

func (a *assetBrowser) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewHSplit(a.Navigation, a.Selected)
	c.SetOffset(0.4)
	return widget.NewSimpleRenderer(c)
}

func (a *assetBrowser) updateAsync() {
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
	ctx := context.Background()
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
	a.Navigation.updateAsync(at.Locations())
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
	node      *asset.Node
	itemCount optional.Optional[int]
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
	navigation     *iwidget.Tree[assetContainerNode]
	selectCategory *kxwidget.FilterChipSelect
	filteredTrees  map[assetFilter]filteredTree
	filters        []assetFilter
	top            *widget.Label
}

func newAssetBrowserNavigation(ab *assetBrowser) *assetBrowserNavigation {
	a := &assetBrowserNavigation{
		ab:            ab,
		filteredTrees: make(map[assetFilter]filteredTree),
		top:           makeTopLabel(),
		filters:       make([]assetFilter, 0),
	}
	a.ExtendBaseWidget(a)

	a.navigation = iwidget.NewTree(
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
	a.navigation.OnSelectedNode = func(cn *assetContainerNode) {
		a.ab.Selected.set(cn)
		if a.OnSelected != nil {
			a.OnSelected()
		}
		if ab.u.isMobile {
			a.navigation.UnselectAll()
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
			a.redraw()
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
			a.redraw()
		})
		a.selectCategory.Selected = assetCategoryPersonal
		a.selectCategory.SortDisabled = true
	}
	return a
}

func (a *assetBrowserNavigation) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(
		a.selectCategory,
		a.top,
		nil,
		nil,
		a.navigation,
	))
}

func (a *assetBrowserNavigation) clear() {
	a.navigation.Clear()
	a.navigation.UnselectAll()
	a.top.SetText("")
}

func (a *assetBrowserNavigation) updateAsync(trees []*asset.Node) {
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
		a.redraw()
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
		cn := &assetContainerNode{
			node: n,
		}
		var hasContainerChildren bool
		for _, n := range children {
			if n.IsContainer() {
				hasContainerChildren = true
				break
			}
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
		n.itemCount.Clear()
		return true
	})
	for _, location := range td.Children(nil) {
		for _, top := range td.Children(location) {
			switch top.node.Category() {
			case asset.NodeOfficeFolder, asset.NodeAssetSafetyCharacter:
				for _, n1 := range td.Children(top) {
					n1.itemCount = optional.FromIntegerWithZero(len(n1.node.Children()))
					top.itemCount = optional.Sum(top.itemCount, n1.itemCount)
				}
			case asset.NodeAssetSafetyCorporation, asset.NodeImpounded:
				for _, n1 := range td.Children(top) {
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

func (a *assetBrowserNavigation) redraw() {
	filter := assetFilterLookup[a.selectCategory.Selected]
	ft := a.filteredTrees[filter]
	a.navigation.UnselectAll()
	a.navigation.CloseAllBranches()
	a.navigation.Set(ft.td)
	a.ab.Selected.clear()
	count, _ := ft.td.ChildrenCount(nil)
	top := fmt.Sprintf("%d locations", count)
	a.setTop(top, widget.MediumImportance)
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
	a.navigation.UnselectAll()
	if !a.ab.u.isMobile {
		a.navigation.SelectNode(cn)
	}
	for _, cn2 := range a.navigation.Data().Path(nil, cn) {
		a.navigation.OpenBranchNode(cn2)
	}
	a.navigation.ScrollToNode(cn)
}

type assetBrowserSelected struct {
	widget.BaseWidget

	ab       *assetBrowser
	bottom   *widget.Label
	children []*asset.Node
	grid     *widget.GridWrap
	node     *asset.Node
	location *assetBrowserLocation
}

func newAssetBrowserSelected(ab *assetBrowser) *assetBrowserSelected {
	a := &assetBrowserSelected{
		ab:     ab,
		bottom: widget.NewLabel(""),
	}
	a.ExtendBaseWidget(a)
	a.grid = a.makeAssetGrid()
	a.location = newAssetBrowserLocation(a)
	return a
}

func (a *assetBrowserSelected) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(
		a.location,
		a.bottom,
		nil,
		nil,
		a.grid,
	))
}

func (a *assetBrowserSelected) makeAssetGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.children)
		},
		func() fyne.CanvasObject {
			return newAssetNodeIcon(a.ab.u.eis)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.children) {
				return
			}
			n := a.children[id]
			item := co.(*assetNodeIcon)
			item.Set(n)
		},
	)
	g.OnSelected = func(id widget.GridWrapItemID) {
		defer g.UnselectAll()
		if id >= len(a.children) {
			return
		}
		n := a.children[id]
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

func (a *assetBrowserSelected) showNodeInfo(n *asset.Node) {
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

func (a *assetBrowserSelected) clear() {
	a.node = nil
	a.children = make([]*asset.Node, 0)
	a.grid.Refresh()
	a.location.clear()
	a.bottom.SetText("")
}

func (a *assetBrowserSelected) set(cn *assetContainerNode) {
	a.node = cn.node
	var nodes []*asset.Node
	if cn.node.AncestorCount() == 0 {
		// ensuring the location container shows the same items like the nav tree
		for _, n := range a.ab.Navigation.navigation.Data().Children(cn) {
			nodes = append(nodes, n.node)
		}
	} else {
		nodes = cn.node.Children()
	}

	sortName := func(n *asset.Node) string {
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
		return n.Category().DisplayName()
	}
	slices.SortFunc(nodes, func(a, b *asset.Node) int {
		return strings.Compare(sortName(a), sortName(b))
	})

	a.children = nodes
	a.grid.Refresh()
	a.location.set(cn)

	var itemCount int64
	var value float64
	for _, n := range nodes {
		if as, ok := n.Asset(); ok {
			itemCount++
			value += as.Price.ValueOrZero() * float64(as.Quantity)
		}
	}
	var s string
	if itemCount > 0 {
		s = fmt.Sprintf("%s Items - %s ISK Est. Price", humanize.Comma(itemCount), ihumanize.Comma(int(value)))
	}
	a.bottom.SetText(s)
}

type assetBrowserLocation struct {
	widget.BaseWidget

	breadcrumbs *fyne.Container
	info        *iwidget.TappableIcon
	selected    *assetBrowserSelected
}

func newAssetBrowserLocation(selected *assetBrowserSelected) *assetBrowserLocation {
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
	a.breadcrumbs.RemoveAll()
	p := theme.Padding()
	if path := cn.node.Path(); len(path) > 0 {
		for _, n := range path[:len(path)-1] {
			l := widget.NewHyperlink(n.String(), nil)
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
	a.breadcrumbs.Add(widget.NewLabel(cn.node.String()))

	switch cn.node.Category() {
	case asset.NodeLocation:
		el, ok := cn.node.Location()
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
			a.selected.showNodeInfo(cn.node)
		}
		a.info.Show()
	default:
		a.info.Hide()
	}
}

const (
	typeIconSize                      = 55
	sizeLabelText                     = 12
	colorAssetQuantityBadgeBackground = theme.ColorNameMenuBackground
	labelMaxCharacters                = 10
)

type assetNodeIconEIS interface {
	InventoryTypeBPC(id int32, size int) (fyne.Resource, error)
	InventoryTypeBPO(id int32, size int) (fyne.Resource, error)
	InventoryTypeIcon(id int32, size int) (fyne.Resource, error)
	InventoryTypeSKIN(id int32, size int) (fyne.Resource, error)
}

type assetNodeIcon struct {
	widget.BaseWidget

	badge *assetQuantityBadge
	icon  *canvas.Image
	eis   assetNodeIconEIS
	label *assetLabel
}

func newAssetNodeIcon(eis assetNodeIconEIS) *assetNodeIcon {
	icon := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(typeIconSize))
	w := &assetNodeIcon{
		icon:  icon,
		label: newAssetLabel(),
		eis:   eis,
		badge: newAssetQuantityBadge(),
	}
	w.badge.Hide()
	w.ExtendBaseWidget(w)
	return w
}

func (w *assetNodeIcon) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewPadded(container.New(layout.NewCustomPaddedVBoxLayout(0),
		container.New(&bottomRightLayout{}, w.icon, w.badge),
		w.label,
	))
	return widget.NewSimpleRenderer(c)
}

func (w *assetNodeIcon) Set(n *asset.Node) {
	defer w.Refresh()
	showFolder := func(w *assetNodeIcon) {
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
	iwidget.RefreshImageAsync(w.icon, func() (fyne.Resource, error) {
		switch as.Variant() {
		case app.VariantBPO:
			return w.eis.InventoryTypeBPO(as.Type.ID, app.IconPixelSize)
		case app.VariantBPC:
			return w.eis.InventoryTypeBPC(as.Type.ID, app.IconPixelSize)
		case app.VariantSKIN:
			return w.eis.InventoryTypeSKIN(as.Type.ID, app.IconPixelSize)
		default:
			return w.eis.InventoryTypeIcon(as.Type.ID, app.IconPixelSize)
		}
	})
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
