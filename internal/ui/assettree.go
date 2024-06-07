package ui

import "github.com/ErikKalkoken/evebuddy/internal/model"

type assetNode struct {
	ca       *model.CharacterAsset
	children map[int64]assetNode
}

func newAssetNode(ca *model.CharacterAsset) assetNode {
	return assetNode{ca: ca, children: make(map[int64]assetNode)}
}

func newAssetTree(assets []*model.CharacterAsset) map[int64]assetNode {
	// initial map of all assets
	// assets will be removed from this map as they are added to the tree
	m := make(map[int64]*model.CharacterAsset)
	for _, ca := range assets {
		m[ca.ID] = ca
	}
	// create parent nodes
	nodes := make(map[int64]assetNode)
	for _, ca := range m {
		_, found := m[ca.LocationID]
		if !found {
			nodes[ca.ID] = newAssetNode(ca)
		}
	}
	for _, n := range nodes {
		delete(m, n.ca.ID)
	}
	// add child nodes
	addChildNodes(m, nodes)
	// return parent nodes
	return nodes
}

func addChildNodes(m map[int64]*model.CharacterAsset, nodes map[int64]assetNode) {
	for _, ca := range m {
		_, found := nodes[ca.LocationID]
		if found {
			nodes[ca.LocationID].children[ca.ID] = newAssetNode(ca)
			delete(m, ca.ID)
		}
	}
	for _, n := range nodes {
		if len(n.children) > 0 {
			addChildNodes(m, n.children)
		}
	}
}

type assetParentLocation struct {
	assetID    int64
	locationID int64
}

func collectAssetParentLocations(nodes map[int64]assetNode) []assetParentLocation {
	assets := make([]assetParentLocation, 0)
	// add parents
	for _, n := range nodes {
		assets = append(assets, assetParentLocation{assetID: n.ca.ID, locationID: n.ca.LocationID})
	}
	// add children
	assets = collectAssetChildrenLocations(assets, nodes, 0)
	return assets
}

func collectAssetChildrenLocations(assets []assetParentLocation, nodes map[int64]assetNode, parentLocationID int64) []assetParentLocation {
	for _, parent := range nodes {
		var parentLocationID2 int64
		if parentLocationID == 0 {
			parentLocationID2 = parent.ca.LocationID
		} else {
			parentLocationID2 = parentLocationID
		}
		if len(parent.children) > 0 {
			for _, child := range parent.children {
				assets = append(assets, assetParentLocation{assetID: child.ca.ID, locationID: parentLocationID2})
			}
			assets = collectAssetChildrenLocations(assets, parent.children, parentLocationID2)
		}
	}
	return assets
}
