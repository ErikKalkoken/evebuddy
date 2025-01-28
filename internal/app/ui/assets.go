package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
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

// AssetsArea is the UI area that shows the skillqueue
type AssetsArea struct {
	Content        fyne.CanvasObject
	Locations      fyne.CanvasObject
	LocationAssets fyne.CanvasObject
	OnSelected     func()

	assetCollection  assetcollection.AssetCollection
	assetGrid        *widget.GridWrap
	assets           []*app.CharacterAsset
	assetsBottom     *widget.Label
	locationPath     *widget.Label
	locationsData    *fynetree.FyneTree[locationDataNode]
	locationsTop     *widget.Label
	locationsWidget  *widget.Tree
	selectedLocation optional.Optional[locationDataNode]
	u                *BaseUI
}

func (u *BaseUI) NewAssetsArea() *AssetsArea {
	lp := widget.NewLabel("")
	lp.Wrapping = fyne.TextWrapWord
	a := AssetsArea{
		assets:        make([]*app.CharacterAsset, 0),
		assetsBottom:  widget.NewLabel(""),
		locationPath:  lp,
		locationsData: fynetree.New[locationDataNode](),
		locationsTop:  widget.NewLabel(""),
		u:             u,
	}
	a.locationsTop.TextStyle.Bold = true
	a.locationsWidget = a.makeLocationsTree()
	a.Locations = container.NewBorder(
		container.NewVBox(a.locationsTop, widget.NewSeparator()),
		nil,
		nil,
		nil,
		a.locationsWidget,
	)

	a.assetGrid = a.makeAssetGrid()
	a.LocationAssets = container.NewBorder(
		container.NewVBox(a.locationPath, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), a.assetsBottom),
		nil,
		nil,
		a.assetGrid,
	)
	main := container.NewHSplit(a.Locations, a.LocationAssets)
	main.SetOffset(0.33)
	a.Content = main
	return &a
}

func (a *AssetsArea) makeLocationsTree() *widget.Tree {
	labelSizeName := theme.SizeNameText
	if !a.u.IsDesktop() {
		labelSizeName = theme.SizeNameCaptionText
	}
	makeNameWithCount := func(name string, count int) string {
		if count == 0 {
			return name
		}
		return fmt.Sprintf("%s (%s)", name, humanize.Comma(int64(count)))
	}
	t := widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return a.locationsData.ChildUIDs(uid)
		},
		func(uid widget.TreeNodeID) bool {
			return a.locationsData.IsBranch(uid)
		},
		func(branch bool) fyne.CanvasObject {
			return container.New(layout.NewCustomPaddedHBoxLayout(-5),
				widgets.NewLabelWithSize("1.0", labelSizeName),
				widgets.NewLabelWithSize("Location", labelSizeName),
			)
		},
		func(uid widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			row := co.(*fyne.Container).Objects
			prefix := row[0].(*widgets.Label)
			label := row[1].(*widgets.Label)
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
		if n.Type == nodeLocation {
			t.OpenBranch(uid)
			t.UnselectAll()
			return
		}
		if err := a.selectLocation(n); err != nil {
			slog.Warn("Failed to redraw assets", "err", err)
		}
		if a.OnSelected != nil {
			a.OnSelected()
			t.UnselectAll()
		}
	}
	return t
}

func (a *AssetsArea) clearAssets() error {
	a.assets = make([]*app.CharacterAsset, 0)
	a.assetGrid.Refresh()
	a.locationPath.SetText("")
	a.selectedLocation.Clear()
	return nil
}

func (a *AssetsArea) makeAssetGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.assets)
		},
		func() fyne.CanvasObject {
			const assetListIconSize = 64
			return widgets.NewAsset(func(image *canvas.Image, ca *app.CharacterAsset) {
				RefreshImageResourceAsync(image, func() (fyne.Resource, error) {
					switch ca.Variant() {
					case app.VariantSKIN:
						return a.u.EveImageService.InventoryTypeSKIN(ca.EveType.ID, assetListIconSize)
					case app.VariantBPO:
						return a.u.EveImageService.InventoryTypeBPO(ca.EveType.ID, assetListIconSize)
					case app.VariantBPC:
						return a.u.EveImageService.InventoryTypeBPC(ca.EveType.ID, assetListIconSize)
					default:
						return a.u.EveImageService.InventoryTypeIcon(ca.EveType.ID, assetListIconSize)
					}
				})
			})
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.assets) {
				return
			}
			ca := a.assets[id]
			item := co.(*widgets.Asset)
			item.Set(ca)
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
					if err := a.selectLocation(n); err != nil {
						slog.Warn("failed to select location", "error", "err")
					}
				}
			}
		} else {
			a.u.ShowTypeInfoWindow(ca.EveType.ID, a.u.CharacterID(), DescriptionTab)
		}
	}
	return g
}

func (a *AssetsArea) Redraw() {
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

func (a *AssetsArea) newLocationData() (*fynetree.FyneTree[locationDataNode], error) {
	ctx := context.TODO()
	tree := fynetree.New[locationDataNode]()
	if !a.u.HasCharacter() {
		return tree, nil
	}
	characterID := a.u.CharacterID()
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

func (a *AssetsArea) makeTopText(total int) (string, widget.Importance, error) {
	if !a.u.HasCharacter() {
		return "No character", widget.LowImportance, nil
	}
	hasData := a.u.StatusCacheService.CharacterSectionExists(a.u.CharacterID(), app.SectionAssets)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	locations := humanize.Comma(int64(total))
	return fmt.Sprintf("%s locations", locations), widget.MediumImportance, nil
}

func (a *AssetsArea) selectLocation(location locationDataNode) error {
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

func (a *AssetsArea) updateLocationPath(location locationDataNode) {
	path := make([]locationDataNode, 0)
	for _, uid := range a.locationsData.Path(location.UID()) {
		n, ok := a.locationsData.Value(uid)
		if !ok {
			continue
		}
		path = append(path, n)
	}
	path = append(path, location)
	parts := make([]string, 0)
	for _, n := range path {
		parts = append(parts, n.Name)
	}
	a.locationPath.SetText(strings.Join(parts, " ＞ "))
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
