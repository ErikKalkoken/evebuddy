package asset

import (
	"fmt"
	"iter"
	"slices"
	"strconv"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

//go:generate go tool stringer -type=NodeCategory

// NodeCategory represents the category of a node.
// [NodeAsset] represents assets and [NodeLocation] represent eve locations.
// All other categories represent custom nodes,
// except for [NodeOfficeFolder] which is both an asset and a custom node.
type NodeCategory uint

const (
	NodeUndefined NodeCategory = iota
	NodeAsset
	NodeAssetSafetyCharacter
	NodeAssetSafetyCorporation
	NodeCargoBay
	NodeDeliveries
	NodeDroneBay
	NodeFighterBay
	NodeFitting
	NodeFrigateEscapeBay
	NodeFuelBay
	NodeImpounded
	NodeInSpace
	NodeItemHangar
	NodeLocation
	NodeOfficeFolder
	NodeOffice1
	NodeOffice2
	NodeOffice3
	NodeOffice4
	NodeOffice5
	NodeOffice6
	NodeOffice7
	NodeShipHangar
)

var nodeCategoryNames = map[NodeCategory]string{
	NodeAsset:                  "Asset",
	NodeAssetSafetyCharacter:   "Asset Safety",
	NodeAssetSafetyCorporation: "Asset Safety",
	NodeItemHangar:             "Item Hangar",
	NodeLocation:               "Location",
	NodeShipHangar:             "Ship Hangar",
	NodeInSpace:                "In Space",
	NodeCargoBay:               "Cargo Bay",
	NodeDroneBay:               "Drone Bay",
	NodeFitting:                "Fitting",
	NodeFrigateEscapeBay:       "Frigate Escape Bay",
	NodeFighterBay:             "Fighter Bay",
	NodeFuelBay:                "Fuel Bay",
	NodeOfficeFolder:           "Office",
	NodeOffice1:                "1st Division",
	NodeOffice2:                "2nd Division",
	NodeOffice3:                "3rd Division",
	NodeOffice4:                "4th Division",
	NodeOffice5:                "5th Division",
	NodeOffice6:                "6th Division",
	NodeOffice7:                "7th Division",
	NodeImpounded:              "Impounded",
	NodeDeliveries:             "Deliveries",
}

func (c NodeCategory) DisplayName() string {
	if n, ok := nodeCategoryNames[c]; ok {
		return n
	}
	return c.String()
}

// Node is a node in an asset tree.
// A node can represent an Eve asset, an Eve location or a custom node.
type Node struct {
	category    NodeCategory
	children    []*Node
	isContainer bool
	isExcluded  bool
	isShip      bool
	item        Item
	itemCount   optional.Optional[int]
	location    *app.EveLocation
	parent      *Node
}

func newLocationNode(location *app.EveLocation) *Node {
	return &Node{
		category:    NodeLocation,
		location:    location,
		isContainer: true,
	}
}

func newAssetNode(it Item) *Node {
	as := it.Unwrap()
	var c NodeCategory
	if as.Type != nil && as.Type.ID == app.EveTypeOffice {
		c = NodeOfficeFolder
	} else {
		c = NodeAsset
	}
	n := &Node{
		category:    c,
		item:        it,
		isContainer: as.IsContainer(),
	}
	if as.Type != nil {
		n.isShip = as.Type.IsShip()
	}
	return n
}

func newCustomNode(category NodeCategory) *Node {
	switch category {
	case NodeAsset, NodeLocation, NodeUndefined:
		panic("invalid category for custom node: ")
	}
	return &Node{
		category:    category,
		isContainer: true,
	}
}

// All returns an iterator over all nodes of a sub tree.
// The iterator runs a depth-first search and returns the children of each node
// in the same order as they where added.
func (n *Node) All() iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		if n == nil {
			return
		}

		var traverse func(*Node) bool
		traverse = func(curr *Node) bool {
			if curr == nil {
				return true
			}
			if !yield(curr) {
				return false
			}
			for _, c := range curr.children {
				if !traverse(c) {
					return false
				}
			}
			return true
		}

		traverse(n)
	}
}

// all returns a new slices with all nodes in a breath first search order.
func (n *Node) all() []*Node {
	s := make([]*Node, 0)
	s = append(s, n)
	for _, c := range n.children {
		if c.isExcluded {
			continue
		}
		s = slices.Concat(s, c.all())
	}
	return s
}

func (n *Node) ItemCount() optional.Optional[int] {
	return n.itemCount
}

// Children returns a new slice containing the unfiltered children of a node.
func (n *Node) Children() []*Node {
	return xslices.Filter(n.children, func(x *Node) bool {
		return !x.isExcluded
	})
}

func (n *Node) ChildrenCount() int {
	var count int
	for _, n := range n.children {
		if !n.isExcluded {
			count++
		}
	}
	return count
}

// IsContainer reports whether this node is a container
func (n *Node) IsContainer() bool {
	return n.isContainer
}

// IsShip reports whether this node is a ship
func (n *Node) IsShip() bool {
	return n.isShip
}

// ID returns the ID of the node. This is the item ID or the location ID.
// Returns 0 when node has no ID.
func (n *Node) ID() int64 {
	if n.item != nil {
		return n.item.ID()
	}
	if n.location != nil {
		return n.location.ID
	}
	return 0
}

func (n *Node) Asset() (app.Asset, bool) {
	if n.item == nil {
		return app.Asset{}, false
	}
	return n.item.Unwrap(), true
}

func (n *Node) Category() NodeCategory {
	return n.category
}

// IsRoot reports whether a node is the root in a tree.
func (n *Node) IsRoot() bool {
	return n.parent == nil
}

// IsRootDirectChild reports whether a node is direct child of the root.
func (n *Node) IsRootDirectChild() bool {
	return n.parent != nil && n.parent.parent == nil
}

// Location returns the location for a node and reports whether the node is a location.
func (n *Node) Location() (*app.EveLocation, bool) {
	if n.location == nil {
		return nil, false
	}
	return n.location, true
}

// CharacterAsset tries to return the current item as character asset
// and reports whether it was found.
func (n *Node) CharacterAsset() (*app.CharacterAsset, bool) {
	if n.item == nil {
		return nil, false
	}
	x, ok := n.item.(*app.CharacterAsset)
	if !ok {
		return nil, false
	}
	return x, true
}

// CorporationAsset tries to return the current item as corporation asset
// and reports whether it was found.
func (n *Node) CorporationAsset() (*app.CorporationAsset, bool) {
	if n.item == nil {
		return nil, false
	}
	x, ok := n.item.(*app.CorporationAsset)
	if !ok {
		return nil, false
	}
	return x, true
}

func (n *Node) DisplayName() string {
	switch n.category {
	case NodeLocation:
		el, ok := n.Location()
		if !ok {
			return "?"
		}
		return el.DisplayName()
	case NodeAsset:
		n, ok := n.Asset()
		if !ok {
			return "?"
		}
		return n.DisplayName2()
	}
	return n.category.DisplayName()
}

func (n *Node) Path() []*Node {
	nodes := make([]*Node, 0)
	current := n
	for current.parent != nil {
		nodes = append(nodes, current.parent)
		current = current.parent
	}
	slices.Reverse(nodes)
	return nodes
}

func (n *Node) addChild(c *Node) {
	c.parent = n
	n.children = append(n.children, c)
}

func (n *Node) addChildFromItem(it Item) *Node {
	c := newAssetNode(it)
	n.addChild(c)
	return c
}

// PrintTree prints the subtree of n.
func (n *Node) PrintTree() {
	var printTree func(n *Node, indent string, last bool)
	printTree = func(n *Node, indent string, last bool) {
		var id string
		if v := n.ID(); v != 0 {
			id = fmt.Sprintf(" (#%d)", v)
		}
		var count string
		if x := n.ChildrenCount(); x > 0 {
			count = fmt.Sprint(x)
		} else {
			count = "-"
		}
		fmt.Printf("%s+-%s%s [%s] %s: %s\n",
			indent,
			n.DisplayName(),
			id,
			count,
			n.Category().String(),
			n.itemCount.StringFunc("-", func(v int) string {
				return strconv.Itoa(v)
			}),
		)
		if last {
			indent += "   "
		} else {
			indent += "|  "
		}
		for _, c := range n.children {
			printTree(c, indent, len(c.children) == 0)
		}
	}

	printTree(n, "", false)
	fmt.Println()
}

func (n *Node) String() string {
	var id string
	if v := n.ID(); v != 0 {
		id = fmt.Sprintf(" (#%d)", v)
	}
	return fmt.Sprintf("%s%s", n.DisplayName(), id)
}
