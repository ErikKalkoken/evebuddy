// Package assettree allows building and working with assets in a tree structure.
package assettree

import "github.com/ErikKalkoken/evebuddy/internal/model"

// AssetNode is a node in a tree of character assets.
type AssetNode struct {
	ca       *model.CharacterAsset
	children map[int64]AssetNode
}

func newAssetNode(ca *model.CharacterAsset) AssetNode {
	return AssetNode{ca: ca, children: make(map[int64]AssetNode)}
}

// New returns a new asset tree from a slice of character assets.
func New(assets []*model.CharacterAsset) map[int64]AssetNode {
	// initial map of all assets
	// assets will be removed from this map as they are added to the tree
	m := make(map[int64]*model.CharacterAsset)
	for _, ca := range assets {
		m[ca.ItemID] = ca
	}
	// create parent nodes
	nodes := make(map[int64]AssetNode)
	for _, ca := range m {
		_, found := m[ca.LocationID]
		if !found {
			nodes[ca.ItemID] = newAssetNode(ca)
		}
	}
	for _, n := range nodes {
		delete(m, n.ca.ItemID)
	}
	// add child nodes
	addChildNodes(m, nodes)
	// return parent nodes
	return nodes
}

func addChildNodes(m map[int64]*model.CharacterAsset, nodes map[int64]AssetNode) {
	for _, ca := range m {
		_, found := nodes[ca.LocationID]
		if found {
			nodes[ca.LocationID].children[ca.ItemID] = newAssetNode(ca)
			delete(m, ca.ItemID)
		}
	}
	for _, n := range nodes {
		if len(n.children) > 0 {
			addChildNodes(m, n.children)
		}
	}
}

// CompileParentLocations returns a map of asset ID to parent location ID
func CompileParentLocations(nodes map[int64]AssetNode) map[int64]int64 {
	assets := make(map[int64]int64)
	// add parents
	for _, n := range nodes {
		assets[n.ca.ItemID] = n.ca.LocationID
	}
	// add children
	addAssetChildrenLocations(assets, nodes, 0)
	return assets
}

func addAssetChildrenLocations(assets map[int64]int64, nodes map[int64]AssetNode, realParentLocationID int64) {
	for _, parent := range nodes {
		var parentLocationID int64
		if realParentLocationID == 0 {
			parentLocationID = parent.ca.LocationID
		} else {
			parentLocationID = realParentLocationID
		}
		if len(parent.children) > 0 {
			for _, child := range parent.children {
				assets[child.ca.ItemID] = parentLocationID
			}
			addAssetChildrenLocations(assets, parent.children, parentLocationID)
		}
	}
}
