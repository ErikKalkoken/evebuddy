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

	"github.com/ErikKalkoken/evebuddy/internal/assettree"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/helper/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/dustin/go-humanize"
)

type locationNodeType uint

const (
	nodeLocation locationNodeType = iota + 1
	nodeShipHangar
	nodeItemHangar
	nodeContainer
	nodeShip
	nodeCargoBay
	nodeFuelBay
	nodeAssetSafety
)

// locationNode is a node for the asset tree widget.
type locationNode struct {
	CharacterID         int32
	ContainerID         int64
	Name                string
	IsUnknown           bool
	SystemName          string
	SystemSecurityValue float32
	SystemSecurityType  model.SolarSystemSecurityType
	Type                locationNodeType
}

func (n locationNode) UID() widget.TreeNodeID {
	if n.CharacterID == 0 || n.ContainerID == 0 || n.Type == 0 {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%d-%d", n.CharacterID, n.ContainerID, n.Type)
}

func (n locationNode) isBranch() bool {
	return n.Type == nodeLocation
}

func (n locationNode) addToTree(parentUID string, ids map[string][]string, values map[string]string) string {
	uid := n.UID()
	ids[parentUID] = append(ids[parentUID], uid)
	values[uid] = objectToJSONOrPanic(n)
	return uid
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
			n, err := treeNodeFromDataItem[locationNode](di)
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
		n, err := treeNodeFromBoundTree[locationNode](a.locationsData, uid)
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
		ids, values, total, err := a.updateTreeData()
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

func (a *assetsArea) updateTreeData() (map[string][]string, map[string]string, int, error) {
	values := make(map[string]string)
	ids := make(map[string][]string)
	if !a.ui.hasCharacter() {
		return ids, values, 0, nil
	}
	characterID := a.ui.characterID()
	ctx := context.Background()
	assets, err := a.ui.sv.Character.ListCharacterAssets(ctx, characterID)
	if err != nil {
		return nil, nil, 0, err
	}
	locations, err := a.ui.sv.EveUniverse.ListEveLocations(ctx)
	if err != nil {
		return nil, nil, 0, err
	}
	a.assetTree = assettree.New(assets, locations)
	ll := a.assetTree.Locations()
	slices.SortFunc(ll, func(a assettree.LocationNode, b assettree.LocationNode) int {
		return cmp.Compare(a.Location.DisplayName(), b.Location.DisplayName())
	})
	for _, atl := range ll {
		loc := atl.Location
		ln := locationNode{
			CharacterID: characterID,
			ContainerID: loc.ID,
			Type:        nodeLocation,
			Name:        makeNameWithCount(loc.DisplayName(), len(atl.Nodes())),
		}
		if loc.SolarSystem != nil {
			ln.SystemName = loc.SolarSystem.Name
			ln.SystemSecurityValue = float32(loc.SolarSystem.SecurityStatus)
			ln.SystemSecurityType = loc.SolarSystem.SecurityType()
		} else {
			ln.IsUnknown = true
		}
		locationUID := ln.addToTree("", ids, values)

		topAssets := atl.Nodes()
		slices.SortFunc(topAssets, func(a assettree.AssetNode, b assettree.AssetNode) int {
			return cmp.Compare(a.Asset.DisplayName(), b.Asset.DisplayName())
		})
		ships := make([]assettree.AssetNode, 0)
		itemContainers := make([]assettree.AssetNode, 0)
		assetSafety := make([]assettree.AssetNode, 0)
		for _, an := range topAssets {
			if an.Asset.IsInAssetSafety() {
				assetSafety = append(assetSafety, an)
			} else if an.Asset.IsContainer() {
				if an.Asset.IsShip() {
					ships = append(ships, an)
				} else {
					itemContainers = append(itemContainers, an)
				}
			}
		}

		nsh := makeHangarNode(nodeShipHangar, ln.ContainerID, len(ships), characterID)
		shipsUID := nsh.addToTree(locationUID, ids, values)
		for _, an := range ships {
			ship := an.Asset
			ln := locationNode{
				CharacterID: characterID,
				ContainerID: an.Asset.ItemID,
				Name:        fmt.Sprintf("%s (%s)", ship.Name, ship.EveType.Name),
				Type:        nodeShip,
			}
			shipUID := ln.addToTree(shipsUID, ids, values)
			cargo := make([]assettree.AssetNode, 0)
			fuel := make([]assettree.AssetNode, 0)
			for _, an2 := range an.Nodes() {
				if an2.Asset.IsInCargoBay() {
					cargo = append(cargo, an2)
				} else if an2.Asset.IsInFuelBay() {
					fuel = append(fuel, an2)
				}
			}
			cln := locationNode{
				CharacterID: characterID,
				ContainerID: ship.ItemID,
				Name:        makeNameWithCount("Cargo Bay", len(cargo)),
				Type:        nodeCargoBay,
			}
			cln.addToTree(shipUID, ids, values)
			if ship.EveType.HasFuelBay() {
				fln := locationNode{
					CharacterID: characterID,
					ContainerID: an.Asset.ItemID,
					Name:        makeNameWithCount("Fuel Bay", len(fuel)),
					Type:        nodeFuelBay,
				}
				fln.addToTree(shipUID, ids, values)
			}
		}

		itemsCount := len(topAssets) - len(ships) - len(assetSafety)
		nih := makeHangarNode(nodeItemHangar, ln.ContainerID, itemsCount, characterID)
		itemsUID := nih.addToTree(locationUID, ids, values)
		for _, an := range itemContainers {
			ln := locationNode{
				CharacterID: characterID,
				ContainerID: an.Asset.ItemID,
				Name:        makeNameWithCount(an.Asset.Name, len(an.Nodes())),
				Type:        nodeContainer,
			}
			ln.addToTree(itemsUID, ids, values)
		}

		if len(assetSafety) > 0 {
			ln := locationNode{
				CharacterID: characterID,
				ContainerID: ln.ContainerID,
				Name:        makeNameWithCount("Asset Safety", len(assetSafety)),
				Type:        nodeAssetSafety,
			}
			ln.addToTree(locationUID, ids, values)

		}
	}
	return ids, values, len(a.assetTree.Locations()), nil
}

func makeHangarNode(t locationNodeType, locationID int64, n int, characterID int32) locationNode {
	var name string
	switch t {
	case nodeShipHangar:
		name = "Ship Hangar"
	case nodeItemHangar:
		name = "Item Hangar"
	}
	hn := locationNode{
		CharacterID: characterID,
		ContainerID: locationID,
		Name:        makeNameWithCount(name, n),
		Type:        t,
	}
	return hn
}

func (a *assetsArea) makeTopText(total int) (string, widget.Importance, error) {
	if !a.ui.hasCharacter() {
		return "No character", widget.LowImportance, nil
	}
	hasData, err := a.ui.sv.Character.SectionWasUpdated(
		context.Background(), a.ui.characterID(), model.SectionAssets)
	if err != nil {
		return "", 0, err
	}
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	return fmt.Sprintf("%d locations", total), widget.MediumImportance, nil
}

func (a *assetsArea) redrawAssets(n locationNode) error {
	empty := make([]*model.CharacterAsset, 0)
	if err := a.assetsData.Set(copyToUntypedSlice(empty)); err != nil {
		return err
	}
	var f func(context.Context, int32, int64) ([]*model.CharacterAsset, error)
	switch n.Type {
	case nodeShipHangar:
		f = a.ui.sv.Character.ListCharacterAssetsInShipHangar
	case nodeItemHangar:
		f = a.ui.sv.Character.ListCharacterAssetsInItemHangar
	default:
		f = a.ui.sv.Character.ListCharacterAssetsInLocation
	}
	assets, err := f(context.Background(), n.CharacterID, n.ContainerID)
	if err != nil {
		return err
	}
	switch n.Type {
	case nodeCargoBay:
		cargo := make([]*model.CharacterAsset, 0)
		for _, ca := range assets {
			if !ca.IsInCargoBay() {
				continue
			}
			cargo = append(cargo, ca)
		}
		assets = cargo
	case nodeFuelBay:
		fuel := make([]*model.CharacterAsset, 0)
		for _, ca := range assets {
			if !ca.IsInFuelBay() {
				continue
			}
			fuel = append(fuel, ca)
		}
		assets = fuel
	case nodeAssetSafety:
		xx := make([]*model.CharacterAsset, 0)
		for _, ca := range assets {
			if !ca.IsInAssetSafety() {
				continue
			}
			xx = append(xx, ca)
		}
		assets = xx
	}
	if err := a.assetsData.Set(copyToUntypedSlice(assets)); err != nil {
		return err
	}
	var total float64
	for _, ca := range assets {
		total += ca.Price.Float64
	}
	a.assetsTop.SetText(fmt.Sprintf("%d Items - %s ISK Est. Price", len(assets), ihumanize.Number(total, 1)))
	return nil
}

func (a *assetsArea) clearAssets() error {
	empty := make([]*model.CharacterAsset, 0)
	if err := a.assetsData.Set(copyToUntypedSlice(empty)); err != nil {
		return err
	}
	a.assetsTop.SetText("")
	return nil
}

func (u *ui) showNewAssetWindow(ca *model.CharacterAsset) {
	w := u.app.NewWindow(fmt.Sprintf("%s \"%s\" (%s): Contents", ca.EveType.Name, ca.Name, ca.EveType.Group.Name))
	oo, err := u.sv.Character.ListCharacterAssetsInLocation(context.Background(), ca.CharacterID, ca.ItemID)
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
			return NewAssetListWidget(u.sv.EveImage, defaultAssetIcon)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			ca, err := convertDataItem[*model.CharacterAsset](di)
			if err != nil {
				panic(err)
			}
			item := co.(*AssetListWidget)
			item.SetAsset(ca.DisplayName(), ca.Quantity, ca.IsSingleton, ca.EveType.ID, ca.Variant())
		},
	)
	g.OnSelected = func(id widget.GridWrapItemID) {
		ca, err := getItemUntypedList[*model.CharacterAsset](assetsData, id)
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
