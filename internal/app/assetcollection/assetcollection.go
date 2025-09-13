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
	locations            map[int64]LocationNode
	assetParentLocations map[int64]int64
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
	locations := make(map[int64]LocationNode)
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
	// create parent nodes
	for _, ca := range assetMap {
		location, found := locations[ca.LocationID]
		if !found {
			continue
		}
		location.add(ca)
		delete(assetMap, ca.ItemID)
	}
	for _, l := range locations {
		addChildNodes(l.children, assetMap)
	}
	ac := AssetCollection{
		locations:            locations,
		assetParentLocations: gatherParentLocations(locations),
	}
	return ac
}

// addChildNodes adds assets as nodes to parents. Recursive.
func addChildNodes(parents map[int64]AssetNode, assets map[int64]*app.CharacterAsset) {
	for _, ca := range assets {
		_, found := parents[ca.LocationID]
		if found {
			parents[ca.LocationID].add(ca)
			delete(assets, ca.ItemID)
		}
	}
	for _, n := range parents {
		if len(n.children) > 0 {
			addChildNodes(n.children, assets)
		}
	}
}

// gatherParentLocations returns the mapping of asset ID to parent location ID
func gatherParentLocations(locations map[int64]LocationNode) map[int64]int64 {
	assetLocations := make(map[int64]int64)
	// add parents
	for id, ln := range locations {
		for _, n := range ln.children {
			assetLocations[n.Asset.ItemID] = id
		}
		// add children
		addAssetChildrenLocations(assetLocations, ln.children, 0)
	}
	return assetLocations
}

func addAssetChildrenLocations(assets map[int64]int64, nodes map[int64]AssetNode, realParentLocationID int64) {
	for _, parent := range nodes {
		var parentLocationID int64
		if realParentLocationID == 0 {
			parentLocationID = parent.Asset.LocationID
		} else {
			parentLocationID = realParentLocationID
		}
		if len(parent.children) > 0 {
			for _, child := range parent.children {
				assets[child.Asset.ItemID] = parentLocationID
			}
			addAssetChildrenLocations(assets, parent.children, parentLocationID)
		}
	}
}

func (ac AssetCollection) AssetParentLocation(itemID int64) (*app.EveLocation, bool) {
	itemID, found := ac.assetParentLocations[itemID]
	if !found {
		return nil, false
	}
	ln, found := ac.locations[itemID]
	if !found {
		return nil, false
	}
	return ln.Location, true
}

func (ac AssetCollection) Locations() []LocationNode {
	nn := make([]LocationNode, 0)
	for _, ln := range ac.locations {
		nn = append(nn, ln)
	}
	return nn
}

func (ac AssetCollection) Location(locationID int64) (LocationNode, bool) {
	ln, found := ac.locations[locationID]
	if !found {
		return LocationNode{}, false
	}
	return ln, true
}

type baseNode struct {
	children       map[int64]AssetNode
	totalItemCount int // item count summary of all nodes
}

func newBaseNode() baseNode {
	return baseNode{children: make(map[int64]AssetNode)}
}

// Children returns the child nodes of the current node.
func (bn baseNode) Children() []AssetNode {
	return slices.Collect(maps.Values(bn.children))
}

func (bn baseNode) add(ca *app.CharacterAsset) AssetNode {
	n := newAssetNode(ca)
	bn.children[ca.ItemID] = n
	return n
}

// Size returns the size of a sub-tree starting with the node as root.
func (bn baseNode) Size() int {
	var counter func(nodes map[int64]AssetNode) int
	counter = func(nodes map[int64]AssetNode) int {
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

func newAssetNode(ca *app.CharacterAsset) AssetNode {
	return AssetNode{Asset: ca, baseNode: newBaseNode()}
}

func (an AssetNode) ItemCount() int {
	q := int(an.Asset.Quantity)
	for _, c := range an.Children() {
		q += c.ItemCount()
	}
	return q
}

// LocationNode is the root node in an asset tree representing a location, e.g. a station.
type LocationNode struct {
	baseNode
	Location *app.EveLocation
}

func newLocationNode(l *app.EveLocation) LocationNode {
	return LocationNode{Location: l, baseNode: newBaseNode()}
}

func (ln LocationNode) ItemCount() int {
	var q int
	for _, c := range ln.Children() {
		q += c.ItemCount()
	}
	return q
}
