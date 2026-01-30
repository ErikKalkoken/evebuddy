package asset

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNode_Leafs(t *testing.T) {
	root := newCustomNode(NodeItemHangar)
	a := newCustomNode(NodeCargoBay)
	root.addChild(a)
	b := newCustomNode(NodeFuelBay)
	a.addChild(b)
	c := newCustomNode(NodeDroneBay)
	root.addChild(c)

	got := root.LeafPaths()
	want := [][]string{
		{"Item Hangar", "Cargo Bay", "Fuel Bay"},
		{"Item Hangar", "Drone Bay"},
	}
	assert.ElementsMatch(t, want, got)
}
