package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/app/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type locationDataNodeType uint

const (
	nodeLocation locationDataNodeType = iota + 1
	nodeShipHangar
	nodeItemHangar
	nodeContainer
	nodeShip
	nodeCargoBay
	nodeFuelBay
	nodeAssetSafety
)

// locationDataNode is a node for the asset tree widget.
type locationDataNode struct {
	CharacterID         int32
	ContainerID         int64
	Name                string
	Count               int
	IsUnknown           bool
	SystemName          string
	SystemSecurityValue float32
	SystemSecurityType  app.SolarSystemSecurityType
	Type                locationDataNodeType
}

func (n locationDataNode) UID() widget.TreeNodeID {
	if n.CharacterID == 0 || n.ContainerID == 0 || n.Type == 0 {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%d-%d", n.CharacterID, n.ContainerID, n.Type)
}

func (n locationDataNode) IsRoot() bool {
	return n.Type == nodeLocation
}

var defaultAssetIcon = theme.NewDisabledResource(resourceQuestionmarkSvg)

// assetsArea is the UI area that shows the skillqueue
type assetsArea struct {
	content          fyne.CanvasObject
	assetGrid        *widget.GridWrap
	assets           []*app.CharacterAsset
	locationPath     *fyne.Container
	assetsBottom     *widget.Label
	locationsWidget  *widget.Tree
	locationsData    *fynetree.FyneTree[locationDataNode]
	locationsTop     *widget.Label
	selectedLocation optional.Optional[locationDataNode]
	assetCollection  assetcollection.AssetCollection
	u                *UI
}

func (u *UI) newAssetsArea() *assetsArea {
	myHBox := layout.NewCustomPaddedHBoxLayout(-5)
	a := assetsArea{
		assets:        make([]*app.CharacterAsset, 0),
		locationPath:  container.New(myHBox),
		assetsBottom:  widget.NewLabel(""),
		locationsData: fynetree.New[locationDataNode](),
		locationsTop:  widget.NewLabel(""),
		u:             u,
	}
	a.locationsTop.TextStyle.Bold = true
	a.locationsWidget = a.makeLocationsTree()
	locations := container.NewBorder(
		container.NewVBox(a.locationsTop, widget.NewSeparator()),
		nil,
		nil,
		nil,
		a.locationsWidget,
	)

	a.assetGrid = a.makeAssetGrid()
	assets := container.NewBorder(
		container.NewVBox(a.locationPath, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), a.assetsBottom),
		nil,
		nil,
		a.assetGrid,
	)
	main := container.NewHSplit(locations, assets)
	main.SetOffset(0.33)
	a.content = main
	return &a
}

func (a *assetsArea) makeLocationsTree() *widget.Tree {
	t := widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return a.locationsData.ChildUIDs(uid)
		},
		func(uid widget.TreeNodeID) bool {
			return a.locationsData.IsBranch(uid)
		},
		func(branch bool) fyne.CanvasObject {
			prefix := widget.NewLabel("1.0")
			prefix.Importance = widget.HighImportance
			return container.NewHBox(prefix, widget.NewLabel("Location"))
		},
		func(uid widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			row := co.(*fyne.Container).Objects
			prefix := row[0].(*widget.Label)
			label := row[1].(*widget.Label)
			n, ok := a.locationsData.Value(uid)
			if !ok {
				return
			}
			label.SetText(makeNameWithCount(n.Name, n.Count))
			if n.IsRoot() {
				if !n.IsUnknown {
					prefix.Text = fmt.Sprintf("%.1f", n.SystemSecurityValue)
					prefix.Importance = n.SystemSecurityType.ToImportance()
				} else {
					prefix.Text = "?"
					prefix.Importance = widget.LowImportance
				}
				prefix.Refresh()
				prefix.Show()
			} else {
				prefix.Hide()
			}
		},
	)
	t.OnSelected = func(uid widget.TreeNodeID) {
		n, ok := a.locationsData.Value(uid)
		if !ok {
			return
		}
		if err := a.selectLocation(n); err != nil {
			slog.Warn("Failed to redraw assets", "err", err)
		}
	}
	return t
}

func (a *assetsArea) clearAssets() error {
	a.assets = make([]*app.CharacterAsset, 0)
	a.assetGrid.Refresh()
	a.locationPath.RemoveAll()
	a.selectedLocation.Clear()
	return nil
}

func (a *assetsArea) makeAssetGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.assets)
		},
		func() fyne.CanvasObject {
			return widgets.NewAssetListWidget(a.u.EveImageService, defaultAssetIcon)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.assets) {
				return
			}
			ca := a.assets[id]
			item := co.(*widgets.AssetListWidget)
			item.SetAsset(ca)
		},
	)
	g.OnSelected = func(id widget.GridWrapItemID) {
		defer g.UnselectAll()
		if id >= len(a.assets) {
			return
		}
		ca := a.assets[id]
		if ca.IsContainer() {
			if a.selectedLocation.IsEmpty() {
				return
			}
			location := a.selectedLocation.ValueOrZero()
			for _, uid := range a.locationsData.ChildUIDs(location.UID()) {
				n, ok := a.locationsData.Value(uid)
				if !ok {
					continue
				}
				if n.ContainerID == ca.ItemID {
					a.selectLocation(n)
				}
			}
		} else {
			a.u.showTypeInfoWindow(ca.EveType.ID, a.u.characterID())
		}
	}
	return g
}

func (a *assetsArea) redraw() {
	a.locationsWidget.CloseAllBranches()
	a.locationsWidget.ScrollToTop()
	t, i, err := func() (string, widget.Importance, error) {
		if err := a.clearAssets(); err != nil {
			return "", 0, err
		}
		tree, err := a.newLocationData()
		if err != nil {
			return "", 0, err
		}
		a.locationsData = tree
		locationsCount := len(a.locationsData.ChildUIDs(""))
		return a.makeTopText(locationsCount)
	}()
	if err != nil {
		slog.Error("Failed to redraw asset locations UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.locationsTop.Text = t
	a.locationsTop.Importance = i
	a.locationsTop.Refresh()
	a.locationsWidget.Refresh()
}

func (a *assetsArea) newLocationData() (*fynetree.FyneTree[locationDataNode], error) {
	ctx := context.TODO()
	tree := fynetree.New[locationDataNode]()
	if !a.u.hasCharacter() {
		return tree, nil
	}
	characterID := a.u.characterID()
	assets, err := a.u.CharacterService.ListCharacterAssets(ctx, characterID)
	if err != nil {
		return tree, err
	}
	oo, err := a.u.EveUniverseService.ListEveLocations(ctx)
	if err != nil {
		return tree, err
	}
	a.assetCollection = assetcollection.New(assets, oo)
	locationNodes := a.assetCollection.Locations()
	slices.SortFunc(locationNodes, func(a assetcollection.LocationNode, b assetcollection.LocationNode) int {
		return cmp.Compare(a.Location.DisplayName(), b.Location.DisplayName())
	})
	for _, ln := range locationNodes {
		el := ln.Location
		location := locationDataNode{
			CharacterID: characterID,
			ContainerID: el.ID,
			Type:        nodeLocation,
			Name:        el.DisplayName(),
			Count:       len(ln.Nodes()),
		}
		if el.SolarSystem != nil {
			location.SystemName = el.SolarSystem.Name
			location.SystemSecurityValue = float32(el.SolarSystem.SecurityStatus)
			location.SystemSecurityType = el.SolarSystem.SecurityType()
		} else {
			location.IsUnknown = true
		}
		locationUID := tree.MustAdd("", location.UID(), location)

		topAssets := ln.Nodes()
		slices.SortFunc(topAssets, func(a assetcollection.AssetNode, b assetcollection.AssetNode) int {
			return cmp.Compare(a.Asset.DisplayName(), b.Asset.DisplayName())
		})
		itemCount := 0
		shipCount := 0
		ships := make([]assetcollection.AssetNode, 0)
		itemContainers := make([]assetcollection.AssetNode, 0)
		assetSafety := make([]assetcollection.AssetNode, 0)
		for _, an := range topAssets {
			if an.Asset.IsInAssetSafety() {
				assetSafety = append(assetSafety, an)
			} else if an.Asset.IsInHangar() {
				if an.Asset.EveType.IsShip() {
					shipCount++
				} else {
					itemCount++
				}
				if an.Asset.IsContainer() {
					if an.Asset.EveType.IsShip() {
						ships = append(ships, an)
					} else {
						itemContainers = append(itemContainers, an)
					}
				}
			}
		}

		shipHangar := locationDataNode{
			CharacterID: characterID,
			ContainerID: el.ID,
			Name:        "Ship Hangar",
			Count:       shipCount,
			Type:        nodeShipHangar,
		}
		shipsUID := tree.MustAdd(locationUID, shipHangar.UID(), shipHangar)
		for _, an := range ships {
			ship := an.Asset
			ldn := locationDataNode{
				CharacterID: characterID,
				ContainerID: an.Asset.ItemID,
				Name:        ship.DisplayName2(),
				Type:        nodeShip,
			}
			shipUID := tree.MustAdd(shipsUID, ldn.UID(), ldn)
			cargo := make([]assetcollection.AssetNode, 0)
			fuel := make([]assetcollection.AssetNode, 0)
			for _, an2 := range an.Nodes() {
				if an2.Asset.IsInCargoBay() {
					cargo = append(cargo, an2)
				} else if an2.Asset.IsInFuelBay() {
					fuel = append(fuel, an2)
				}
			}
			cln := locationDataNode{
				CharacterID: characterID,
				ContainerID: ship.ItemID,
				Name:        "Cargo Bay",
				Count:       len(cargo),
				Type:        nodeCargoBay,
			}
			tree.MustAdd(shipUID, cln.UID(), cln)
			if ship.EveType.HasFuelBay() {
				ldn := locationDataNode{
					CharacterID: characterID,
					ContainerID: an.Asset.ItemID,
					Name:        "Fuel Bay",
					Count:       len(fuel),
					Type:        nodeFuelBay,
				}
				tree.MustAdd(shipUID, ldn.UID(), ldn)
			}
		}

		itemHangar := locationDataNode{
			CharacterID: characterID,
			ContainerID: el.ID,
			Name:        "Item Hangar",
			Count:       itemCount,
			Type:        nodeItemHangar,
		}
		itemsUID := tree.MustAdd(locationUID, itemHangar.UID(), itemHangar)
		for _, an := range itemContainers {
			ldn := locationDataNode{
				CharacterID: characterID,
				ContainerID: an.Asset.ItemID,
				Name:        an.Asset.DisplayName(),
				Count:       len(an.Nodes()),
				Type:        nodeContainer,
			}
			tree.MustAdd(itemsUID, ldn.UID(), ldn)
		}

		if len(assetSafety) > 0 {
			an := assetSafety[0]
			ldn := locationDataNode{
				CharacterID: characterID,
				ContainerID: an.Asset.ItemID,
				Name:        "Asset Safety",
				Count:       len(an.Nodes()),
				Type:        nodeAssetSafety,
			}
			tree.MustAdd(locationUID, ldn.UID(), ldn)
		}
	}
	return tree, nil
}

func (a *assetsArea) makeTopText(total int) (string, widget.Importance, error) {
	if !a.u.hasCharacter() {
		return "No character", widget.LowImportance, nil
	}
	hasData := a.u.StatusCacheService.CharacterSectionExists(a.u.characterID(), app.SectionAssets)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	locations := humanize.Comma(int64(total))
	return fmt.Sprintf("%s locations", locations), widget.MediumImportance, nil
}

func (a *assetsArea) selectLocation(location locationDataNode) error {
	a.assets = make([]*app.CharacterAsset, 0)
	a.assetGrid.Refresh()
	a.selectedLocation.Set(location)
	selectedUID := location.UID()
	for _, uid := range a.locationsData.Path(selectedUID) {
		a.locationsWidget.OpenBranch(uid)
	}
	a.locationsWidget.ScrollTo(selectedUID)
	a.locationsWidget.Select(selectedUID)
	var f func(context.Context, int32, int64) ([]*app.CharacterAsset, error)
	switch location.Type {
	case nodeShipHangar:
		f = a.u.CharacterService.ListCharacterAssetsInShipHangar
	case nodeItemHangar:
		f = a.u.CharacterService.ListCharacterAssetsInItemHangar
	default:
		f = a.u.CharacterService.ListCharacterAssetsInLocation
	}
	assets, err := f(context.TODO(), location.CharacterID, location.ContainerID)
	if err != nil {
		return err
	}
	switch location.Type {
	case nodeItemHangar:
		containers := make([]*app.CharacterAsset, 0)
		items := make([]*app.CharacterAsset, 0)
		for _, ca := range assets {
			if ca.IsContainer() {
				containers = append(containers, ca)
			} else {
				items = append(items, ca)
			}
		}
		assets = slices.Concat(containers, items)
	case nodeCargoBay:
		cargo := make([]*app.CharacterAsset, 0)
		for _, ca := range assets {
			if !ca.IsInCargoBay() {
				continue
			}
			cargo = append(cargo, ca)
		}
		assets = cargo
	case nodeFuelBay:
		fuel := make([]*app.CharacterAsset, 0)
		for _, ca := range assets {
			if !ca.IsInFuelBay() {
				continue
			}
			fuel = append(fuel, ca)
		}
		assets = fuel
	}
	a.assets = assets
	a.assetGrid.Refresh()
	var total float64
	for _, ca := range assets {
		total += ca.Price.ValueOrZero() * float64(ca.Quantity)
	}
	a.updateLocationPath(location)
	a.assetsBottom.SetText(fmt.Sprintf("%d Items - %s ISK Est. Price", len(assets), ihumanize.Number(total, 1)))
	return nil
}

func (a *assetsArea) updateLocationPath(location locationDataNode) {
	path := make([]locationDataNode, 0)
	for _, uid := range a.locationsData.Path(location.UID()) {
		n, ok := a.locationsData.Value(uid)
		if !ok {
			continue
		}
		path = append(path, n)
	}
	path = append(path, location)
	a.locationPath.RemoveAll()
	for i, n := range path {
		isLast := i == len(path)-1
		if !isLast {
			l := kwidget.NewTappableLabel(n.Name, func() {
				if err := a.selectLocation(n); err != nil {
					slog.Warn("Failed to redraw assets", "err", err)
				}
			})
			a.locationPath.Add(l)
		} else {
			l := widget.NewLabel(n.Name)
			l.TextStyle.Bold = true
			a.locationPath.Add(l)
		}
		if n.IsRoot() {
			if !n.IsUnknown {
				a.locationPath.Add(kwidget.NewTappableIcon(theme.InfoIcon(), func() {
					a.u.showLocationInfoWindow(n.ContainerID)
				}))
				a.locationPath.Add(container.NewPadded())
			}
		}
		if !isLast {
			l := widget.NewLabel("＞")
			l.Importance = widget.LowImportance
			a.locationPath.Add(l)
		}
	}
}

func makeNameWithCount(name string, count int) string {
	if count == 0 {
		return name
	}
	return fmt.Sprintf("%s (%s)", name, humanize.Comma(int64(count)))
}

// func (u *ui) showNewAssetWindow(ca *app.CharacterAsset) {
// 	var name string
// 	if ca.Name != "" {
// 		name = fmt.Sprintf(" \"%s\" ", ca.Name)
// 	}
// 	title := fmt.Sprintf("%s%s(%s): Contents", ca.EveType.Name, name, ca.EveType.Group.Name)
// 	w := u.fyneApp.NewWindow(u.makeWindowTitle(title))
// 	oo, err := u.CharacterService.ListCharacterAssetsInLocation(context.TODO(), ca.CharacterID, ca.ItemID)
// 	if err != nil {
// 		panic(err)
// 	}
// 	data := binding.NewUntypedList()
// 	if err := data.Set(copyToUntypedSlice(oo)); err != nil {
// 		panic(err)
// 	}
// 	top := widget.NewLabel(fmt.Sprintf("%s items", humanize.Comma(int64(len(oo)))))
// 	top.TextStyle.Bold = true
// 	assets := u.makeAssetGrid(data)
// 	content := container.NewBorder(container.NewVBox(top, widget.NewSeparator()), nil, nil, nil, assets)

// 	w.SetContent(content)
// 	w.Resize(fyne.Size{Width: 650, Height: 500})
// 	w.Show()
// }
