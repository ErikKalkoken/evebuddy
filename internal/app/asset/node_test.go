package asset

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestNode_Leafs(t *testing.T) {
	top := newCustomNode(NodeItemHangar)
	a := newCustomNode(NodeCargoBay)
	top.addChild(a)
	b := newCustomNode(NodeFuelBay)
	a.addChild(b)
	c := newCustomNode(NodeDroneBay)
	top.addChild(c)

	got := top.LeafPaths()

	want := [][]string{
		{"Item Hangar", "Cargo Bay", "Fuel Bay"},
		{"Item Hangar", "Drone Bay"},
	}
	assert.ElementsMatch(t, want, got)
}

func TestNode_Path(t *testing.T) {
	top := newCustomNode(NodeItemHangar)
	a := newCustomNode(NodeCargoBay)
	top.addChild(a)
	b := newCustomNode(NodeFuelBay)
	a.addChild(b)
	c := newCustomNode(NodeDroneBay)
	top.addChild(c)

	got := b.Path()

	want := []*Node{top, a, b}
	xassert.Equal(t, want, got)
}
