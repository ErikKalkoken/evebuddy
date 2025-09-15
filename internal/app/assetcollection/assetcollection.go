// Package assetcollection provides data structures to analyze and process asset data.
package assetcollection

import (
	"maps"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// AssetCollection is a collection of assets and their locations.
// The assets are structured as one asset tree per location
// and may contain assets belonging to one or multiple characters.
type AssetCollection struct {
	assetLocations map[int64]*LocationNode
	assets         map[int64]*AssetNode
	locations      map[int64]*LocationNode
}

// New returns a new AssetCollection from assets and locations.
func New(ca []*app.CharacterAsset, loc []*app.EveLocation) AssetCollection {
	locationMap := make(map[int64]*app.EveLocation)
	for _, loc := range loc {
		locationMap[loc.ID] = loc
	}
	// initial map of all assetMap
	// assetMap will be removed from this map as they are added to the tree
	assetMap := make(map[int64]*app.CharacterAsset)
	for _, ca := range ca {
		assetMap[ca.ItemID] = ca
	}
	locations := make(map[int64]*LocationNode)
	for _, ca := range assetMap {
		_, found := assetMap[ca.LocationID]
		if found {
			continue
		}
		loc, found := locationMap[ca.LocationID]
		if !found {
			continue
		}
		locations[loc.ID] = newLocationNode(loc)
	}
	// add top assets to locations
	assets := make(map[int64]*AssetNode)
	for _, ca := range assetMap {
		location, found := locations[ca.LocationID]
		if !found {
			continue
		}
		an := location.add(ca)
		assets[ca.ItemID] = an
		delete(assetMap, ca.ItemID)
	}
	// add others assets to locations
	for _, l := range locations {
		addChildNodes(l.children, assetMap, assets)
	}
	ac := AssetCollection{
		assetLocations: gatherParentLocations(locations),
		assets:         assets,
		locations:      locations,
	}
	return ac
}

// addChildNodes adds assets as nodes to parents. Recursive.
func addChildNodes(parents map[int64]*AssetNode, assetMap map[int64]*app.CharacterAsset, assets map[int64]*AssetNode) {
	for _, ca := range assetMap {
		_, found := parents[ca.LocationID]
		if found {
			an := parents[ca.LocationID].add(ca)
			assets[ca.ItemID] = an
			delete(assetMap, ca.ItemID)
		}
	}
	for _, n := range parents {
		if len(n.children) > 0 {
			addChildNodes(n.children, assetMap, assets)
		}
	}
}

// gatherParentLocations returns the mapping of asset ID to parent location ID
func gatherParentLocations(locations map[int64]*LocationNode) map[int64]*LocationNode {
	m := make(map[int64]*LocationNode)
	for _, ln := range locations {
		for _, n := range ln.children {
			for _, c := range n.All() {
				m[c.Asset.ItemID] = ln
			}
		}
	}
	return m
}

// AssetLocation returns the location node for an asset.
func (ac AssetCollection) AssetLocation(itemID int64) (*LocationNode, bool) {
	ln, found := ac.assetLocations[itemID]
	if !found {
		return nil, false
	}
	return ln, true
}

// Asset returns the node for an asset and reports whether it was found.
func (ac AssetCollection) Asset(itemID int64) (*AssetNode, bool) {
	an, found := ac.assets[itemID]
	if !found {
		return nil, false
	}
	return an, true
}

// ItemCountFiltered returns the consolidated count of all items excluding items inside ships containers.
func (ac AssetCollection) ItemCountFiltered() int {
	var n int
	for _, l := range ac.locations {
		n += l.ItemCountFiltered()
	}
	return n
}

// Locations returns a slice of all location nodes.
func (ac AssetCollection) Locations() []*LocationNode {
	nn := make([]*LocationNode, 0)
	for _, ln := range ac.locations {
		nn = append(nn, ln)
	}
	return nn
}

// Location returns the node for a location and reports whether it was found.
func (ac AssetCollection) Location(locationID int64) (*LocationNode, bool) {
	ln, found := ac.locations[locationID]
	if !found {
		return nil, false
	}
	return ln, true
}

type baseNode struct {
	children       map[int64]*AssetNode
	totalItemCount int // item count summary of all nodes
}

func newBaseNode() baseNode {
	return baseNode{children: make(map[int64]*AssetNode)}
}

// Children returns the child nodes of the current node.
func (bn baseNode) Children() []*AssetNode {
	return slices.Collect(maps.Values(bn.children))
}

func (bn baseNode) add(ca *app.CharacterAsset) *AssetNode {
	n := newAssetNode(ca)
	bn.children[ca.ItemID] = n
	return n
}

// Size returns the size of a sub-tree starting with the node as root.
func (bn baseNode) Size() int {
	var counter func(nodes map[int64]*AssetNode) int
	counter = func(nodes map[int64]*AssetNode) int {
		var count int
		for _, n := range nodes {
			if len(n.children) == 0 {
				count++
			} else {
				count += counter(n.children)
			}
		}
		return count
	}
	return counter(bn.children)
}

// AssetNode is a node in an asset tree representing an asset, e.g. a ship.
type AssetNode struct {
	baseNode
	Asset *app.CharacterAsset
}

func newAssetNode(ca *app.CharacterAsset) *AssetNode {
	return &AssetNode{Asset: ca, baseNode: newBaseNode()}
}

// ItemCountFiltered returns the consolidated count of items in this sub tree
// excluding items that are inside a ship container, e.g. fittings.
func (an AssetNode) ItemCountFiltered() int {
	if an.Asset.IsFitted() ||
		an.Asset.IsInDroneBay() ||
		an.Asset.IsInFrigateEscapeBay() ||
		an.Asset.IsInFighterBay() ||
		an.Asset.IsInFuelBay() ||
		an.Asset.IsInAnyCargoHold() {
		return 0
	}
	q := int(an.Asset.Quantity)
	for _, c := range an.Children() {
		q += c.ItemCountFiltered()
	}
	return q
}

// All returns all nodes of a sub tree (order is undefined)
func (an *AssetNode) All() []*AssetNode {
	s := make([]*AssetNode, 0)
	s = append(s, an)
	for _, c := range an.Children() {
		s = slices.Concat(s, c.All())
	}
	return s
}

// ItemCountAny returns the consolidated count of any items in this sub tree.
func (an AssetNode) ItemCountAny() int {
	q := int(an.Asset.Quantity)
	for _, c := range an.Children() {
		q += c.ItemCountAny()
	}
	return q
}

// LocationNode is the root node in an asset tree representing a location, e.g. a station.
type LocationNode struct {
	baseNode
	Location *app.EveLocation
}

func newLocationNode(l *app.EveLocation) *LocationNode {
	return &LocationNode{Location: l, baseNode: newBaseNode()}
}

// ItemCountFiltered returns the consolidated count of items in this location
// excluding items inside ships containers.
func (ln LocationNode) ItemCountFiltered() int {
	var q int
	for _, c := range ln.Children() {
		q += c.ItemCountFiltered()
	}
	return q
}
