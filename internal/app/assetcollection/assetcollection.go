// Package assetcollection provides data structures to analyze and process asset data.
package assetcollection

import (
	"maps"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type baseNode struct {
	nodes map[int64]AssetNode
}

func newBaseNode() baseNode {
	return baseNode{nodes: make(map[int64]AssetNode)}
}
func (bn baseNode) Nodes() []AssetNode {
	return slices.Collect(maps.Values(bn.nodes))
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

// AssetCollection is a collection of assets.
// The assets are structured as one asset tree per location
// and may contain assets belonging to one or multiple characters.
type AssetCollection struct {
	lns                  map[int64]LocationNode
	assetParentLocations map[int64]int64
}

// New returns a new AssetCollection from a slice of character assets.
func New(assets []*app.CharacterAsset, locations []*app.EveLocation) AssetCollection {
	// initial map of all assets
	// assets will be removed from this map as they are added to the tree
	lm := make(map[int64]*app.EveLocation)
	for _, loc := range locations {
		lm[loc.ID] = loc
	}
	am := make(map[int64]*app.CharacterAsset)
	for _, ca := range assets {
		am[ca.ItemID] = ca
	}
	lns := make(map[int64]LocationNode)
	for _, ca := range am {
		_, found := am[ca.LocationID]
		if found {
			continue
		}
		loc, found := lm[ca.LocationID]
		if !found {
			continue
		}
		lns[loc.ID] = newLocationNode(loc)
	}
	// create parent nodes
	for _, ca := range am {
		location, found := lns[ca.LocationID]
		if !found {
			continue
		}
		location.nodes[ca.ItemID] = newAssetNode(ca)
		delete(am, ca.ItemID)
	}
	for _, l := range lns {
		// add child nodes
		addChildNodes(am, l.nodes)
	}
	// return parent nodes
	parentLocations := gatherParentLocations(lns)
	return AssetCollection{lns: lns, assetParentLocations: parentLocations}
}

func addChildNodes(m map[int64]*app.CharacterAsset, nodes map[int64]AssetNode) {
	for _, ca := range m {
		_, found := nodes[ca.LocationID]
		if found {
			nodes[ca.LocationID].nodes[ca.ItemID] = newAssetNode(ca)
			delete(m, ca.ItemID)
		}
	}
	for _, n := range nodes {
		if len(n.nodes) > 0 {
			addChildNodes(m, n.nodes)
		}
	}
}

// gatherParentLocations returns the mapping of asset ID to parent location ID
func gatherParentLocations(locations map[int64]LocationNode) map[int64]int64 {
	assetLocations := make(map[int64]int64)
	// add parents
	for id, ln := range locations {
		for _, n := range ln.nodes {
			assetLocations[n.Asset.ItemID] = id
		}
		// add children
		addAssetChildrenLocations(assetLocations, ln.nodes, 0)
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
		if len(parent.nodes) > 0 {
			for _, child := range parent.nodes {
				assets[child.Asset.ItemID] = parentLocationID
			}
			addAssetChildrenLocations(assets, parent.nodes, parentLocationID)
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
