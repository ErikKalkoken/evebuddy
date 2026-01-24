package ui

import (
	"context"
	"fmt"
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

	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/asset"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

// assetBrowser shows the attributes for the current character.
type assetBrowser struct {
	widget.BaseWidget

	Navigation *assetBrowserNavigation
	Selected   *assetBrowserSelected

	ac             asset.Collection
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
			a.update()
		})
		a.u.corporationSectionChanged.AddListener(func(ctx context.Context, arg corporationSectionUpdated) {
			if corporationIDOrZero(a.corporation.Load()) != arg.corporationID {
				return
			}
			if arg.section == app.SectionCorporationAssets {
				a.update()
			}
		})
	} else {
		a.u.currentCharacterExchanged.AddListener(func(_ context.Context, c *app.Character) {
			a.character.Store(c)
			a.update()
		})
		a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
			if characterIDOrZero(a.character.Load()) != arg.characterID {
				return
			}
			if arg.section == app.SectionCharacterAssets {
				a.update()
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

func (a *assetBrowser) update() {
	clear := func() {
		fyne.Do(func() {
			a.Navigation.clear()
			a.Selected.clear()
		})
	}
	ctx := context.Background()
	el, err := a.u.eus.ListLocations(ctx)
	if err != nil {
		panic(err)
	}
	var ac asset.Collection
	if a.forCorporation {
		corporationID := corporationIDOrZero(a.corporation.Load())
		if corporationID == 0 {
			clear()
			return
		}
		assets, err := a.u.rs.ListAssets(ctx, corporationID)
		if err != nil {
			panic(err)
		}
		ac = asset.NewFromCorporationAssets(assets, el)
	} else {
		characterID := characterIDOrZero(a.character.Load())
		if characterID == 0 {
			clear()
			return
		}
		assets, err := a.u.cs.ListAssets(ctx, characterID)
		if err != nil {
			panic(err)
		}
		ac = asset.NewFromCharacterAssets(assets, el)
	}
	a.Navigation.update(ac.LocationNodes())

	fyne.Do(func() {
		a.ac = ac
	})
}

// assetNavNode represents a node in the navigation tree
type assetNavNode struct {
	id        int
	itemCount optional.Optional[int]
	node      *asset.Node
}

func (an assetNavNode) UID() widget.TreeNodeID {
	return widget.TreeNodeID(strconv.Itoa(an.id))
}

type assetBrowserNavigation struct {
	widget.BaseWidget

	ab         *assetBrowser
	navigation *iwidget.Tree[assetNavNode]
	nodeLookup map[*asset.Node]widget.TreeNodeID
	top        *widget.Label
}

func newAssetBrowserNavigation(ab *assetBrowser) *assetBrowserNavigation {
	a := &assetBrowserNavigation{
		ab:         ab,
		nodeLookup: make(map[*asset.Node]widget.TreeNodeID),
		top:        makeTopLabel(),
	}
	a.ExtendBaseWidget(a)
	a.navigation = a.makeNavigation()
	return a
}

func (a *assetBrowserNavigation) makeNavigation() *iwidget.Tree[assetNavNode] {
	tree := iwidget.NewTree(
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
		func(n assetNavNode, _ bool, co fyne.CanvasObject) {
			b := co.(*fyne.Container).Objects
			b[0].(*widget.Label).SetText(n.node.DisplayName())
			var s string
			if !n.itemCount.IsEmpty() {
				s = humanize.Comma(int64(n.itemCount.ValueOrZero()))
			}
			b[1].(*widget.Label).SetText(s)
		},
	)
	tree.OnSelectedNode = func(n assetNavNode) {
		a.ab.Selected.set(n.node)
	}
	return tree
}

func (a *assetBrowserNavigation) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(
		a.top,
		nil,
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

func (a *assetBrowserNavigation) update(nodes []*asset.Node) {
	var td iwidget.TreeData[assetNavNode]
	var id int
	nodeLookUp := make(map[*asset.Node]widget.TreeNodeID)

	// addNodes adds a list of nodes with children recursively to tree data at uid.
	var addNodes func(widget.TreeNodeID, []*asset.Node)
	addNodes = func(uid widget.TreeNodeID, nodes []*asset.Node) {
		sortNodes(nodes)
		for _, n := range nodes {
			if !n.IsContainer {
				continue
			}
			id++
			var itemCount optional.Optional[int]
			if n.IsRoot() {
				for _, x := range n.Children() {
					if !x.ItemCount.IsEmpty() {
						itemCount.Set(itemCount.ValueOrZero() + x.ItemCount.ValueOrZero())
					}
				}
			} else if !n.IsShip && n.Category() != asset.NodeFitting {
				itemCount = n.ItemCount
			}
			uid, err := td.Add(uid, assetNavNode{
				id:        id,
				node:      n,
				itemCount: itemCount,
			})
			if err != nil {
				slog.Error("Failed to add node", "ID", n.ID(), "Name", n.DisplayName())
				return
			}
			nodeLookUp[n] = uid
			children := n.Children()
			if len(children) > 0 {
				addNodes(uid, children)
			}
		}
	}
	addNodes(iwidget.TreeRootID, nodes)
	fyne.Do(func() {
		a.nodeLookup = nodeLookUp
		a.navigation.Set(td)
		a.ab.Selected.clear()
		a.navigation.UnselectAll()
		a.top.SetText(fmt.Sprintf("%d locations", len(nodes)))
	})
}

func (a *assetBrowserNavigation) setTop(s string, i widget.Importance) {
	a.top.Text = s
	a.top.Importance = i
	a.top.Refresh()
}

func (a *assetBrowserNavigation) selectContainer(n *asset.Node) {
	uid, found := a.nodeLookup[n]
	if !found {
		return
	}
	a.navigation.UnselectAll()
	a.navigation.Select(uid)
	a.navigation.ScrollTo(uid)
	for _, x := range a.navigation.Data().Path(uid) {
		a.navigation.OpenBranch(x)
	}
}

type assetBrowserSelected struct {
	widget.BaseWidget

	ab       *assetBrowser
	bottom   *widget.Label
	children []*asset.Node
	grid     *widget.GridWrap
	node     *asset.Node
	top      *assetBreadcrumbs
}

func newAssetBrowserSelected(ab *assetBrowser) *assetBrowserSelected {
	a := &assetBrowserSelected{
		ab:     ab,
		bottom: widget.NewLabel(""),
	}
	a.ExtendBaseWidget(a)
	a.grid = a.makeAssetGrid()
	a.top = newAssetBreadcrumbs(a)
	return a
}

func (a *assetBrowserSelected) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(
		a.top,
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
		if n.IsContainer {
			a.ab.Navigation.selectContainer(n)
			a.set(n)
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
		showAssetDetailWindow(a.ab.u, newCorporationAssetRow(ca, a.ab.ac, name))
		return
	}
	ca, ok := n.CharacterAsset()
	if !ok {
		return
	}
	showAssetDetailWindow(a.ab.u, newCharacterAssetRow(ca, a.ab.ac, a.ab.u.scs.CharacterName))
}

func (a *assetBrowserSelected) clear() {
	a.node = nil
	a.children = make([]*asset.Node, 0)
	a.grid.Refresh()
	a.top.clear()
	a.bottom.SetText("")
}

func (a *assetBrowserSelected) set(node *asset.Node) {
	a.node = node
	nodes := node.Children()
	sortNodes(nodes)
	a.children = nodes
	a.grid.Refresh()
	a.top.set(node)

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
		s = fmt.Sprintf("%s Items - %s ISK Est. Price", humanize.Comma(itemCount), ihumanize.NumberF(value, 1))
	}
	a.bottom.SetText(s)
}

func sortNodes(nodes []*asset.Node) {
	slices.SortFunc(nodes, func(a, b *asset.Node) int {
		return strings.Compare(a.DisplayName(), b.DisplayName())
	})
}

type assetBreadcrumbs struct {
	widget.BaseWidget

	body     *fyne.Container
	infoIcon *iwidget.TappableIcon
	selected *assetBrowserSelected
}

func newAssetBreadcrumbs(selected *assetBrowserSelected) *assetBreadcrumbs {
	a := &assetBreadcrumbs{
		body:     container.New(layout.NewRowWrapLayoutWithCustomPadding(0, 0)),
		infoIcon: iwidget.NewTappableIcon(theme.InfoIcon(), nil),
		selected: selected,
	}
	a.ExtendBaseWidget(a)
	return a
}

func (a *assetBreadcrumbs) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, nil, a.infoIcon, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *assetBreadcrumbs) clear() {
	a.body.RemoveAll()
	a.infoIcon.Hide()
}

func (a *assetBreadcrumbs) set(node *asset.Node) {
	a.body.RemoveAll()
	p := theme.Padding()
	for _, n := range node.Path() {
		l := widget.NewHyperlink(n.DisplayName(), nil)
		l.OnTapped = func() {
			a.selected.ab.Navigation.selectContainer(n)
			a.selected.set(n)
		}
		a.body.Add(l)
		x := container.New(layout.NewCustomPaddedLayout(0, 0, -2*p, -2*p), widget.NewLabel("＞"))
		a.body.Add(x)
	}
	a.body.Add(widget.NewLabel(node.DisplayName()))

	switch node.Category() {
	case asset.NodeLocation:
		el, ok := node.Location()
		if !ok {
			return
		}
		if el.Variant() == app.EveLocationUnknown {
			return
		}
		a.infoIcon.OnTapped = func() {
			a.selected.ab.u.ShowLocationInfoWindow(el.ID)
		}
		a.infoIcon.Show()
	case asset.NodeAsset:
		a.infoIcon.OnTapped = func() {
			a.selected.showNodeInfo(node)
		}
		a.infoIcon.Show()
	default:
		a.infoIcon.Hide()
	}
}

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
		w.label.SetText(n.DisplayName())
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
	if as.Type.ID == app.EveTypeOffice {
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
