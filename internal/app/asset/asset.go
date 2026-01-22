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
		items = append(items, ca)
	}
	return new(items, loc, true)
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
		locationNodes[loc.ID] = &Node{location: loc}
	}
	// add top itemNodes to locations
	itemNodes := make(map[int64]*Node)
	for _, it := range items2 {
		location, found := locationNodes[it.Unwrap().LocationID]
		if !found {
			continue
		}
		itemNodes[it.ID()] = location.add(it)
		delete(items2, it.ID())
	}
	// add others assets to locations
	for _, l := range locationNodes {
		addChildNodes(l.children, items2, itemNodes)
	}
	rootLocations := make(map[int64]*Node)
	for _, ln := range locationNodes {
		for _, n := range ln.children {
			for _, c := range n.All() {
				rootLocations[c.item.ID()] = ln
			}
		}
	}
	ac := Collection{
		isCorporation: isCorporation,
		items:         itemNodes,
		locations:     locationNodes,
		rootLocations: rootLocations,
	}
	return ac
}

// addChildNodes adds assets as nodes to parents. Recursive.
func addChildNodes(parents []*Node, items2 map[int64]Item, itemNodes map[int64]*Node) {
	parents2 := make(map[int64]*Node)
	for _, n := range parents {
		parents2[n.ID()] = n
	}

	for _, it := range items2 {
		asset := it.Unwrap()
		_, found := parents2[asset.LocationID]
		if found {
			n := parents2[asset.LocationID].add(it)
			itemNodes[it.ID()] = n
			delete(items2, it.ID())
		}
	}

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

// Node is a node in an asset tree.
// A node can represent an asset item or a location in Eve.
type Node struct {
	children []*Node
	item     Item
	location *app.EveLocation
	name     string
	parent   *Node
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
func (n *Node) ID() int64 {
	if n.location != nil {
		return n.location.ID
	}
	return n.item.ID()
}

func (n *Node) Item() Item {
	return n.item
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

func (n *Node) Location() *app.EveLocation {
	return n.location
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
		panic(fmt.Sprintf("Not a character asset: %d", n.ID()))
	}
	return x
}

func (n *Node) DisplayName() string {
	if n.name != "" {
		return n.name
	}
	if n.location != nil {
		return n.location.DisplayName()
	}
	if n.item != nil {
		return n.item.Unwrap().DisplayName2()
	}
	return "?"
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

func (n *Node) add(it Item) *Node {
	n2 := &Node{item: it, parent: n}
	n.children = append(n.children, n2)
	return n2
}
