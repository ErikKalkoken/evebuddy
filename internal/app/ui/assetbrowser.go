package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"
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

// assetBrowser shows the attributes for the current character.
type assetBrowser struct {
	widget.BaseWidget

	Navigation *assetBrowserNavigation
	Selected   *assetBrowserSelected

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
	var ac asset.Collection
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
		ac = asset.NewFromCorporationAssets(assets, el)
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
		ac = asset.NewFromCharacterAssets(assets, el)
	}
	a.Navigation.update(ac)
}

const (
	assetCategoryAll        = "All"
	assetCategoryDeliveries = "Deliveries"
	assetCategoryImpounded  = "Impounded"
	assetCategoryInSpace    = "In Space"
	assetCategoryOffice     = "Office"
	assetCategoryPersonal   = "Personal"
	assetCategorySafety     = "Safety"
)

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

	OnSelected func()

	ab             *assetBrowser
	navigation     *iwidget.Tree[assetNavNode]
	nodeLookup     map[*asset.Node]widget.TreeNodeID
	selectCategory *kxwidget.FilterChipSelect
	top            *widget.Label

	mu sync.RWMutex
	ac asset.Collection
}

func newAssetBrowserNavigation(ab *assetBrowser) *assetBrowserNavigation {
	a := &assetBrowserNavigation{
		ab:         ab,
		nodeLookup: make(map[*asset.Node]widget.TreeNodeID),
		top:        makeTopLabel(),
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
	a.navigation.OnSelectedNode = func(n assetNavNode) {
		a.ab.Selected.set(n.node)
		if a.OnSelected != nil {
			a.OnSelected()
		}
		if ab.u.isMobile {
			a.navigation.UnselectAll()
		}
	}
	if a.ab.forCorporation {
		a.selectCategory = kxwidget.NewFilterChipSelect("", []string{
			assetCategoryOffice,
			assetCategoryImpounded,
			assetCategoryDeliveries,
			assetCategoryInSpace,
			assetCategorySafety,
			assetCategoryAll,
		}, func(string) {
			go a.redraw()
		})
		a.selectCategory.Selected = assetCategoryOffice
		a.selectCategory.SortDisabled = true
	} else {
		a.selectCategory = kxwidget.NewFilterChipSelect("", []string{
			assetCategoryPersonal,
			assetCategoryDeliveries,
			assetCategoryInSpace,
			assetCategorySafety,
			assetCategoryAll,
		}, func(string) {
			go a.redraw()
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

func (a *assetBrowserNavigation) update(ac asset.Collection) {
	a.mu.Lock()
	a.ac = ac
	a.mu.Unlock()
	a.redraw()
}

func (a *assetBrowserNavigation) redraw() {
	m := map[string]asset.Filter{
		assetCategoryDeliveries: asset.FilterDeliveries,
		assetCategoryImpounded:  asset.FilterImpounded,
		assetCategoryInSpace:    asset.FilterInSpace,
		assetCategoryOffice:     asset.FilterOffice,
		assetCategoryPersonal:   asset.FilterPersonalAssets,
		assetCategorySafety:     asset.FilterSafety,
	}

	a.mu.Lock()
	a.ac.ApplyFilter(m[a.selectCategory.Selected])
	trees := a.ac.Trees()
	a.mu.Unlock()

	var td iwidget.TreeData[assetNavNode]
	var id int
	addNodes(&td, iwidget.TreeRootID, trees, &id)

	nodeLookUp := make(map[*asset.Node]widget.TreeNodeID)
	for n := range td.All() {
		nodeLookUp[n.node] = n.UID()
	}

	// addSumsFrom2LevelsDown := func() {
	// 	for _, locations := range td.Children(iwidget.TreeRootID) {
	// 		for _, n1 := range td.Children(locations.UID()) {
	// 			var sum2 optional.Optional[int]
	// 			for _, n2 := range td.Children(n1.UID()) {
	// 				sum2 = optional.Sum(sum2, n2.itemCount)
	// 			}
	// 			n1.itemCount = sum2
	// 			td.Replace(n1)
	// 		}
	// 	}
	// }

	// addSumsFrom3LevelsDown := func() {
	// 	for _, locations := range td.Children(iwidget.TreeRootID) {
	// 		for _, n1 := range td.Children(locations.UID()) {
	// 			var sum2 optional.Optional[int]
	// 			for _, n2 := range td.Children(n1.UID()) {
	// 				var sum3 optional.Optional[int]
	// 				for _, n3 := range td.Children(n2.UID()) {
	// 					sum3 = optional.Sum(sum3, n3.itemCount)
	// 				}
	// 				n2.itemCount = sum3
	// 				td.Replace(n2)
	// 				sum2 = optional.Sum(sum2, sum3)
	// 			}
	// 			n1.itemCount = sum2
	// 			td.Replace(n1)
	// 		}
	// 	}
	// }

	// // Update counts
	// switch filter {
	// case assetCategoryOffice:
	// 	addSumsFrom2LevelsDown()
	// case assetCategorySafety:
	// 	if a.ab.forCorporation {
	// 		addSumsFrom3LevelsDown()
	// 	} else {
	// 		addSumsFrom2LevelsDown()
	// 	}
	// case assetCategoryImpounded:
	// 	addSumsFrom3LevelsDown()

	// }
	// for _, locations := range td.Children(iwidget.TreeRootID) {
	// 	var sum optional.Optional[int]
	// 	for _, n1 := range td.Children(locations.UID()) {
	// 		sum = optional.Sum(sum, n1.itemCount)
	// 	}
	// 	locations.itemCount = sum
	// 	td.Replace(locations)
	// }

	count, _ := td.ChildrenCount(iwidget.TreeRootID)
	top := fmt.Sprintf("%d locations", count)
	fyne.Do(func() {
		a.nodeLookup = nodeLookUp
		a.navigation.UnselectAll()
		a.navigation.CloseAllBranches()
		a.navigation.Set(td)
		a.ab.Selected.clear()
		a.setTop(top, widget.MediumImportance)
	})
}

// addNodes adds nodes with children recursively to tree data at uid
// and sets initial item counts.
func addNodes(td *iwidget.TreeData[assetNavNode], uid widget.TreeNodeID, nodes []*asset.Node, id *int) {
	slices.SortFunc(nodes, func(a, b *asset.Node) int {
		return strings.Compare(a.DisplayName(), b.DisplayName())
	})
	for _, n := range nodes {
		if !n.IsContainer() {
			continue
		}
		var itemCount optional.Optional[int]
		if n.IsRoot() {
			for _, c := range n.Children() {
				if n := len(c.Children()); n > 0 {
					itemCount.Set(itemCount.ValueOrZero() + n)
				}
			}
		} else if !n.IsShip() && n.Category() != asset.NodeFitting {
			if n := len(n.Children()); n > 0 {
				itemCount.Set(n)
			}
		}
		*id++
		uid, err := td.Add(uid, assetNavNode{
			id:        *id,
			node:      n,
			itemCount: itemCount,
		})
		if err != nil {
			slog.Error("Failed to add node", "ID", n.ID(), "Name", n.DisplayName(), "error", err)
			return
		}
		children := n.Children()
		if len(children) > 0 {
			addNodes(td, uid, children, id)
		}
	}
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
	if !a.ab.u.isMobile {
		a.navigation.Select(uid)
	}
	for _, x := range a.navigation.Data().Path(uid) {
		a.navigation.OpenBranch(x)
	}
	a.navigation.ScrollTo(uid)
}

type assetBrowserSelected struct {
	widget.BaseWidget

	ab       *assetBrowser
	bottom   *widget.Label
	children []*asset.Node
	grid     *widget.GridWrap
	node     *asset.Node
	top      *assetBrowserLocation
}

func newAssetBrowserSelected(ab *assetBrowser) *assetBrowserSelected {
	a := &assetBrowserSelected{
		ab:     ab,
		bottom: widget.NewLabel(""),
	}
	a.ExtendBaseWidget(a)
	a.grid = a.makeAssetGrid()
	a.top = newAssetBrowserLocation(a)
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
		if n.IsContainer() {
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
		showAssetDetailWindow(a.ab.u, newCorporationAssetRow(ca, a.ab.Navigation.ac, name))
		return
	}
	ca, ok := n.CharacterAsset()
	if !ok {
		return
	}
	showAssetDetailWindow(a.ab.u, newCharacterAssetRow(ca, a.ab.Navigation.ac, a.ab.u.scs.CharacterName))
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

func (a *assetBrowserLocation) set(node *asset.Node) {
	a.breadcrumbs.RemoveAll()
	p := theme.Padding()
	for _, n := range node.Path() {
		l := widget.NewHyperlink(n.DisplayName(), nil)
		l.OnTapped = func() {
			a.selected.ab.Navigation.selectContainer(n)
			a.selected.set(n)
		}
		a.breadcrumbs.Add(l)
		x := container.New(layout.NewCustomPaddedLayout(0, 0, -2*p, -2*p), widget.NewLabel("＞"))
		a.breadcrumbs.Add(x)
	}
	a.breadcrumbs.Add(widget.NewLabel(node.DisplayName()))

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
