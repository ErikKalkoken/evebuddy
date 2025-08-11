package ui

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
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/assetcollection"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type characterAssets struct {
	widget.BaseWidget

	Locations      fyne.CanvasObject // TODO: Refactor into own widget
	LocationAssets fyne.CanvasObject // TODO: Refactor into own widget
	OnSelected     func()
	OnRedraw       func(string)

	assetCollection  assetcollection.AssetCollection
	assetGrid        *widget.GridWrap
	assets           []*app.CharacterAsset
	assetsBottom     *widget.Label
	infoIcon         *widget.Icon
	locationPath     *kxwidget.TappableLabel
	locations        *iwidget.Tree[locationNode]
	locationsTop     *widget.Label
	selectedLocation optional.Optional[locationNode]
	u                *baseUI
}

func newCharacterAssets(u *baseUI) *characterAssets {
	lp := kxwidget.NewTappableLabel("", nil)
	lp.Wrapping = fyne.TextWrapWord
	a := &characterAssets{
		assets:       make([]*app.CharacterAsset, 0),
		assetsBottom: widget.NewLabel(""),
		locationPath: lp,
		locationsTop: makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.infoIcon = widget.NewIcon(theme.InfoIcon())
	a.infoIcon.Hide()
	a.locations = a.makeLocationsTree()
	a.Locations = container.NewBorder(
		a.locationsTop,
		nil,
		nil,
		nil,
		a.locations,
	)
	a.assetGrid = a.makeAssetGrid()
	a.LocationAssets = container.NewBorder(
		container.NewBorder(
			nil,
			nil,
			nil,
			a.infoIcon,
			a.locationPath,
		),
		a.assetsBottom,
		nil,
		nil,
		a.assetGrid,
	)
	return a
}

func (a *characterAssets) CreateRenderer() fyne.WidgetRenderer {
	main := container.NewHSplit(a.Locations, a.LocationAssets)
	main.SetOffset(0.33)
	p := theme.Padding()
	c := container.NewBorder(
		widget.NewSeparator(),
		nil,
		nil,
		nil,
		container.New(layout.NewCustomPaddedLayout(-p, 0, 0, 0), main),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *characterAssets) makeLocationsTree() *iwidget.Tree[locationNode] {
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
			label.SetText(n.displayName())
			if n.isRoot() {
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
	t.OnSelectedNode = func(n locationNode) {
		if n.variant == nodeLocation {
			t.OpenBranch(n.UID())
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

func (a *characterAssets) makeAssetGrid() *widget.GridWrap {
	g := widget.NewGridWrap(
		func() int {
			return len(a.assets)
		},
		func() fyne.CanvasObject {
			return newAssetItem(func(image *canvas.Image, ca *app.CharacterAsset) {
				iwidget.RefreshImageAsync(image, func() (fyne.Resource, error) {
					switch ca.Variant() {
					case app.VariantSKIN:
						return a.u.eis.InventoryTypeSKIN(ca.Type.ID, app.IconPixelSize)
					case app.VariantBPO:
						return a.u.eis.InventoryTypeBPO(ca.Type.ID, app.IconPixelSize)
					case app.VariantBPC:
						return a.u.eis.InventoryTypeBPC(ca.Type.ID, app.IconPixelSize)
					default:
						return a.u.eis.InventoryTypeIcon(ca.Type.ID, app.IconPixelSize)
					}
				})
			})
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			if id >= len(a.assets) {
				return
			}
			ca := a.assets[id]
			item := co.(*assetItem)
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
			showAssetDetailWindow(a.u, newAssetRow(ca, a.assetCollection, a.u.scs.CharacterName))
		}
	}
	return g
}

func (a *characterAssets) update() {
	fyne.Do(func() {
		a.locations.CloseAllBranches()
		a.locations.ScrollToTop()
	})
	t, i, err := func() (string, widget.Importance, error) {
		fyne.Do(func() {
			a.assets = make([]*app.CharacterAsset, 0)
			a.assetGrid.Refresh()
			a.locationPath.SetText("")
			a.locationPath.OnTapped = nil
			a.selectedLocation.Clear()
			a.infoIcon.Hide()
		})
		ac, locations, err := a.fetchData(a.u.currentCharacterID(), a.u.services())
		if err != nil {
			return "", 0, err
		}
		fyne.Do(func() {
			a.assetCollection = ac
			a.locations.Set(locations)
		})
		locationsCount := len(locations.ChildUIDs(""))
		t, i := a.makeTopText(locationsCount)
		return t, i, nil
	}()
	if err != nil {
		slog.Error("Failed to redraw asset locations UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	fyne.Do(func() {
		a.locationsTop.Text = t
		a.locationsTop.Importance = i
		a.locationsTop.Refresh()
		a.locations.Refresh()
	})
	if a.OnRedraw != nil {
		c := a.u.currentCharacter()
		if c != nil {
			s := ihumanize.OptionalWithDecimals(c.AssetValue, 1, "?")
			a.OnRedraw(s)
		}
	}
}

func (*characterAssets) fetchData(characterID int32, s services) (assetcollection.AssetCollection, iwidget.TreeData[locationNode], error) {
	var ac assetcollection.AssetCollection
	var locations iwidget.TreeData[locationNode]
	if characterID == 0 {
		return ac, locations, nil
	}
	ctx := context.Background()
	assets, err := s.cs.ListAssets(ctx, characterID)
	if err != nil {
		return ac, locations, err
	}
	el, err := s.eus.ListLocations(ctx)
	if err != nil {
		return ac, locations, err
	}
	ac = assetcollection.New(assets, el)
	locationNodes := ac.Locations()
	slices.SortFunc(locationNodes, func(x assetcollection.LocationNode, y assetcollection.LocationNode) int {
		return cmp.Compare(x.Location.DisplayName(), y.Location.DisplayName())
	})
	locations = makeLocationTreeData(locationNodes, characterID)
	return ac, locations, nil
}

func makeLocationTreeData(locationNodes []assetcollection.LocationNode, characterID int32) iwidget.TreeData[locationNode] {
	var tree iwidget.TreeData[locationNode]
	for _, ln := range locationNodes {
		location := locationNode{
			characterID: characterID,
			containerID: ln.Location.ID,
			variant:     nodeLocation,
			name:        ln.Location.DisplayName(),
			count:       ln.Size(),
		}
		if ln.Location.SolarSystem != nil {
			location.systemName = ln.Location.SolarSystem.Name
			location.systemSecurityValue = float32(ln.Location.SolarSystem.SecurityStatus)
			location.systemSecurityType = ln.Location.SolarSystem.SecurityType()
		} else {
			location.isUnknown = true
		}
		locationUID := tree.MustAdd(iwidget.TreeRootID, location)
		topAssets := ln.Nodes()
		slices.SortFunc(topAssets, func(a, b assetcollection.AssetNode) int {
			return cmp.Compare(a.Asset.DisplayName(), b.Asset.DisplayName())
		})
		itemCount := 0
		shipCount := 0
		ships := make([]assetcollection.AssetNode, 0)
		itemContainers := make([]assetcollection.AssetNode, 0)
		assetSafety := make([]assetcollection.AssetNode, 0)
		inSpace := make([]assetcollection.AssetNode, 0)
		for _, an := range topAssets {
			if an.Asset.InAssetSafety() {
				assetSafety = append(assetSafety, an)
			} else if an.Asset.IsInHangar() {
				if an.Asset.Type.IsShip() {
					shipCount++
				} else {
					itemCount++
				}
				if an.Asset.IsContainer() {
					if an.Asset.Type.IsShip() {
						ships = append(ships, an)
					} else {
						itemContainers = append(itemContainers, an)
					}
				}
			} else {
				inSpace = append(inSpace, an)
			}
		}

		// ship hangar
		slices.SortFunc(ships, func(a, b assetcollection.AssetNode) int {
			return cmp.Compare(a.Asset.DisplayName2(), b.Asset.DisplayName2())
		})
		shipsUID := tree.MustAdd(locationUID, locationNode{
			characterID: characterID,
			containerID: ln.Location.ID,
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
			containerID: ln.Location.ID,
			name:        "Item Hangar",
			count:       itemCount,
			variant:     nodeItemHangar,
		})
		for _, an := range itemContainers {
			tree.MustAdd(itemsUID, locationNode{
				characterID: characterID,
				containerID: an.Asset.ItemID,
				name:        an.Asset.DisplayName(),
				count:       an.Size(),
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
				count:       an.Size(),
				variant:     nodeAssetSafety,
			})
		}

		// items in space
		if len(inSpace) > 0 {
			tree.MustAdd(locationUID, locationNode{
				characterID: characterID,
				containerID: ln.Location.ID,
				name:        "In Space",
				count:       len(inSpace),
				variant:     nodeInSpace,
			})
		}
	}
	return tree
}

func (a *characterAssets) makeTopText(total int) (string, widget.Importance) {
	c := a.u.currentCharacter()
	if c == nil {
		return "No character", widget.LowImportance
	}
	hasData := a.u.scs.HasCharacterSection(c.ID, app.SectionCharacterAssets)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	locations := humanize.Comma(int64(total))
	text := fmt.Sprintf("%s locations • %s total value", locations, ihumanize.OptionalWithDecimals(c.AssetValue, 1, "?"))
	return text, widget.MediumImportance
}

func (a *characterAssets) selectLocation(location locationNode) error {
	a.assets = make([]*app.CharacterAsset, 0)
	a.assetGrid.Refresh()
	a.selectedLocation.Set(location)
	selectedUID := location.UID()
	for _, uid := range a.locations.Data().Path(selectedUID) {
		n, ok := a.locations.Data().Node(uid)
		if !ok {
			continue
		}
		a.locations.OpenBranch(n.UID())
	}
	a.locations.ScrollTo(location.UID())
	a.locations.Select(location.UID())
	var f func(context.Context, int32, int64) ([]*app.CharacterAsset, error)
	switch location.variant {
	case nodeShipHangar:
		f = a.u.cs.ListAssetsInShipHangar
	case nodeItemHangar:
		f = a.u.cs.ListAssetsInItemHangar
	default:
		f = a.u.cs.ListAssetsInLocation
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
	case nodeInSpace:
		s := make([]*app.CharacterAsset, 0)
		for _, it := range assets {
			if !it.IsInSpace() {
				continue
			}
			s = append(s, it)
		}
		assets = s
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

func (a *characterAssets) updateLocationPath(location locationNode) {
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
	a.infoIcon.Show()
}

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
	nodeInSpace
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
	if n.characterID == 0 || n.variant == 0 {
		panic("locationNode: some IDs are not set")
	}
	return fmt.Sprintf("%d-%d-%d", n.characterID, n.containerID, n.variant)
}

func (n locationNode) displayName() string {
	return n.name

	// FIXME
	// if n.count == 0 || n.variant == nodeShipHangar {
	// 	return n.name
	// }
	// return fmt.Sprintf("%s - %s Items", n.name, humanize.Comma(int64(n.count)))
}

func (n locationNode) isRoot() bool {
	return n.variant == nodeLocation
}

const (
	typeIconSize                      = 55
	sizeLabelText                     = 12
	colorAssetQuantityBadgeBackground = theme.ColorNameMenuBackground
	labelMaxCharacters                = 10
)

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

type assetItem struct {
	widget.BaseWidget

	badge      *assetQuantityBadge
	icon       *canvas.Image
	iconLoader func(*canvas.Image, *app.CharacterAsset)
	label      *assetLabel
}

func newAssetItem(iconLoader func(image *canvas.Image, ca *app.CharacterAsset)) *assetItem {
	icon := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(typeIconSize))
	w := &assetItem{
		icon:       icon,
		label:      newAssetLabel(),
		iconLoader: iconLoader,
		badge:      newAssetQuantityBadge(),
	}
	w.badge.Hide()
	w.ExtendBaseWidget(w)
	return w
}

func (o *assetItem) Set(ca *app.CharacterAsset) {
	o.label.SetText(ca.DisplayName())
	if !ca.IsSingleton {
		o.badge.SetQuantity(int(ca.Quantity))
		o.badge.Show()
	} else {
		o.badge.Hide()
	}
	o.iconLoader(o.icon, ca)
}

func (o *assetItem) CreateRenderer() fyne.WidgetRenderer {
	customVBox := layout.NewCustomPaddedVBoxLayout(0)
	c := container.NewPadded(container.New(
		customVBox,
		container.New(NewBottomRightLayout(), o.icon, o.badge),
		o.label,
	))
	return widget.NewSimpleRenderer(c)
}

type bottomRightLayout struct{}

func NewBottomRightLayout() fyne.Layout {
	return &bottomRightLayout{}
}

func (d *bottomRightLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, h := float32(0), float32(0)
	for _, o := range objects {
		childSize := o.MinSize()
		if childSize.Width > w {
			w = childSize.Width
		}
		if childSize.Height > h {
			h = childSize.Height
		}
	}
	return fyne.NewSize(w, h)
}

func (d *bottomRightLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
	pos := fyne.NewPos(containerSize.Width, containerSize.Height)
	for _, o := range objects {
		size := o.MinSize()
		o.Resize(size)
		o.Move(pos.SubtractXY(size.Width, size.Height))
	}
}
