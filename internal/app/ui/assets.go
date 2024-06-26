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
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/app/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
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

func (n locationDataNode) IsRoot() bool {
	return n.Type == nodeLocation
}

var defaultAssetIcon = theme.NewDisabledResource(resourceQuestionmarkSvg)

// assetsArea is the UI area that shows the skillqueue
type assetsArea struct {
	content         fyne.CanvasObject
	assets          *widget.GridWrap
	assetsData      binding.UntypedList
	assetsTop       *widget.Label
	locationsWidget *widget.Tree
	locationsData   *fynetree.FyneTree[locationDataNode]
	locationsTop    *widget.Label
	assetCollection assetcollection.AssetCollection
	ui              *ui
}

func (u *ui) newAssetsArea() *assetsArea {
	a := assetsArea{
		assetsData:    binding.NewUntypedList(),
		assetsTop:     widget.NewLabel(""),
		locationsData: fynetree.New[locationDataNode](),
		locationsTop:  widget.NewLabel(""),
		ui:            u,
	}
	a.locationsTop.TextStyle.Bold = true
	a.locationsWidget = a.makeLocationsTree()
	locations := container.NewBorder(container.NewVBox(a.locationsTop, widget.NewSeparator()), nil, nil, nil, a.locationsWidget)

	a.assetsTop.TextStyle.Bold = true
	a.assets = u.makeAssetGrid(a.assetsData)
	assets := container.NewBorder(container.NewVBox(a.assetsTop, widget.NewSeparator()), nil, nil, nil, a.assets)

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
			row := co.(*fyne.Container)
			prefix := row.Objects[0].(*widget.Label)
			label := row.Objects[1].(*widget.Label)
			n := a.locationsData.Value(uid)
			label.SetText(n.Name)
			if n.IsRoot() {
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
		n := a.locationsData.Value(uid)
		if n.IsRoot() {
			if !n.IsUnknown {
				a.ui.showLocationInfoWindow(n.ContainerID)
			}
			t.UnselectAll()
			return
		}
		if err := a.redrawAssets(n); err != nil {
			slog.Warn("Failed to redraw assets", "err", err)
		}
	}
	return t
}

func (a *assetsArea) redraw() {
	a.locationsWidget.CloseAllBranches()
	a.locationsWidget.ScrollToTop()
	t, i, err := func() (string, widget.Importance, error) {
		if err := a.clearAssets(); err != nil {
			return "", 0, err
		}
		total, err := a.updateLocationData()
		if err != nil {
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
	a.locationsWidget.Refresh()
}

func (a *assetsArea) updateLocationData() (int, error) {
	a.locationsData.Clear()
	if !a.ui.hasCharacter() {
		return 0, nil
	}
	characterID := a.ui.characterID()
	ctx := context.TODO()
	assets, err := a.ui.CharacterService.ListCharacterAssets(ctx, characterID)
	if err != nil {
		return 0, err
	}
	oo, err := a.ui.EveUniverseService.ListEveLocations(ctx)
	if err != nil {
		return 0, err
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
			Name:        makeNameWithCount(el.DisplayName(), len(ln.Nodes())),
		}
		if el.SolarSystem != nil {
			location.SystemName = el.SolarSystem.Name
			location.SystemSecurityValue = float32(el.SolarSystem.SecurityStatus)
			location.SystemSecurityType = el.SolarSystem.SecurityType()
		} else {
			location.IsUnknown = true
		}
		locationUID := a.locationsData.MustAdd("", location)

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
			Name:        makeNameWithCount("Ship Hangar", shipCount),
			Type:        nodeShipHangar,
		}
		shipsUID := a.locationsData.MustAdd(locationUID, shipHangar)
		for _, an := range ships {
			ship := an.Asset
			ldn := locationDataNode{
				CharacterID: characterID,
				ContainerID: an.Asset.ItemID,
				Name:        fmt.Sprintf("%s (%s)", ship.Name, ship.EveType.Name),
				Type:        nodeShip,
			}
			shipUID := a.locationsData.MustAdd(shipsUID, ldn)
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
				Name:        makeNameWithCount("Cargo Bay", len(cargo)),
				Type:        nodeCargoBay,
			}
			a.locationsData.MustAdd(shipUID, cln)
			if ship.EveType.HasFuelBay() {
				ldn := locationDataNode{
					CharacterID: characterID,
					ContainerID: an.Asset.ItemID,
					Name:        makeNameWithCount("Fuel Bay", len(fuel)),
					Type:        nodeFuelBay,
				}
				a.locationsData.MustAdd(shipUID, ldn)
			}
		}

		itemHangar := locationDataNode{
			CharacterID: characterID,
			ContainerID: el.ID,
			Name:        makeNameWithCount("Item Hangar", itemCount),
			Type:        nodeItemHangar,
		}
		itemsUID := a.locationsData.MustAdd(locationUID, itemHangar)
		for _, an := range itemContainers {
			ldn := locationDataNode{
				CharacterID: characterID,
				ContainerID: an.Asset.ItemID,
				Name:        makeNameWithCount(an.Asset.DisplayName(), len(an.Nodes())),
				Type:        nodeContainer,
			}
			a.locationsData.MustAdd(itemsUID, ldn)
		}

		if len(assetSafety) > 0 {
			an := assetSafety[0]
			ldn := locationDataNode{
				CharacterID: characterID,
				ContainerID: an.Asset.ItemID,
				Name:        makeNameWithCount("Asset Safety", len(an.Nodes())),
				Type:        nodeAssetSafety,
			}
			a.locationsData.MustAdd(locationUID, ldn)
		}
	}
	return len(a.assetCollection.Locations()), nil
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
	assets, err := f(context.TODO(), n.CharacterID, n.ContainerID)
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
		total += ca.Price.ValueOrZero() * float64(ca.Quantity)
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
	oo, err := u.CharacterService.ListCharacterAssetsInLocation(context.TODO(), ca.CharacterID, ca.ItemID)
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
			item.SetAsset(ca)
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
