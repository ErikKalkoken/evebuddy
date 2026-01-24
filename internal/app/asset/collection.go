// Package asset provides data structures to analyze and process asset data.
package asset

import (
	"fmt"
	"maps"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

type Item interface {
	ID() int64
	Unwrap() app.Asset
}

// TODO: Consider combining item and location nodes

// Collection is a collection of asset trees.
type Collection struct {
	isCorporation bool            // True when this collection contains corporation assets, false for character assets.
	items         map[int64]*Node // Trees of asset items
	locations     map[int64]*Node // Trees of asset locations
	rootLocations map[int64]*Node // lookup for root location of items
}

func NewFromCharacterAssets(assets []*app.CharacterAsset, loc []*app.EveLocation) Collection {
	items := make([]Item, 0)
	for _, ca := range assets {
		items = append(items, ca)
	}
	return new(items, loc, false)
}

func NewFromCorporationAssets(assets []*app.CorporationAsset, loc []*app.EveLocation) Collection {
	items := make([]Item, 0)
	for _, ca := range assets {
		if ca.Type.ID == app.EveTypeAlliance {
			continue // workaround for filtering the "alliance" item
		}
		items = append(items, ca)
	}
	return new(items, loc, true)
}

// insert custom nodes (e.g. ship hangars)
var flag2CategoryRoot = map[app.LocationFlag]NodeCategory{
	app.FlagAssetSafety:         NodeAssetSafety,
	app.FlagCapsuleerDeliveries: NodeDeliveries,
	app.FlagCorpDeliveries:      NodeDeliveries,
}

// TODO: Add node categories for all cargo variant

var flag2CategoryOther = map[app.LocationFlag]NodeCategory{
	app.FlagCargo:                               NodeCargoBay,
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

// new returns a new Collection.
func new(items []Item, loc []*app.EveLocation, isCorporation bool) Collection {
	locationMap := make(map[int64]*app.EveLocation)
	for _, loc := range loc {
		locationMap[loc.ID] = loc
	}
	// initial map of all items
	// items will be removed from this map as they are added to the tree
	items2 := make(map[int64]Item)
	for _, it := range items {
		items2[it.ID()] = it
	}
	locationNodes := make(map[int64]*Node)
	for _, it := range items2 {
		asset := it.Unwrap()
		_, found := items2[asset.LocationID]
		if found {
			continue
		}
		loc, found := locationMap[asset.LocationID]
		if !found {
			continue
		}
		locationNodes[loc.ID] = newLocationNode(loc)
	}
	// add top itemNodes to locations
	itemNodes := make(map[int64]*Node)
	for _, it := range items2 {
		location, found := locationNodes[it.Unwrap().LocationID]
		if !found {
			continue
		}
		itemNodes[it.ID()] = location.addChildFromItem(it)
		delete(items2, it.ID())
	}
	// add others assets to locations
	for _, l := range locationNodes {
		addChildNodes(l.children, items2, itemNodes)
	}

	// create root locations lookup
	rootLocations := make(map[int64]*Node)
	for _, ln := range locationNodes {
		for _, n := range ln.children {
			for _, c := range n.All() {
				rootLocations[c.item.ID()] = ln
			}
		}
	}

	for _, tree := range itemNodes {
		for _, n := range tree.All() {
			asset, ok := n.Asset()
			if !ok {
				continue
			}
			if n.seen {
				continue
			}
			if n.parent.IsRoot() {
				if c, ok := flag2CategoryRoot[asset.LocationFlag]; ok {
					addToCustomNode(n, c)
					continue
				}
				if asset.IsInSpace() {
					addToCustomNode(n, NodeInSpace)
					continue
				}
				if asset.Type != nil && asset.Type.IsShip() {
					addToCustomNode(n, NodeShipHangar)
					continue
				}
				if asset.LocationFlag == app.FlagOfficeFolder {
					continue
				}
				addToCustomNode(n, NodeItemHangar)
				continue
			}
			if c, ok := flag2CategoryOther[asset.LocationFlag]; ok {
				addToCustomNode(n, c)
				continue
			}
		}
	}

	// update item counts
	for _, tree := range locationNodes {
		tree.updateItemCounts()
	}

	ac := Collection{
		isCorporation: isCorporation,
		items:         itemNodes,
		locations:     locationNodes,
		rootLocations: rootLocations,
	}
	return ac
}

// TODO: Add lookup for finding existing custom nodes quicker

func addToCustomNode(n *Node, category NodeCategory) {
	parent := n.parent
	var p2 *Node
	idx := slices.IndexFunc(parent.children, func(x *Node) bool {
		return x.category == category // TODO: can be optimized
	})
	if idx != -1 {
		p2 = parent.children[idx]
	} else {
		p2 = newCustomNode(category)
		p2.parent = parent
		p2.IsContainer = true
		parent.addChild(p2)
	}
	p2.children = append(p2.children, n)
	parent.children = slices.DeleteFunc(parent.children, func(x *Node) bool {
		return x == n
	})
	n.parent = p2
	n.seen = true
}

// addChildNodes adds assets as nodes to parents. Recursive.
func addChildNodes(parents []*Node, items2 map[int64]Item, itemNodes map[int64]*Node) {
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
			n := parents2[asset.LocationID].addChildFromItem(it)
			itemNodes[it.ID()] = n
			delete(items2, it.ID())
		}
	}

	// process children of each parent
	for _, n := range parents {
		if len(n.children) > 0 {
			addChildNodes(n.children, items2, itemNodes)
		}
	}
}

// RootLocationNode returns the root location for an asset.
func (ac Collection) RootLocationNode(itemID int64) (*Node, bool) {
	ln, found := ac.rootLocations[itemID]
	if !found {
		return nil, false
	}
	return ln, true
}

// Node returns the node for an ID and reports whether it was found.
func (ac Collection) Node(itemID int64) (*Node, bool) {
	an, found := ac.items[itemID]
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

// ItemCountFiltered returns the consolidated count of all items excluding items inside ships containers.
func (ac Collection) ItemCountFiltered() int {
	var n int
	for _, l := range ac.locations {
		n += l.ItemCountFiltered()
	}
	return n
}

// LocationNodes returns a slice of all location nodes.
func (ac Collection) LocationNodes() []*Node {
	return slices.Collect(maps.Values(ac.locations))
}
