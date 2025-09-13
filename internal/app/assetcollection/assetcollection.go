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
	lns                  map[int64]LocationNode
	assetParentLocations map[int64]int64
}

// New returns a new AssetCollection from assets and locations.
func New(ca []*app.CharacterAsset, loc []*app.EveLocation) AssetCollection {
	locations := make(map[int64]*app.EveLocation)
	for _, loc := range loc {
		locations[loc.ID] = loc
	}
	// initial map of all assets
	// assets will be removed from this map as they are added to the tree
	assets := make(map[int64]*app.CharacterAsset)
	for _, ca := range ca {
		assets[ca.ItemID] = ca
	}
	locationNodes := make(map[int64]LocationNode)
	for _, ca := range assets {
		_, found := assets[ca.LocationID]
		if found {
			continue
		}
		loc, found := locations[ca.LocationID]
		if !found {
			continue
		}
		locationNodes[loc.ID] = newLocationNode(loc)
	}
	// create parent nodes
	for _, ca := range assets {
		location, found := locationNodes[ca.LocationID]
		if !found {
			continue
		}
		location.add(ca)
		delete(assets, ca.ItemID)
	}
	for _, l := range locationNodes {
		addChildNodes(l.children, assets)
	}
	ac := AssetCollection{
		lns:                  locationNodes,
		assetParentLocations: gatherParentLocations(locationNodes),
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

func (at AssetCollection) AssetParentLocation(id int64) (*app.EveLocation, bool) {
	id, found := at.assetParentLocations[id]
	if !found {
		return nil, false
	}
	ln, found := at.lns[id]
	if !found {
		return nil, false
	}
	return ln.Location, true
}

func (at AssetCollection) Locations() []LocationNode {
	nn := make([]LocationNode, 0)
	for _, ln := range at.lns {
		nn = append(nn, ln)
	}
	return nn
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

// LocationNode is the root node in an asset tree representing a location, e.g. a station.
type LocationNode struct {
	baseNode
	Location *app.EveLocation
}

func newLocationNode(l *app.EveLocation) LocationNode {
	return LocationNode{Location: l, baseNode: newBaseNode()}
}
