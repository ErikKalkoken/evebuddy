// Package asset provides data structures to analyze and process asset data.
package asset

import (
	"fmt"
	"maps"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

type Item interface {
	ID() int64
	Unwrap() app.Asset
}

// Collection represents a collection of asset trees.
// There is one tree for each location.
type Collection struct {
	isCorporation bool            // True for corporation assets, false for character assets.
	trees         map[int64]*Node // Map of location IDs to root nodes.
	nodeLookup    map[int64]*Node // Lookup of nodes for items
	rootLookup    map[int64]*Node // Lookup of root nodes for items
}

func NewFromCharacterAssets(assets []*app.CharacterAsset, locations []*app.EveLocation) Collection {
	items := make([]Item, 0)
	for _, ca := range assets {
		items = append(items, ca)
	}
	return new(items, locations, false)
}

func NewFromCorporationAssets(assets []*app.CorporationAsset, locations []*app.EveLocation) Collection {
	items := make([]Item, 0)
	for _, ca := range assets {
		items = append(items, ca)
	}
	return new(items, locations, true)
}

// new returns a new Collection.
func new(items []Item, locations []*app.EveLocation, isCorporation bool) Collection {
	// map of eve locationLookup
	locationLookup := make(map[int64]*app.EveLocation)
	for _, loc := range locations {
		locationLookup[loc.ID] = loc
	}

	// initial map of all items
	// items will be removed from this map as they are added to the trees
	items2 := make(map[int64]Item)
	for _, it := range items {
		items2[it.ID()] = it
	}

	// create tree roots
	trees := make(map[int64]*Node)
	for _, it := range items2 {
		asset := it.Unwrap()
		_, found := items2[asset.LocationID]
		if found {
			continue
		}
		loc, found := locationLookup[asset.LocationID]
		if !found {
			continue
		}
		trees[loc.ID] = newLocationNode(loc)
	}

	// add items to trees and make node lookup
	nodeLookup := make(map[int64]*Node)
	for _, it := range items2 {
		root, found := trees[it.Unwrap().LocationID]
		if !found {
			continue
		}
		node := root.addChildFromItem(it)
		nodeLookup[it.ID()] = node
		delete(items2, it.ID())
	}
	for _, root := range trees {
		addChildNodes(root.children, items2, nodeLookup)
	}

	// make root lookup
	rootLookup := make(map[int64]*Node)
	for _, root := range trees {
		for _, n := range root.children {
			for _, c := range n.All() {
				rootLookup[c.item.ID()] = root
			}
		}
	}

	insertCustomNodes(trees)
	addMissingOffices(trees)

	ac := Collection{
		isCorporation: isCorporation,
		nodeLookup:    nodeLookup,
		rootLookup:    rootLookup,
		trees:         trees,
	}
	return ac
}

// addChildNodes adds assets as nodes to parents. Recursive.
func addChildNodes(parents []*Node, items2 map[int64]Item, nodeLookup map[int64]*Node) {
	// lookup table
	parents2 := make(map[int64]*Node)
	for _, n := range parents {
		parents2[n.ID()] = n
	}

	// add items to matching parents
	for _, it := range items2 {
		asset := it.Unwrap()
		_, found := parents2[asset.LocationID]
		if found {
			node := parents2[asset.LocationID].addChildFromItem(it)
			nodeLookup[it.ID()] = node
			delete(items2, it.ID())
		}
	}

	// process children of each parent
	for _, n := range parents {
		if len(n.children) > 0 {
			addChildNodes(n.children, items2, nodeLookup)
		}
	}
}

// TODO: Add node categories for all cargo variant

var locationFlag2Category = map[app.LocationFlag]NodeCategory{
	app.FlagAssetSafety:                         NodeAssetSafety,
	app.FlagCapsuleerDeliveries:                 NodeDeliveries,
	app.FlagCargo:                               NodeCargoBay,
	app.FlagCorpDeliveries:                      NodeDeliveries,
	app.FlagCorpSAG1:                            NodeOffice1,
	app.FlagCorpSAG2:                            NodeOffice2,
	app.FlagCorpSAG3:                            NodeOffice3,
	app.FlagCorpSAG4:                            NodeOffice4,
	app.FlagCorpSAG5:                            NodeOffice5,
	app.FlagCorpSAG6:                            NodeOffice6,
	app.FlagCorpSAG7:                            NodeOffice7,
	app.FlagDroneBay:                            NodeDroneBay,
	app.FlagFighterBay:                          NodeFighterBay,
	app.FlagFighterTube0:                        NodeFighterBay,
	app.FlagFighterTube1:                        NodeFighterBay,
	app.FlagFighterTube2:                        NodeFighterBay,
	app.FlagFighterTube3:                        NodeFighterBay,
	app.FlagFighterTube4:                        NodeFighterBay,
	app.FlagFleetHangar:                         NodeCargoBay,
	app.FlagFrigateEscapeBay:                    NodeFrigateEscapeBay,
	app.FlagHiSlot0:                             NodeFitting,
	app.FlagHiSlot1:                             NodeFitting,
	app.FlagHiSlot2:                             NodeFitting,
	app.FlagHiSlot3:                             NodeFitting,
	app.FlagHiSlot4:                             NodeFitting,
	app.FlagHiSlot5:                             NodeFitting,
	app.FlagHiSlot6:                             NodeFitting,
	app.FlagHiSlot7:                             NodeFitting,
	app.FlagImpounded:                           NodeImpounded,
	app.FlagLoSlot0:                             NodeFitting,
	app.FlagLoSlot1:                             NodeFitting,
	app.FlagLoSlot2:                             NodeFitting,
	app.FlagLoSlot3:                             NodeFitting,
	app.FlagLoSlot4:                             NodeFitting,
	app.FlagLoSlot5:                             NodeFitting,
	app.FlagLoSlot6:                             NodeFitting,
	app.FlagLoSlot7:                             NodeFitting,
	app.FlagMedSlot0:                            NodeFitting,
	app.FlagMedSlot1:                            NodeFitting,
	app.FlagMedSlot2:                            NodeFitting,
	app.FlagMedSlot3:                            NodeFitting,
	app.FlagMedSlot4:                            NodeFitting,
	app.FlagMedSlot5:                            NodeFitting,
	app.FlagMedSlot6:                            NodeFitting,
	app.FlagMedSlot7:                            NodeFitting,
	app.FlagMobileDepotHold:                     NodeCargoBay,
	app.FlagMoonMaterialBay:                     NodeCargoBay,
	app.FlagQuafeBay:                            NodeCargoBay,
	app.FlagRigSlot0:                            NodeFitting,
	app.FlagRigSlot1:                            NodeFitting,
	app.FlagRigSlot2:                            NodeFitting,
	app.FlagRigSlot3:                            NodeFitting,
	app.FlagRigSlot4:                            NodeFitting,
	app.FlagRigSlot5:                            NodeFitting,
	app.FlagRigSlot6:                            NodeFitting,
	app.FlagRigSlot7:                            NodeFitting,
	app.FlagShipHangar:                          NodeShipHangar,
	app.FlagSpecializedAmmoHold:                 NodeCargoBay,
	app.FlagSpecializedAsteroidHold:             NodeCargoBay,
	app.FlagSpecializedCommandCenterHold:        NodeCargoBay,
	app.FlagSpecializedFuelBay:                  NodeFuelBay,
	app.FlagSpecializedGasHold:                  NodeCargoBay,
	app.FlagSpecializedIceHold:                  NodeCargoBay,
	app.FlagSpecializedIndustrialShipHold:       NodeCargoBay,
	app.FlagSpecializedLargeShipHold:            NodeCargoBay,
	app.FlagSpecializedMaterialBay:              NodeCargoBay,
	app.FlagSpecializedMediumShipHold:           NodeCargoBay,
	app.FlagSpecializedMineralHold:              NodeCargoBay,
	app.FlagSpecializedOreHold:                  NodeCargoBay,
	app.FlagSpecializedPlanetaryCommoditiesHold: NodeCargoBay,
	app.FlagSpecializedSalvageHold:              NodeCargoBay,
	app.FlagSpecializedShipHold:                 NodeCargoBay,
	app.FlagSpecializedSmallShipHold:            NodeCargoBay,
	app.FlagStructureDeedBay:                    NodeCargoBay,
	app.FlagSubSystemSlot0:                      NodeFitting,
	app.FlagSubSystemSlot1:                      NodeFitting,
	app.FlagSubSystemSlot2:                      NodeFitting,
	app.FlagSubSystemSlot3:                      NodeFitting,
	app.FlagSubSystemSlot4:                      NodeFitting,
	app.FlagSubSystemSlot5:                      NodeFitting,
	app.FlagSubSystemSlot6:                      NodeFitting,
	app.FlagSubSystemSlot7:                      NodeFitting,
}

func insertCustomNodes(trees map[int64]*Node) {
	for _, root := range trees {
		for _, n := range root.All() {
			asset, ok := n.Asset()
			if !ok {
				continue
			}
			if asset.LocationType == app.TypeSolarSystem {
				addToCustomNode(n, NodeInSpace)
				continue
			}
			if n.IsRootDirectChild() && asset.Type != nil && asset.Type.IsShip() {
				addToCustomNode(n, NodeShipHangar)
				continue
			}
			if n.IsRootDirectChild() && asset.LocationFlag == app.FlagHangar {
				addToCustomNode(n, NodeItemHangar)
				continue
			}
			if c, ok := locationFlag2Category[asset.LocationFlag]; ok {
				addToCustomNode(n, c)
				continue
			}
		}
	}
}

// TODO: Add lookup for finding existing custom nodes more efficiently

func addToCustomNode(n *Node, category NodeCategory) bool {
	var isCreated bool
	parent := n.parent
	var p2 *Node
	idx := slices.IndexFunc(parent.children, func(x *Node) bool {
		return x.category == category
	})
	if idx != -1 {
		p2 = parent.children[idx]
	} else {
		p2 = newCustomNode(category)
		p2.parent = parent
		parent.addChild(p2)
		isCreated = true
	}
	p2.children = append(p2.children, n)
	parent.children = slices.DeleteFunc(parent.children, func(x *Node) bool {
		return x == n
	})
	n.parent = p2
	return isCreated
}

func addMissingOffices(trees map[int64]*Node) {
	for _, root := range trees {
		for _, n := range root.All() {
			if n.category != NodeOfficeFolder {
				continue
			}
			current := set.Collect(xiter.MapSlice(n.children, func(x *Node) NodeCategory {
				return x.category
			}))
			missing := set.Difference(
				set.Of(
					NodeOffice1,
					NodeOffice2,
					NodeOffice3,
					NodeOffice4,
					NodeOffice5,
					NodeOffice6,
					NodeOffice7,
				),
				current,
			)
			for c := range missing.All() {
				n2 := newCustomNode(c)
				n2.parent = n
				n.addChild(n2)
			}
		}
	}
}

// RootLocationNode returns the root location for an asset.
func (ac Collection) RootLocationNode(itemID int64) (*Node, bool) {
	ln, found := ac.rootLookup[itemID]
	if !found {
		return nil, false
	}
	return ln, true
}

// Node returns the node for an ID and reports whether it was found.
func (ac Collection) Node(itemID int64) (*Node, bool) {
	an, found := ac.nodeLookup[itemID]
	if !found {
		return nil, false
	}
	return an, true
}

// MustNode returns the node for an ID or panics if not found.
func (ac Collection) MustNode(itemID int64) *Node {
	n, ok := ac.Node(itemID)
	if !ok {
		panic(fmt.Sprintf("node not found for ID %d", itemID))
	}
	return n
}

// Trees returns the location trees.
func (ac Collection) Trees() []*Node {
	return slices.Collect(maps.Values(ac.trees))
}

// LocationTree returns the tree for a location and reports if the tree was found.
func (ac Collection) LocationTree(locationID int64) (*Node, bool) {
	root, ok := ac.trees[locationID]
	if !ok {
		return nil, false
	}
	return root, true
}

func (ac Collection) MustLocationTree(locationID int64) *Node {
	root, ok := ac.trees[locationID]
	if !ok {
		panic(fmt.Sprintf("location tree not found for ID %d", locationID))
	}
	return root
}
