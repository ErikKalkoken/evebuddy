// Package asset provides data structures to analyze and process asset data.
package asset

import (
	"maps"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type Item interface {
	ID() int64
	LocationID_() int64
	Quantity_() int
	QuantityFiltered() (int, bool)
}

// Collection is a collection of asset trees.
//
// The assets are structured as one asset tree per location
// and may contain assets belonging to one or multiple characters.
type Collection struct {
	assetLocations map[int64]*LocationNode
	assets         map[int64]*ItemNode
	locations      map[int64]*LocationNode
}

// New returns a new Tree from assets and locations.
func New(items []Item, loc []*app.EveLocation) Collection {
	locationMap := make(map[int64]*app.EveLocation)
	for _, loc := range loc {
		locationMap[loc.ID] = loc
	}
	// initial map of all assetMap
	// assetMap will be removed from this map as they are added to the tree
	assetMap := make(map[int64]Item)
	for _, it := range items {
		assetMap[it.ID()] = it
	}
	locations := make(map[int64]*LocationNode)
	for _, it := range assetMap {
		_, found := assetMap[it.LocationID_()]
		if found {
			continue
		}
		loc, found := locationMap[it.LocationID_()]
		if !found {
			continue
		}
		locations[loc.ID] = newLocationNode(loc)
	}
	// add top assets to locations
	assets := make(map[int64]*ItemNode)
	for _, it := range assetMap {
		location, found := locations[it.LocationID_()]
		if !found {
			continue
		}
		an := location.add(it)
		assets[it.ID()] = an
		delete(assetMap, it.ID())
	}
	// add others assets to locations
	for _, l := range locations {
		addChildNodes(l.children, assetMap, assets)
	}
	ac := Collection{
		assetLocations: gatherParentLocations(locations),
		assets:         assets,
		locations:      locations,
	}
	return ac
}

// addChildNodes adds assets as nodes to parents. Recursive.
func addChildNodes(parents map[int64]*ItemNode, assetMap map[int64]Item, assets map[int64]*ItemNode) {
	for _, it := range assetMap {
		_, found := parents[it.LocationID_()]
		if found {
			an := parents[it.LocationID_()].add(it)
			assets[it.ID()] = an
			delete(assetMap, it.ID())
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
				m[c.it.ID()] = ln
			}
		}
	}
	return m
}

// AssetLocation returns the location node for an asset.
func (ac Collection) AssetLocation(itemID int64) (*LocationNode, bool) {
	ln, found := ac.assetLocations[itemID]
	if !found {
		return nil, false
	}
	return ln, true
}

// Asset returns the node for an asset and reports whether it was found.
func (ac Collection) Asset(itemID int64) (*ItemNode, bool) {
	an, found := ac.assets[itemID]
	if !found {
		return nil, false
	}
	return an, true
}

// ItemCountFiltered returns the consolidated count of all items excluding items inside ships containers.
func (ac Collection) ItemCountFiltered() int {
	var n int
	for _, l := range ac.locations {
		n += l.ItemCountFiltered()
	}
	return n
}

// Locations returns a slice of all location nodes.
func (ac Collection) Locations() []*LocationNode {
	nn := make([]*LocationNode, 0)
	for _, ln := range ac.locations {
		nn = append(nn, ln)
	}
	return nn
}

// Location returns the node for a location and reports whether it was found.
func (ac Collection) Location(locationID int64) (*LocationNode, bool) {
	ln, found := ac.locations[locationID]
	if !found {
		return nil, false
	}
	return ln, true
}

type baseNode struct {
	children       map[int64]*ItemNode
	totalItemCount int // item count summary of all nodes
}

func newBaseNode() baseNode {
	return baseNode{children: make(map[int64]*ItemNode)}
}

// Children returns the child nodes of the current node.
func (bn baseNode) Children() []*ItemNode {
	return slices.Collect(maps.Values(bn.children))
}

func (bn baseNode) add(it Item) *ItemNode {
	n := newAssetNode(it)
	bn.children[it.ID()] = n
	return n
}

// Size returns the size of a sub-tree starting with the node as root.
func (bn baseNode) Size() int {
	var counter func(nodes map[int64]*ItemNode) int
	counter = func(nodes map[int64]*ItemNode) int {
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

// ItemNode is a node in an asset tree representing an asset, e.g. a ship.
type ItemNode struct {
	baseNode
	it Item
}

func newAssetNode(ai Item) *ItemNode {
	return &ItemNode{it: ai, baseNode: newBaseNode()}
}

func (in ItemNode) Item() Item {
	return in.it
}

// MustCharacterAsset returns the current item as character asset.
// Will panic if the item has a different type.
func (in ItemNode) MustCharacterAsset() *app.CharacterAsset {
	return in.it.(*app.CharacterAsset)
}

// ItemCountFiltered returns the consolidated count of items in this sub tree
// excluding items that are inside a ship container, e.g. fittings.
func (in ItemNode) ItemCountFiltered() int {
	q, ok := in.it.QuantityFiltered()
	if !ok {
		return 0
	}
	for _, c := range in.Children() {
		q += c.ItemCountFiltered()
	}
	return q
}

// All returns all nodes of a sub tree (order is undefined)
func (in *ItemNode) All() []*ItemNode {
	s := make([]*ItemNode, 0)
	s = append(s, in)
	for _, c := range in.Children() {
		s = slices.Concat(s, c.All())
	}
	return s
}

// ItemCountAny returns the consolidated count of any items in this sub tree.
func (in ItemNode) ItemCountAny() int {
	q := in.it.Quantity_()
	for _, c := range in.Children() {
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

func ItemsFromCharacterAssets(assets []*app.CharacterAsset) []Item {
	items := make([]Item, 0)
	for _, ca := range assets {
		items = append(items, ca)
	}
	return items
}

func ItemsFromCorporationAssets(assets []*app.CorporationAsset) []Item {
	items := make([]Item, 0)
	for _, ca := range assets {
		items = append(items, ca)
	}
	return items
}
