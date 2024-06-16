package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assettree"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/dustin/go-humanize"
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

// locationDataTree represents a tree of location nodes for rendering with the tree widget.
type locationDataTree struct {
	ids    map[string][]string
	values map[string]locationDataNode
}

func newLocationDataTree() locationDataTree {
	ltd := locationDataTree{
		values: make(map[string]locationDataNode),
		ids:    make(map[string][]string),
	}
	return ltd
}

func (ltd locationDataTree) add(parentUID string, node locationDataNode) string {
	if parentUID != "" {
		_, found := ltd.values[parentUID]
		if !found {
			panic(fmt.Sprintf("parent UID does not exist: %s", parentUID))
		}
	}
	uid := node.UID()
	_, found := ltd.values[uid]
	if found {
		panic(fmt.Sprintf("UID for this node already exists: %v", node))
	}
	ltd.ids[parentUID] = append(ltd.ids[parentUID], uid)
	ltd.values[uid] = node
	return uid
}

func (ltd locationDataTree) stringTree() (map[string][]string, map[string]string, error) {
	values := make(map[string]string)
	for id, node := range ltd.values {
		v, err := objectToJSON(node)
		if err != nil {
			return nil, nil, err
		}
		values[id] = v
	}
	return ltd.ids, values, nil
}

// locationDataNode is a node for the asset tree widget.
type locationDataNode struct {
	CharacterID         int32
	ContainerID         int64
	Name                string
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

func (n locationDataNode) isBranch() bool {
	return n.Type == nodeLocation
}

var defaultAssetIcon = theme.NewDisabledResource(resourceQuestionmarkSvg)

// assetsArea is the UI area that shows the skillqueue
type assetsArea struct {
	content       fyne.CanvasObject
	assets        *widget.GridWrap
	assetsData    binding.UntypedList
	assetsTop     *widget.Label
	locations     *widget.Tree
	locationsData binding.StringTree
	locationsTop  *widget.Label

	assetTree assettree.AssetTree

	ui *ui
}

func (u *ui) newAssetsArea() *assetsArea {
	a := assetsArea{
		assetsData:    binding.NewUntypedList(),
		assetsTop:     widget.NewLabel(""),
		locationsData: binding.NewStringTree(),
		locationsTop:  widget.NewLabel(""),
		ui:            u,
	}
	a.locationsTop.TextStyle.Bold = true
	a.locations = a.makeLocationsTree()
	locations := container.NewBorder(container.NewVBox(a.locationsTop, widget.NewSeparator()), nil, nil, nil, a.locations)

	a.assetsTop.TextStyle.Bold = true
	a.assets = u.makeAssetGrid(a.assetsData)
	assets := container.NewBorder(container.NewVBox(a.assetsTop, widget.NewSeparator()), nil, nil, nil, a.assets)

	main := container.NewHSplit(locations, assets)
	main.SetOffset(0.33)
	a.content = main
	return &a
}

func (a *assetsArea) makeLocationsTree() *widget.Tree {
	t := widget.NewTreeWithData(
		a.locationsData,
		func(branch bool) fyne.CanvasObject {
			prefix := widget.NewLabel("1.0")
			prefix.Importance = widget.HighImportance
			return container.NewHBox(prefix, widget.NewLabel("Location"))
		},
		func(di binding.DataItem, branch bool, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			prefix := row.Objects[0].(*widget.Label)
			label := row.Objects[1].(*widget.Label)
			n, err := treeNodeFromDataItem[locationDataNode](di)
			if err != nil {
				slog.Error("Failed to render asset location in UI", "err", err)
				label.SetText("ERROR")
				return
			}
			label.SetText(n.Name)
			if n.isBranch() {
				if !n.IsUnknown {
					prefix.Text = fmt.Sprintf("%.1f", n.SystemSecurityValue)
					prefix.Importance = systemSecurity2Importance(n.SystemSecurityType)
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
		n, err := treeNodeFromBoundTree[locationDataNode](a.locationsData, uid)
		if err != nil {
			slog.Error("Failed to select location", "err", err)
			t.UnselectAll()
			return
		}
		if n.isBranch() {
			t.ToggleBranch(uid)
			t.UnselectAll()
			return
		}
		if err := a.redrawAssets(n); err != nil {
			slog.Warn("Failed to redraw assets", "err", err)
			t.UnselectAll()
		}
	}
	return t
}

func (a *assetsArea) redraw() {
	a.locations.CloseAllBranches()
	a.locations.ScrollToTop()
	t, i, err := func() (string, widget.Importance, error) {
		if err := a.clearAssets(); err != nil {
			return "", 0, err
		}
		tree, total, err := a.createTreeData()
		if err != nil {
			return "", 0, err
		}
		ids, values, err := tree.stringTree()
		if err != nil {
			return "", 0, err
		}
		if err := a.locationsData.Set(ids, values); err != nil {
			return "", 0, err
		}
		return a.makeTopText(total)
	}()
	if err != nil {
		slog.Error("Failed to redraw asset locations UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.locationsTop.Text = t
	a.locationsTop.Importance = i
	a.locationsTop.Refresh()
}

func (a *assetsArea) createTreeData() (locationDataTree, int, error) {
	tree := newLocationDataTree()
	if !a.ui.hasCharacter() {
		return tree, 0, nil
	}
	characterID := a.ui.characterID()
	ctx := context.Background()
	assets, err := a.ui.CharacterService.ListCharacterAssets(ctx, characterID)
	if err != nil {
		return tree, 0, err
	}
	oo, err := a.ui.EveUniverseService.ListEveLocations(ctx)
	if err != nil {
		return tree, 0, err
	}
	a.assetTree = assettree.New(assets, oo)
	locationNodes := a.assetTree.Locations()
	slices.SortFunc(locationNodes, func(a assettree.LocationNode, b assettree.LocationNode) int {
		return cmp.Compare(a.Location.DisplayName(), b.Location.DisplayName())
	})
	for _, ln := range locationNodes {
		el := ln.Location
		location := locationDataNode{
			CharacterID: characterID,
			ContainerID: el.ID,
			Type:        nodeLocation,
			Name:        makeNameWithCount(el.DisplayName(), len(ln.Nodes())),
		}
		if el.SolarSystem != nil {
			location.SystemName = el.SolarSystem.Name
			location.SystemSecurityValue = float32(el.SolarSystem.SecurityStatus)
			location.SystemSecurityType = el.SolarSystem.SecurityType()
		} else {
			location.IsUnknown = true
		}
		locationUID := tree.add("", location)

		topAssets := ln.Nodes()
		slices.SortFunc(topAssets, func(a assettree.AssetNode, b assettree.AssetNode) int {
			return cmp.Compare(a.Asset.DisplayName(), b.Asset.DisplayName())
		})
		itemCount := 0
		shipCount := 0
		ships := make([]assettree.AssetNode, 0)
		itemContainers := make([]assettree.AssetNode, 0)
		assetSafety := make([]assettree.AssetNode, 0)
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
			Name:        makeNameWithCount("Ship Hangar", shipCount),
			Type:        nodeShipHangar,
		}
		shipsUID := tree.add(locationUID, shipHangar)
		for _, an := range ships {
			ship := an.Asset
			ldn := locationDataNode{
				CharacterID: characterID,
				ContainerID: an.Asset.ItemID,
				Name:        fmt.Sprintf("%s (%s)", ship.Name, ship.EveType.Name),
				Type:        nodeShip,
			}
			shipUID := tree.add(shipsUID, ldn)
			cargo := make([]assettree.AssetNode, 0)
			fuel := make([]assettree.AssetNode, 0)
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
				Name:        makeNameWithCount("Cargo Bay", len(cargo)),
				Type:        nodeCargoBay,
			}
			tree.add(shipUID, cln)
			if ship.EveType.HasFuelBay() {
				ldn := locationDataNode{
					CharacterID: characterID,
					ContainerID: an.Asset.ItemID,
					Name:        makeNameWithCount("Fuel Bay", len(fuel)),
					Type:        nodeFuelBay,
				}
				tree.add(shipUID, ldn)
			}
		}

		itemHangar := locationDataNode{
			CharacterID: characterID,
			ContainerID: el.ID,
			Name:        makeNameWithCount("Item Hangar", itemCount),
			Type:        nodeItemHangar,
		}
		itemsUID := tree.add(locationUID, itemHangar)
		for _, an := range itemContainers {
			ldn := locationDataNode{
				CharacterID: characterID,
				ContainerID: an.Asset.ItemID,
				Name:        makeNameWithCount(an.Asset.Name, len(an.Nodes())),
				Type:        nodeContainer,
			}
			tree.add(itemsUID, ldn)
		}

		if len(assetSafety) > 0 {
			an := assetSafety[0]
			ldn := locationDataNode{
				CharacterID: characterID,
				ContainerID: an.Asset.ItemID,
				Name:        makeNameWithCount("Asset Safety", len(an.Nodes())),
				Type:        nodeAssetSafety,
			}
			tree.add(locationUID, ldn)
		}
	}
	return tree, len(a.assetTree.Locations()), nil
}

func (a *assetsArea) makeTopText(total int) (string, widget.Importance, error) {
	if !a.ui.hasCharacter() {
		return "No character", widget.LowImportance, nil
	}
	hasData := a.ui.StatusCacheService.CharacterSectionExists(a.ui.characterID(), app.SectionAssets)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	return fmt.Sprintf("%d locations", total), widget.MediumImportance, nil
}

func (a *assetsArea) redrawAssets(n locationDataNode) error {
	empty := make([]*app.CharacterAsset, 0)
	if err := a.assetsData.Set(copyToUntypedSlice(empty)); err != nil {
		return err
	}
	var f func(context.Context, int32, int64) ([]*app.CharacterAsset, error)
	switch n.Type {
	case nodeShipHangar:
		f = a.ui.CharacterService.ListCharacterAssetsInShipHangar
	case nodeItemHangar:
		f = a.ui.CharacterService.ListCharacterAssetsInItemHangar
	default:
		f = a.ui.CharacterService.ListCharacterAssetsInLocation
	}
	assets, err := f(context.Background(), n.CharacterID, n.ContainerID)
	if err != nil {
		return err
	}
	switch n.Type {
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
	if err := a.assetsData.Set(copyToUntypedSlice(assets)); err != nil {
		return err
	}
	var total float64
	for _, ca := range assets {
		total += ca.Price.Float64 * float64(ca.Quantity)
	}
	a.assetsTop.SetText(fmt.Sprintf("%d Items - %s ISK Est. Price", len(assets), ihumanize.Number(total, 1)))
	return nil
}

func (a *assetsArea) clearAssets() error {
	empty := make([]*app.CharacterAsset, 0)
	if err := a.assetsData.Set(copyToUntypedSlice(empty)); err != nil {
		return err
	}
	a.assetsTop.SetText("")
	return nil
}

func (u *ui) showNewAssetWindow(ca *app.CharacterAsset) {
	var name string
	if ca.Name != "" {
		name = fmt.Sprintf(" \"%s\" ", ca.Name)
	}
	title := fmt.Sprintf("%s%s(%s): Contents", ca.EveType.Name, name, ca.EveType.Group.Name)
	w := u.fyneApp.NewWindow(title)
	oo, err := u.CharacterService.ListCharacterAssetsInLocation(context.Background(), ca.CharacterID, ca.ItemID)
	if err != nil {
		panic(err)
	}
	data := binding.NewUntypedList()
	if err := data.Set(copyToUntypedSlice(oo)); err != nil {
		panic(err)
	}
	top := widget.NewLabel(fmt.Sprintf("%s items", humanize.Comma(int64(len(oo)))))
	top.TextStyle.Bold = true
	assets := u.makeAssetGrid(data)
	content := container.NewBorder(container.NewVBox(top, widget.NewSeparator()), nil, nil, nil, assets)

	w.SetContent(content)
	w.Resize(fyne.Size{Width: 650, Height: 500})
	w.Show()
}

func (u *ui) makeAssetGrid(assetsData binding.UntypedList) *widget.GridWrap {
	g := widget.NewGridWrapWithData(
		assetsData,
		func() fyne.CanvasObject {
			return widgets.NewAssetListWidget(u.EveImageService, defaultAssetIcon)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			ca, err := convertDataItem[*app.CharacterAsset](di)
			if err != nil {
				panic(err)
			}
			item := co.(*widgets.AssetListWidget)
			item.SetAsset(ca.DisplayName(), ca.Quantity, ca.IsSingleton, ca.EveType.ID,
				widgetTypeVariantFromModel(ca.Variant()))
		},
	)
	g.OnSelected = func(id widget.GridWrapItemID) {
		ca, err := getItemUntypedList[*app.CharacterAsset](assetsData, id)
		if err != nil {
			slog.Error("failed to access assets in grid", "err", err)
			return
		}
		if ca.IsContainer() {
			u.showNewAssetWindow(ca)
		} else {
			u.showTypeInfoWindow(ca.EveType.ID, u.characterID())
		}
		g.UnselectAll()
	}
	return g
}

func makeNameWithCount(name string, count int) string {
	if count == 0 {
		return name
	}
	return fmt.Sprintf("%s (%s)", name, humanize.Comma(int64(count)))
}

func widgetTypeVariantFromModel(v app.EveTypeVariant) widgets.EveTypeVariant {
	m := map[app.EveTypeVariant]widgets.EveTypeVariant{
		app.VariantBPC:     widgets.VariantBPC,
		app.VariantBPO:     widgets.VariantBPO,
		app.VariantRegular: widgets.VariantRegular,
		app.VariantSKIN:    widgets.VariantSKIN,
	}
	return m[v]
}
