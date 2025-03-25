package characterui

import (
	"cmp"
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type locationNodeVariant uint

const (
	nodeAssetSafety locationNodeVariant = iota + 1
	nodeCargoBay
	nodeContainer
	nodeDroneBay
	nodeFighterBay
	nodeFitting
	nodeFrigateEscapeBay
	nodeFuelBay
	nodeItemHangar
	nodeLocation
	nodeShip
	nodeShipHangar
	nodeShipOther
)

// locationNode is a node in the location tree.
type locationNode struct {
	characterID         int32
	containerID         int64
	name                string
	count               int
	isUnknown           bool
	systemName          string
	systemSecurityValue float32
	systemSecurityType  app.SolarSystemSecurityType
	variant             locationNodeVariant
}

func (n locationNode) UID() widget.TreeNodeID {
	if n.characterID == 0 || n.containerID == 0 || n.variant == 0 {
		panic("locationNode: some IDs are not set")
	}
	return fmt.Sprintf("%d-%d-%d", n.characterID, n.containerID, n.variant)
}

func (n locationNode) IsRoot() bool {
	return n.variant == nodeLocation
}

type CharacterAssets struct {
	widget.BaseWidget

	Locations      fyne.CanvasObject // TODO: Refactor into own widget
	LocationAssets fyne.CanvasObject // TODO: Refactor into own widget
	OnSelected     func()
	OnRedraw       func(string)

	assetCollection  assetcollection.AssetCollection
	assetGrid        *widget.GridWrap
	assets           []*app.CharacterAsset
	assetsBottom     *widget.Label
	locationPath     *kxwidget.TappableLabel
	locationsTop     *widget.Label
	locations        *iwidget.Tree[locationNode]
	selectedLocation optional.Optional[locationNode]
	u                app.UI
}

func NewCharacterAssets(u app.UI) *CharacterAssets {
	lp := kxwidget.NewTappableLabel("", nil)
	lp.Wrapping = fyne.TextWrapWord
	a := &CharacterAssets{
		assets:       make([]*app.CharacterAsset, 0),
		assetsBottom: widget.NewLabel(""),
		locationPath: lp,
		locationsTop: appwidget.MakeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.locations = a.makeLocationsTree()
	a.Locations = container.NewBorder(
		container.NewVBox(a.locationsTop, widget.NewSeparator()),
		nil,
		nil,
		nil,
		a.locations,
	)
	a.assetGrid = a.makeAssetGrid()
	a.LocationAssets = container.NewBorder(
		container.NewBorder(
			nil,
			widget.NewSeparator(),
			nil,
			widget.NewIcon(theme.InfoIcon()),
			a.locationPath,
		),
		container.NewVBox(widget.NewSeparator(), a.assetsBottom),
		nil,
		nil,
		a.assetGrid,
	)
	return a
}

func (a *CharacterAssets) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewHSplit(a.Locations, a.LocationAssets)
	c.SetOffset(0.33)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterAssets) makeLocationsTree() *iwidget.Tree[locationNode] {
	makeNameWithCount := func(name string, count int) string {
		if count == 0 {
			return name
		}
		return fmt.Sprintf("%s (%s)", name, humanize.Comma(int64(count)))
	}
	t := iwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			main := widget.NewLabel("Location")
			main.Truncation = fyne.TextTruncateEllipsis
			spacer := canvas.NewRectangle(color.Transparent)
			spacer.SetMinSize(fyne.NewSize(40, 10))
			return container.NewBorder(
				nil,
				nil,
				container.NewStack(spacer, widget.NewLabel("-9.9")),
				nil,
				main,
			)
		},
		func(n locationNode, isBranch bool, co fyne.CanvasObject) {
			row := co.(*fyne.Container).Objects
			label := row[0].(*widget.Label)
			spacer := row[1].(*fyne.Container).Objects[0]
			prefix := row[1].(*fyne.Container).Objects[1].(*widget.Label)
			label.SetText(makeNameWithCount(n.name, n.count))
			if n.IsRoot() {
				if !n.isUnknown {
					prefix.Text = fmt.Sprintf("%.1f", n.systemSecurityValue)
					prefix.Importance = n.systemSecurityType.ToImportance()
				} else {
					prefix.Text = "?"
					prefix.Importance = widget.LowImportance
				}
				prefix.Refresh()
				prefix.Show()
				spacer.Show()
			} else {
				prefix.Hide()
				spacer.Hide()
			}
		},
	)
	t.OnSelected = func(n locationNode) {
		if n.variant == nodeLocation {
			t.OpenBranch(n)
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

func (a *CharacterAssets) clearAssets() error {
	a.assets = make([]*app.CharacterAsset, 0)
	a.assetGrid.Refresh()
	a.locationPath.SetText("")
	a.locationPath.OnTapped = nil
	a.selectedLocation.Clear()
	return nil
}

func (a *CharacterAssets) makeAssetGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.assets)
		},
		func() fyne.CanvasObject {
			return NewCharacterAsset(func(image *canvas.Image, ca *app.CharacterAsset) {
				appwidget.RefreshImageResourceAsync(image, func() (fyne.Resource, error) {
					switch ca.Variant() {
					case app.VariantSKIN:
						return a.u.EveImageService().InventoryTypeSKIN(ca.EveType.ID, app.IconPixelSize)
					case app.VariantBPO:
						return a.u.EveImageService().InventoryTypeBPO(ca.EveType.ID, app.IconPixelSize)
					case app.VariantBPC:
						return a.u.EveImageService().InventoryTypeBPC(ca.EveType.ID, app.IconPixelSize)
					default:
						return a.u.EveImageService().InventoryTypeIcon(ca.EveType.ID, app.IconPixelSize)
					}
				})
			})
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.assets) {
				return
			}
			ca := a.assets[id]
			item := co.(*CharacterAsset)
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
			for _, uid := range a.locations.Data().ChildUIDs(location.UID()) {
				n, ok := a.locations.Data().Node(uid)
				if !ok {
					continue
				}
				if n.containerID == ca.ItemID {
					if err := a.selectLocation(n); err != nil {
						slog.Warn("failed to select location", "error", "err")
					}
				}
			}
		} else {
			a.u.ShowTypeInfoWindow(ca.EveType.ID)
		}
	}
	return g
}

func (a *CharacterAssets) Update() {
	a.locations.CloseAllBranches()
	a.locations.ScrollToTop()
	t, i, err := func() (string, widget.Importance, error) {
		if err := a.clearAssets(); err != nil {
			return "", 0, err
		}
		tree, err := a.makeLocationData()
		if err != nil {
			return "", 0, err
		}
		a.locations.Set(tree)
		locationsCount := len(tree.ChildUIDs(""))
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
	a.locations.Refresh()
	if a.OnRedraw != nil {
		a.OnRedraw(t)
	}
}

func (a *CharacterAssets) makeLocationData() (*iwidget.TreeData[locationNode], error) {
	tree := iwidget.NewTreeData[locationNode]()
	if !a.u.HasCharacter() {
		return tree, nil
	}
	characterID := a.u.CurrentCharacterID()
	ctx := context.Background()
	assets, err := a.u.CharacterService().ListCharacterAssets(ctx, characterID)
	if err != nil {
		return tree, err
	}
	oo, err := a.u.EveUniverseService().ListLocations(ctx)
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
		location := locationNode{
			characterID: characterID,
			containerID: el.ID,
			variant:     nodeLocation,
			name:        el.DisplayName(),
			count:       len(ln.Nodes()),
		}
		if el.SolarSystem != nil {
			location.systemName = el.SolarSystem.Name
			location.systemSecurityValue = float32(el.SolarSystem.SecurityStatus)
			location.systemSecurityType = el.SolarSystem.SecurityType()
		} else {
			location.isUnknown = true
		}
		locationUID := tree.MustAdd(iwidget.RootUID, location)

		topAssets := ln.Nodes()
		slices.SortFunc(topAssets, func(a, b assetcollection.AssetNode) int {
			return cmp.Compare(a.Asset.DisplayName(), b.Asset.DisplayName())
		})
		itemCount := 0
		shipCount := 0
		ships := make([]assetcollection.AssetNode, 0)
		itemContainers := make([]assetcollection.AssetNode, 0)
		assetSafety := make([]assetcollection.AssetNode, 0)
		for _, an := range topAssets {
			if an.Asset.InAssetSafety() {
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

		// ship hangar
		slices.SortFunc(ships, func(a, b assetcollection.AssetNode) int {
			return cmp.Compare(a.Asset.DisplayName2(), b.Asset.DisplayName2())
		})
		shipsUID := tree.MustAdd(locationUID, locationNode{
			characterID: characterID,
			containerID: el.ID,
			name:        "Ship Hangar",
			count:       shipCount,
			variant:     nodeShipHangar,
		})
		for _, an := range ships {
			ship := an.Asset
			shipUID := tree.MustAdd(shipsUID, locationNode{
				characterID: characterID,
				containerID: an.Asset.ItemID,
				name:        ship.DisplayName2(),
				variant:     nodeShip,
			})
			cargo := make([]assetcollection.AssetNode, 0)
			drones := make([]assetcollection.AssetNode, 0)
			fitting := make([]assetcollection.AssetNode, 0)
			frigate := make([]assetcollection.AssetNode, 0)
			fighters := make([]assetcollection.AssetNode, 0)
			fuel := make([]assetcollection.AssetNode, 0)
			other := make([]assetcollection.AssetNode, 0)
			for _, an2 := range an.Nodes() {
				switch {
				case an2.Asset.InCargoBay():
					cargo = append(cargo, an2)
				case an2.Asset.InDroneBay():
					drones = append(drones, an2)
				case an2.Asset.InFrigateEscapeBay():
					frigate = append(frigate, an2)
				case an2.Asset.IsInFuelBay():
					fuel = append(fuel, an2)
				case an2.Asset.InFighterBay():
					fighters = append(fighters, an2)
				case an2.Asset.IsFitted():
					fitting = append(fitting, an2)
				case an2.Asset.IsShipOther():
					other = append(other, an2)
				}
			}
			if n := len(fitting); n > 0 {
				tree.MustAdd(shipUID, locationNode{
					characterID: characterID,
					containerID: ship.ItemID,
					name:        "Fitting",
					count:       n,
					variant:     nodeFitting,
				})
			}
			if n := len(cargo); n > 0 {
				tree.MustAdd(shipUID, locationNode{
					characterID: characterID,
					containerID: ship.ItemID,
					name:        "Cargo Bay",
					count:       n,
					variant:     nodeCargoBay,
				})
			}
			if n := len(frigate); n > 0 {
				tree.MustAdd(shipUID, locationNode{
					characterID: characterID,
					containerID: ship.ItemID,
					name:        "Frigate Escape Bay",
					count:       n,
					variant:     nodeFrigateEscapeBay,
				})
			}
			if n := len(drones); n > 0 {
				tree.MustAdd(shipUID, locationNode{
					characterID: characterID,
					containerID: an.Asset.ItemID,
					name:        "Drone Bay",
					count:       n,
					variant:     nodeDroneBay,
				})
			}
			if n := len(fuel); n > 0 {
				tree.MustAdd(shipUID, locationNode{
					characterID: characterID,
					containerID: an.Asset.ItemID,
					name:        "Fuel Bay",
					count:       n,
					variant:     nodeFuelBay,
				})
			}
			if n := len(fighters); n > 0 {
				tree.MustAdd(shipUID, locationNode{
					characterID: characterID,
					containerID: an.Asset.ItemID,
					name:        "Fighter Bay",
					count:       n,
					variant:     nodeFighterBay,
				})
			}
			if n := len(other); n > 0 {
				tree.MustAdd(shipUID, locationNode{
					characterID: characterID,
					containerID: an.Asset.ItemID,
					name:        "Other",
					count:       n,
					variant:     nodeShipOther,
				})
			}
		}

		// item hangar
		itemsUID := tree.MustAdd(locationUID, locationNode{
			characterID: characterID,
			containerID: el.ID,
			name:        "Item Hangar",
			count:       itemCount,
			variant:     nodeItemHangar,
		})
		for _, an := range itemContainers {
			tree.MustAdd(itemsUID, locationNode{
				characterID: characterID,
				containerID: an.Asset.ItemID,
				name:        an.Asset.DisplayName(),
				count:       len(an.Nodes()),
				variant:     nodeContainer,
			})
		}

		// asset safety
		if len(assetSafety) > 0 {
			an := assetSafety[0]
			tree.MustAdd(locationUID, locationNode{
				characterID: characterID,
				containerID: an.Asset.ItemID,
				name:        "Asset Safety",
				count:       len(an.Nodes()),
				variant:     nodeAssetSafety,
			})
		}
	}
	return tree, nil
}

func (a *CharacterAssets) makeTopText(total int) (string, widget.Importance, error) {
	c := a.u.CurrentCharacter()
	if c == nil {
		return "No character", widget.LowImportance, nil
	}
	hasData := a.u.StatusCacheService().CharacterSectionExists(c.ID, app.SectionAssets)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance, nil
	}
	locations := humanize.Comma(int64(total))
	text := fmt.Sprintf("%s locations • %s total value", locations, ihumanize.OptionalFloat(c.AssetValue, 1, "?"))
	return text, widget.MediumImportance, nil
}

func (a *CharacterAssets) selectLocation(location locationNode) error {
	a.assets = make([]*app.CharacterAsset, 0)
	a.assetGrid.Refresh()
	a.selectedLocation.Set(location)
	selectedUID := location.UID()
	for _, uid := range a.locations.Data().Path(selectedUID) {
		n, ok := a.locations.Data().Node(uid)
		if !ok {
			continue
		}
		a.locations.OpenBranch(n)
	}
	a.locations.ScrollTo(location)
	a.locations.Select(location)
	var f func(context.Context, int32, int64) ([]*app.CharacterAsset, error)
	switch location.variant {
	case nodeShipHangar:
		f = a.u.CharacterService().ListCharacterAssetsInShipHangar
	case nodeItemHangar:
		f = a.u.CharacterService().ListCharacterAssetsInItemHangar
	default:
		f = a.u.CharacterService().ListCharacterAssetsInLocation
	}
	assets, err := f(context.Background(), location.characterID, location.containerID)
	if err != nil {
		return err
	}
	switch location.variant {
	case nodeCargoBay:
		s := make([]*app.CharacterAsset, 0)
		for _, it := range assets {
			if !it.InCargoBay() {
				continue
			}
			s = append(s, it)
		}
		assets = s
	case nodeDroneBay:
		s := make([]*app.CharacterAsset, 0)
		for _, it := range assets {
			if !it.InDroneBay() {
				continue
			}
			s = append(s, it)
		}
		assets = s
	case nodeFitting:
		s := make([]*app.CharacterAsset, 0)
		for _, it := range assets {
			if !it.IsFitted() {
				continue
			}
			s = append(s, it)
		}
		assets = s
	case nodeFrigateEscapeBay:
		s := make([]*app.CharacterAsset, 0)
		for _, it := range assets {
			if !it.InFrigateEscapeBay() {
				continue
			}
			s = append(s, it)
		}
		assets = s
	case nodeFighterBay:
		s := make([]*app.CharacterAsset, 0)
		for _, it := range assets {
			if !it.InFighterBay() {
				continue
			}
			s = append(s, it)
		}
		assets = s
	case nodeFuelBay:
		s := make([]*app.CharacterAsset, 0)
		for _, it := range assets {
			if !it.IsInFuelBay() {
				continue
			}
			s = append(s, it)
		}
		assets = s
	case nodeShipOther:
		s := make([]*app.CharacterAsset, 0)
		for _, it := range assets {
			if !it.IsShipOther() {
				continue
			}
			s = append(s, it)
		}
		assets = s
	case nodeItemHangar:
		containers := make([]*app.CharacterAsset, 0)
		s := make([]*app.CharacterAsset, 0)
		for _, it := range assets {
			if it.IsContainer() {
				containers = append(containers, it)
			} else {
				s = append(s, it)
			}
		}
		assets = slices.Concat(containers, s)
	}
	// slices.SortFunc(assets, func(a, b *app.CharacterAsset) int {
	// 	return cmp.Compare(a.DisplayName2(), b.DisplayName2())
	// })
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

func (a *CharacterAssets) updateLocationPath(location locationNode) {
	path := make([]locationNode, 0)
	for _, uid := range a.locations.Data().Path(location.UID()) {
		n, ok := a.locations.Data().Node(uid)
		if !ok {
			continue
		}
		path = append(path, n)
	}
	path = append(path, location)
	parts := make([]string, 0)
	for _, n := range path {
		parts = append(parts, n.name)
	}
	a.locationPath.SetText(strings.Join(parts, " ＞ "))
	a.locationPath.OnTapped = func() {
		if len(path) == 0 {
			return
		}
		a.u.ShowLocationInfoWindow(path[0].containerID)
	}
}

// func (u *ui) showNewAssetWindow(ca *app.CharacterAsset) {
// 	var name string
// 	if ca.Name != "" {
// 		name = fmt.Sprintf(" \"%s\" ", ca.Name)
// 	}
// 	title := fmt.Sprintf("%s%s(%s): Contents", ca.EveType.Name, name, ca.EveType.Group.Name)
// 	w := u.fyneApp.NewWindow(u.makeWindowTitle(title))
// 	oo, err := u.CharacterService().ListCharacterAssetsInLocation(context.TODO(), ca.CharacterID, ca.ItemID)
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
