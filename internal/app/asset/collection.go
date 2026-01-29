// Package asset provides data structures to analyze and process asset data.
package asset

import (
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

type Filter uint

const (
	FilterNone = iota
	FilterDeliveries
	FilterImpounded
	FilterInSpace
	FilterOffice
	FilterPersonalAssets
	FilterSafety
	FilterCorpOther
)

type Item interface {
	ID() int64
	Unwrap() app.Asset
}

// Collection represents a collection of asset trees.
// There is one tree for each location.
type Collection struct {
	filter        Filter
	isCorporation bool            // True for corporation assets, false for character assets.
	nodeLookup    map[int64]*Node // Lookup of nodes for items
	rootLookup    map[int64]*Node // Lookup of root nodes for items
	trees         map[int64]*Node // Map of location IDs to root nodes.
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

	insertCustomNodes(trees, isCorporation)
	if isCorporation {
		addMissingOffices(trees)
	} else {
		addMissingHangars(trees)
	}

	ac := Collection{
		isCorporation: isCorporation,
		nodeLookup:    nodeLookup,
		rootLookup:    makeRootLookup(trees),
		trees:         trees,
	}
	return ac
}

func makeRootLookup(trees map[int64]*Node) map[int64]*Node {
	rootLookup := make(map[int64]*Node)
	for _, root := range trees {
		for _, n := range root.children {
			for c := range n.All() {
				if c.item != nil {
					rootLookup[c.item.ID()] = root
				}
			}
		}
	}
	return rootLookup
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

func insertCustomNodes(trees map[int64]*Node, isCorporation bool) {
	for _, root := range trees {
		for _, n := range root.all() {
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
				if n.IsRootDirectChild() {
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

func addMissingOffices(trees map[int64]*Node) {
	for _, root := range trees {
		for n := range root.All() {
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

func addMissingHangars(trees map[int64]*Node) {
	for _, root := range trees {
		current := set.Collect(xiter.MapSlice(root.children, func(x *Node) NodeCategory {
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
			n2.parent = root
			root.addChild(n2)
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

// Trees returns a new slice with all root nodes.
// Trees which do not have any filtered nodes will be excluded.
func (ac Collection) Trees() []*Node {
	trees := make([]*Node, 0)
	for _, root := range ac.trees {
		if root.ChildrenCount() > 0 {
			trees = append(trees, root)
		}
	}
	return trees
}

// LocationTree returns the tree for a location and reports if the tree was found.
func (ac Collection) LocationTree(locationID int64) (*Node, bool) {
	root, ok := ac.trees[locationID]
	if !ok {
		return nil, false
	}
	return root, true
}

// Filter returns the current filter.
func (ac Collection) Filter() Filter {
	return ac.filter
}

// ApplyFilter applies the specified filter to the collection.
func (ac *Collection) ApplyFilter(filter Filter) {
	for _, root := range ac.trees {
		for _, n := range root.children {
			var isExcluded bool
			switch filter {
			case FilterCorpOther:
				switch n.category {
				case
					NodeAssetSafetyCorporation,
					NodeDeliveries,
					NodeImpounded,
					NodeInSpace,
					NodeOfficeFolder:
					isExcluded = true
				default:
					isExcluded = false
				}
			case FilterDeliveries:
				isExcluded = n.category != NodeDeliveries
			case FilterImpounded:
				isExcluded = n.category != NodeImpounded
			case FilterInSpace:
				isExcluded = n.category != NodeInSpace
			case FilterOffice:
				isExcluded = n.category != NodeOfficeFolder

			case FilterPersonalAssets:
				switch n.category {
				case NodeAssetSafetyCharacter, NodeDeliveries, NodeInSpace:
					isExcluded = true
				default:
					isExcluded = false
				}
			case FilterSafety:
				if ac.isCorporation {
					isExcluded = n.category != NodeAssetSafetyCorporation
				} else {
					isExcluded = n.category != NodeAssetSafetyCharacter
				}
			}
			n.isExcluded = isExcluded
		}
	}
	ac.filter = filter
}

func (ac Collection) UpdateItemCounts() {
	for _, location := range ac.trees {
		for n := range location.All() {
			n.itemCount.Clear()
		}
		for _, top := range location.children {
			switch top.category {
			case NodeOfficeFolder, NodeAssetSafetyCharacter:
				for _, n1 := range top.children {
					n1.itemCount = optional.FromIntegerWithZero(len(n1.children))
					top.itemCount = optional.Sum(top.itemCount, n1.itemCount)
				}
			case NodeAssetSafetyCorporation, NodeImpounded:
				for _, n1 := range top.children {
					for _, n2 := range n1.children {
						n2.itemCount = optional.FromIntegerWithZero(len(n2.children))
						n1.itemCount = optional.Sum(n1.itemCount, n2.itemCount)
					}
					top.itemCount = optional.Sum(top.itemCount, n1.itemCount)
				}
			default:
				top.itemCount = optional.FromIntegerWithZero(len(top.children))
			}
			location.itemCount = optional.Sum(location.itemCount, top.itemCount)
		}
	}
}
