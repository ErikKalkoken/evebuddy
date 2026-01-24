package asset

import (
	"fmt"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

//go:generate go tool stringer -type=NodeCategory

type NodeCategory uint

const (
	NodeUndefined NodeCategory = iota
	NodeAsset
	NodeAssetSafety
	NodeCargoBay
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
	NodeShipHangar
	NodeDeliveries
	NodeOther
)

var nodeCategoryNames = map[NodeCategory]string{
	NodeAsset:            "Asset",
	NodeAssetSafety:      "Asset Safety",
	NodeItemHangar:       "Item Hangar",
	NodeLocation:         "Location",
	NodeShipHangar:       "Ship Hangar",
	NodeInSpace:          "In Space",
	NodeCargoBay:         "Cargo Bay",
	NodeDroneBay:         "Drone Bay",
	NodeFitting:          "Fitting",
	NodeFrigateEscapeBay: "Frigate Escape Bay",
	NodeFighterBay:       "Fighter Bay",
	NodeFuelBay:          "Fuel Bay",
	NodeOffice1:          "1st Division",
	NodeOffice2:          "2nd Division",
	NodeOffice3:          "3rd Division",
	NodeOffice4:          "4th Division",
	NodeOffice5:          "5th Division",
	NodeOffice6:          "6th Division",
	NodeOffice7:          "7th Division",
	NodeImpounded:        "Impounded",
	NodeDeliveries:       "Deliveries",
	NodeOther:            "Other",
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
	// Whether this node is a container
	IsContainer bool

	// Whether this node is a ship
	IsShip bool

	// Count of items in this container. Empty for non-containers
	ItemCount optional.Optional[int]

	category NodeCategory
	children []*Node
	item     Item
	location *app.EveLocation
	parent   *Node
	seen     bool
}

func newLocationNode(location *app.EveLocation) *Node {
	return &Node{
		category:    NodeLocation,
		location:    location,
		IsContainer: true,
	}
}

func newAssetNode(ai Item) *Node {
	asset := ai.Unwrap()
	n := &Node{
		category:    NodeAsset,
		item:        ai,
		IsContainer: asset.IsContainer(),
	}
	if asset.Type != nil {
		n.IsShip = asset.Type.IsShip()
	}
	return n
}

func newCustomNode(category NodeCategory) *Node {
	switch category {
	case NodeAsset, NodeLocation, NodeUndefined:
		panic("invalid category for custom node: ")
	}
	return &Node{
		category: category,
	}
}

// All returns all nodes of a sub tree (order is undefined)
func (n *Node) All() []*Node {
	s := make([]*Node, 0)
	s = append(s, n)
	for _, c := range n.children {
		s = slices.Concat(s, c.All())
	}
	return s
}

func (n *Node) Children() []*Node {
	return slices.Clone(n.children)
}

// ID returns the ID of the node. This is the item ID or the location ID.
// Returns 0 when node has no ID.
func (n *Node) ID() int64 {
	if n.category == NodeAsset && n.item != nil {
		return n.item.ID()
	}
	if n.category == NodeLocation && n.location != nil {
		return n.location.ID
	}
	return 0
}

func (n *Node) Asset() (app.Asset, bool) {
	if n.category != NodeAsset || n.item == nil {
		return app.Asset{}, false
	}
	return n.item.Unwrap(), true
}

func (n *Node) Category() NodeCategory {
	return n.category
}

func (n *Node) MustAsset() app.Asset {
	a, ok := n.Asset()
	if !ok {
		panic("No asset found")
	}
	return a
}

func (n *Node) IsRoot() bool {
	return n.parent == nil
}

func (n *Node) updateItemCounts() {
	for _, x := range n.All() {
		if len(x.children) == 0 {
			continue
		}
		x.ItemCount.Set(len(x.children))
	}
}

// ItemCountAny returns the consolidated count of any items in this sub tree.
func (n *Node) ItemCountAny() int {
	var q int
	if n.item != nil {
		q = n.item.Unwrap().Quantity
	}
	for _, c := range n.children {
		q += c.ItemCountAny()
	}
	return q
}

// ItemCountFiltered returns the consolidated count of items in this sub tree
// excluding items that are inside a ship container, e.g. fittings.
func (n *Node) ItemCountFiltered() int {
	var q int
	if n.item != nil {
		q2, ok := n.item.Unwrap().QuantityFiltered()
		if !ok {
			return 0
		}
		q = q2
	}
	for _, c := range n.children {
		q += c.ItemCountFiltered()
	}
	return q
}

// Location returns the location for a node and reports whether the node is a location.
func (n *Node) Location() (*app.EveLocation, bool) {
	if n.category != NodeLocation || n.location == nil {
		return nil, false
	}
	return n.location, true
}

// MustLocation returns the location for a node or panics if the node is not a location.
func (n *Node) MustLocation() *app.EveLocation {
	el, ok := n.Location()
	if !ok {
		panic("not a location")
	}
	return el
}

// MustCharacterAsset returns the current item as character asset.
// Will panic if the item has a different type.
func (n *Node) MustCharacterAsset() *app.CharacterAsset {
	x, ok := n.CharacterAsset()
	if !ok {
		panic(fmt.Sprintf("Not a character asset: %d", n.ID()))
	}
	return x
}

// CharacterAsset tries to return the current item as character asset
// and reports whether it was successful.
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
// and reports whether it was successful.
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

// MustCorporationAsset returns the current item as corporation asset.
// Will panic if the item has a different type.
func (n *Node) MustCorporationAsset() *app.CorporationAsset {
	x, ok := n.CorporationAsset()
	if !ok {
		panic(fmt.Sprintf("Not a corporation asset: %d", n.ID()))
	}
	return x
}

func (n *Node) DisplayName() string {
	switch n.category {
	case NodeLocation:
		return n.location.DisplayName()
	case NodeAsset:
		return n.MustAsset().DisplayName2()
	}
	return n.category.DisplayName()
}

func (n *Node) Parent() *Node {
	return n.parent
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
func PrintTree(n *Node) {
	var printTree func(n *Node, indent string, last bool)
	printTree = func(n *Node, indent string, last bool) {
		var id string
		if x := n.ID(); x != 0 {
			id = fmt.Sprintf("#%d", x)
		}
		fmt.Printf("%s+-(%s)%s(%s)\n", indent, id, n.DisplayName(), n.ItemCount.StringFunc("", func(v int) string {
			return fmt.Sprint(v)
		}))
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
}
