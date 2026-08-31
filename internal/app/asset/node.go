package asset

import (
	"fmt"
	"iter"
	"slices"
	"sync/atomic"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

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
	NodeOffice1
	NodeOffice2
	NodeOffice3
	NodeOffice4
	NodeOffice5
	NodeOffice6
	NodeOffice7
	NodeOfficeFolder
	NodeShipHangar
)

var nodeCategoryNames = map[NodeCategory]string{
	NodeAsset:                  "Asset",
	NodeAssetSafetyCharacter:   "Asset Safety",
	NodeAssetSafetyCorporation: "Asset Safety",
	NodeCargoBay:               "Cargo Bay",
	NodeDeliveries:             "Deliveries",
	NodeDroneBay:               "Drone Bay",
	NodeFighterBay:             "Fighter Bay",
	NodeFitting:                "Fitting",
	NodeFrigateEscapeBay:       "Frigate Escape Bay",
	NodeFuelBay:                "Fuel Bay",
	NodeImpounded:              "Impounded",
	NodeInSpace:                "In Space",
	NodeItemHangar:             "Item Hangar",
	NodeLocation:               "Location",
	NodeOffice1:                "1st Division",
	NodeOffice2:                "2nd Division",
	NodeOffice3:                "3rd Division",
	NodeOffice4:                "4th Division",
	NodeOffice5:                "5th Division",
	NodeOffice6:                "6th Division",
	NodeOffice7:                "7th Division",
	NodeOfficeFolder:           "Office",
	NodeShipHangar:             "Ship Hangar",
}

func (c NodeCategory) String() string {
	if n, ok := nodeCategoryNames[c]; ok {
		return n
	}
	return "?"
}

// sequence provides a sequence of unique numbers for custom categories.
var sequence atomic.Int64

// Node is a node in an asset tree.
// A node can represent an Eve asset, an Eve location or a custom node.
type Node struct {
	category    NodeCategory
	children    []*Node
	id          int64
	isContainer bool
	isShip      bool
	item        Item
	location    *app.EveLocation
	parent      *Node
}

func newLocationNode(location *app.EveLocation) *Node {
	if location == nil {
		panic("must provide a location")
	}
	n := &Node{
		category:    NodeLocation,
		id:          location.ID,
		isContainer: true,
		location:    location,
	}
	return n
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
		id:          it.ID(),
		isContainer: as.IsContainer(),
		item:        it,
	}
	if as.Type != nil {
		n.isShip = as.Type.IsShip()
	}
	return n
}

func newCustomNode(category NodeCategory) *Node {
	switch category {
	case NodeAsset, NodeLocation, NodeUndefined:
		panic("invalid category for custom node: " + category.String())
	}
	n := &Node{
		id:          sequence.Add(1),
		category:    category,
		isContainer: true,
	}
	return n
}

// All returns an iterator over all nodes of a sub tree.
// The nodes are returned in depth-first order.
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

// AncestorCount returns the number of ancestors of a node.
func (n *Node) AncestorCount() int {
	if n == nil {
		return 0
	}
	return len(n.Path()) - 1
}

// Asset tries to return the asset of a node and reports whether it was successful.
func (n *Node) Asset() (app.Asset, bool) {
	if n == nil || n.item == nil {
		return app.Asset{}, false
	}
	return n.item.Unwrap(), true
}

// Category returns the category of a node.
func (n *Node) Category() NodeCategory {
	if n == nil {
		return NodeUndefined
	}
	return n.category
}

// CharacterAsset tries to return the current item as character asset
// and reports whether it was found.
func (n *Node) CharacterAsset() (*app.CharacterAsset, bool) {
	if n == nil || n.item == nil {
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
	if n == nil || n.item == nil {
		return nil, false
	}
	x, ok := n.item.(*app.CorporationAsset)
	if !ok {
		return nil, false
	}
	return x, true
}

// Children returns a new slice containing the children of a node.
func (n *Node) Children() []*Node {
	if n == nil {
		return nil
	}
	return slices.Clone(n.children)
}

// ChildrenCount returns the number of children.
func (n *Node) ChildrenCount() int {
	if n == nil {
		return 0
	}
	return len(n.children)
}

// ID returns the  ID of the node.
func (n *Node) ID() int64 {
	if n == nil {
		return 0
	}
	return n.id
}

// UID returns the unique ID of the node. This is used by the tree widget.
func (n *Node) UID() string {
	if n == nil {
		return ""
	}
	return fmt.Sprintf("%d-%d", n.category, n.id)
}

// IsContainer reports whether this node is a container
func (n *Node) IsContainer() bool {
	if n == nil {
		return false
	}
	return n.isContainer
}

// IsShip reports whether this node is a ship
func (n *Node) IsShip() bool {
	if n == nil {
		return false
	}
	return n.isShip
}

// Location returns the location for a node and reports whether the node is a location.
func (n *Node) Location() (*app.EveLocation, bool) {
	if n == nil || n.location == nil {
		return nil, false
	}
	return n.location, true
}

// Path returns the path from the root to this node.
// The path includes the root and the node itself.
func (n *Node) Path() []*Node {
	if n == nil {
		return nil
	}
	nodes := []*Node{n}
	current := n
	for current.parent != nil {
		nodes = append(nodes, current.parent)
		current = current.parent
	}
	slices.Reverse(nodes)
	return nodes
}

// AllPaths returns a slice of paths to all leafs for a subtree.
// The nil node represents the root.
func (n *Node) AllPaths() [][]string {
	if n == nil {
		return nil
	}
	var all [][]string
	for n := range n.All() {
		if c := n.ChildrenCount(); c == 0 {
			p := xslices.Map(n.Path(), func(x *Node) string {
				return x.String()
			})
			all = append(all, p)
		}
	}
	return all
}

// Parent return the parent of a node.
// Returns nil when node has no parent.
func (n *Node) Parent() *Node {
	if n == nil {
		return nil
	}
	return n.parent
}

// String returns a string representation of the node which is usually it's name.
func (n *Node) String() string {
	if n == nil {
		return ""
	}
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
		return n.DisplayName()
	}
	return n.category.String()
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
