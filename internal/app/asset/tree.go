// Package asset provides data structures to analyze and process asset data.
package asset

import (
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

type Item interface {
	ID() int64
	Unwrap() app.Asset
}

// Tree represents an asset tree in Eve Online.
// It can contain both character and corporation assets from one or multiple owners.
//
// Assets are organized by EVE locations (e.g. station) which form the main branches of the tree.
// The root node is implicit.
type Tree struct {
	nodeLookup     map[int64]*Node // Lookup of nodes for items
	locationLookup map[int64]*Node // Lookup of location nodes for items
	locations      map[int64]*Node // Map of location IDs to location nodes.
}

func NewFromCharacterAssets(assets []*app.CharacterAsset, locations []*app.EveLocation) Tree {
	items := make([]Item, 0)
	for _, ca := range assets {
		items = append(items, ca)
	}
	return new(items, locations, false)
}

func NewFromCorporationAssets(assets []*app.CorporationAsset, locations []*app.EveLocation) Tree {
	items := make([]Item, 0)
	for _, ca := range assets {
		items = append(items, ca)
	}
	return new(items, locations, true)
}

// new returns a new asset tree.
func new(items []Item, locations []*app.EveLocation, isCorporation bool) Tree {
	// map of location ID to eve locations
	locationLookup := make(map[int64]*app.EveLocation)
	for _, loc := range locations {
		locationLookup[loc.ID] = loc
	}

	// initial map of all items
	// items will be removed from this map as they are added to the locations
	items2 := make(map[int64]Item)
	for _, it := range items {
		items2[it.ID()] = it
	}

	// create location nodes, which form the foundation of the main branches
	locations2 := make(map[int64]*Node)
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
		locations2[loc.ID] = newLocationNode(loc)
	}

	// add items to locations and make node lookup
	nodeLookup := make(map[int64]*Node)
	for _, it := range items2 {
		loc, found := locations2[it.Unwrap().LocationID]
		if !found {
			continue
		}
		node := loc.addChildFromItem(it)
		nodeLookup[it.ID()] = node
		delete(items2, it.ID())
	}
	for _, loc := range locations2 {
		addChildNodes(loc.children, items2, nodeLookup)
	}

	insertCustomNodes(locations2, isCorporation)
	if isCorporation {
		addMissingOffices(locations2)
	} else {
		addMissingHangars(locations2)
	}

	ac := Tree{
		nodeLookup:     nodeLookup,
		locationLookup: makeLocationLookup(locations2),
		locations:      locations2,
	}
	return ac
}

func makeLocationLookup(locations map[int64]*Node) map[int64]*Node {
	lookup := make(map[int64]*Node)
	for _, loc := range locations {
		for _, n := range loc.children {
			for c := range n.All() {
				if c.item != nil {
					lookup[c.item.ID()] = loc
				}
			}
		}
	}
	return lookup
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

var locationFlag2CategoryShared = map[app.LocationFlag]NodeCategory{
	app.FlagCargo:                               NodeCargoBay,
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
	app.FlagCapsuleerDeliveries:                 NodeDeliveries,
}

var locationFlag2CategoryCorp = map[app.LocationFlag]NodeCategory{
	app.FlagCorpDeliveries: NodeDeliveries,
	app.FlagCorpSAG1:       NodeOffice1,
	app.FlagCorpSAG2:       NodeOffice2,
	app.FlagCorpSAG3:       NodeOffice3,
	app.FlagCorpSAG4:       NodeOffice4,
	app.FlagCorpSAG5:       NodeOffice5,
	app.FlagCorpSAG6:       NodeOffice6,
	app.FlagCorpSAG7:       NodeOffice7,
	app.FlagImpounded:      NodeImpounded,
}

func insertCustomNodes(locations map[int64]*Node, isCorporation bool) {
	for _, location := range locations {
		for _, n := range location.all() {
			asset, ok := n.Asset()
			if !ok {
				continue
			}
			if asset.LocationType == app.TypeSolarSystem {
				addToCustomNode(n, NodeInSpace)
				continue
			}
			if isCorporation {
				if asset.LocationFlag == app.FlagAssetSafety {
					addToCustomNode(n, NodeAssetSafetyCorporation)
					continue
				}
				if c, ok := locationFlag2CategoryCorp[asset.LocationFlag]; ok {
					addToCustomNode(n, c)
					continue
				}
			} else {
				if n.IsSecondLevel() {
					if asset.Type != nil && asset.Type.IsShip() {
						addToCustomNode(n, NodeShipHangar)
						continue
					}
					if asset.LocationFlag == app.FlagHangar {
						addToCustomNode(n, NodeItemHangar)
						continue
					}
				}
				if asset.LocationFlag == app.FlagAssetSafety {
					addToCustomNode(n, NodeAssetSafetyCharacter)
					continue
				}
			}
			if c, ok := locationFlag2CategoryShared[asset.LocationFlag]; ok {
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

func addMissingOffices(locations map[int64]*Node) {
	for _, location := range locations {
		for n := range location.All() {
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

func addMissingHangars(locations map[int64]*Node) {
	for _, location := range locations {
		current := set.Collect(xiter.MapSlice(location.children, func(x *Node) NodeCategory {
			return x.category
		}))
		var missing set.Set[NodeCategory]
		if current.Contains(NodeItemHangar) && !current.Contains(NodeShipHangar) {
			missing.Add(NodeShipHangar)
		} else if current.Contains(NodeShipHangar) && !current.Contains(NodeItemHangar) {
			missing.Add(NodeShipHangar)
		}
		for c := range missing.All() {
			n2 := newCustomNode(c)
			n2.parent = location
			location.addChild(n2)
		}
	}
}

// LocationForItem returns the location node for an asset.
func (ac Tree) LocationForItem(itemID int64) (*Node, bool) {
	ln, found := ac.locationLookup[itemID]
	if !found {
		return nil, false
	}
	return ln, true
}

// Location returns the location node for an EVE location ID
// and reports whether it was found.
func (ac Tree) Location(locationID int64) (*Node, bool) {
	loc, ok := ac.locations[locationID]
	if !ok {
		return nil, false
	}
	return loc, true
}

// Node returns the node for an item ID and reports whether it was found.
func (ac Tree) Node(itemID int64) (*Node, bool) {
	an, found := ac.nodeLookup[itemID]
	if !found {
		return nil, false
	}
	return an, true
}

// Locations returns a new slice with all location nodes.
// Locations without any further nodes are excluded.
func (ac Tree) Locations() []*Node {
	locations := make([]*Node, 0)
	for _, loc := range ac.locations {
		if loc.ChildrenCount() > 0 {
			locations = append(locations, loc)
		}
	}
	return locations
}
