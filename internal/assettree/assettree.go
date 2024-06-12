// Package assettree allows working with character assets in a tree structure.
package assettree

import (
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

// AssetNode is a node in an asset tree representing an asset, e.g. a ship.
type AssetNode struct {
	Asset    *model.CharacterAsset
	Children map[int64]AssetNode
}

func newAssetNode(ca *model.CharacterAsset) AssetNode {
	return AssetNode{Asset: ca, Children: make(map[int64]AssetNode)}
}

// LocationNode is the root node of an asset tree for a location, e.g. a station.
type LocationNode struct {
	Location *model.EveLocation
	Children map[int64]AssetNode
}

func newLocationNode(l *model.EveLocation) LocationNode {
	return LocationNode{Location: l, Children: make(map[int64]AssetNode)}
}

type AssetTree struct {
	Locations            map[int64]LocationNode
	assetParentLocations map[int64]int64
}

// New returns a new asset tree from a slice of character assets.
func New(assets []*model.CharacterAsset, locations []*model.EveLocation) AssetTree {
	// initial map of all assets
	// assets will be removed from this map as they are added to the tree
	lm := make(map[int64]*model.EveLocation)
	for _, loc := range locations {
		lm[loc.ID] = loc
	}
	am := make(map[int64]*model.CharacterAsset)
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
		location.Children[ca.ItemID] = newAssetNode(ca)
		delete(am, ca.ItemID)
	}
	for _, l := range lns {
		// add child nodes
		addChildNodes(am, l.Children)
	}
	// return parent nodes
	parentLocations := gatherParentLocations(lns)
	return AssetTree{Locations: lns, assetParentLocations: parentLocations}
}

func addChildNodes(m map[int64]*model.CharacterAsset, nodes map[int64]AssetNode) {
	for _, ca := range m {
		_, found := nodes[ca.LocationID]
		if found {
			nodes[ca.LocationID].Children[ca.ItemID] = newAssetNode(ca)
			delete(m, ca.ItemID)
		}
	}
	for _, n := range nodes {
		if len(n.Children) > 0 {
			addChildNodes(m, n.Children)
		}
	}
}

// gatherParentLocations returns the mapping of asset ID to parent location ID
func gatherParentLocations(locations map[int64]LocationNode) map[int64]int64 {
	assetLocations := make(map[int64]int64)
	// add parents
	for id, ln := range locations {
		for _, n := range ln.Children {
			assetLocations[n.Asset.ItemID] = id
		}
		// add children
		addAssetChildrenLocations(assetLocations, ln.Children, 0)
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
		if len(parent.Children) > 0 {
			for _, child := range parent.Children {
				assets[child.Asset.ItemID] = parentLocationID
			}
			addAssetChildrenLocations(assets, parent.Children, parentLocationID)
		}
	}
}

func (at AssetTree) AssetParentLocation(id int64) (*model.EveLocation, bool) {
	id, found := at.assetParentLocations[id]
	if !found {
		return nil, false
	}
	ln, found := at.Locations[id]
	if !found {
		return nil, false
	}
	return ln.Location, true
}
