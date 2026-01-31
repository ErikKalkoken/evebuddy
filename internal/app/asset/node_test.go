package asset

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestNodeCategory_DisplayName(t *testing.T) {
	xassert.Equal(t, "Cargo Bay", NodeCargoBay.String())
}

func TestNode_AllPaths(t *testing.T) {
	top := newCustomNode(NodeItemHangar)
	a := newCustomNode(NodeCargoBay)
	top.addChild(a)
	b := newCustomNode(NodeFuelBay)
	a.addChild(b)
	c := newCustomNode(NodeDroneBay)
	top.addChild(c)

	got := top.AllPaths()

	want := [][]string{
		{"Item Hangar", "Cargo Bay", "Fuel Bay"},
		{"Item Hangar", "Drone Bay"},
	}
	assert.ElementsMatch(t, want, got)
}

func TestNode_AnchestorCount(t *testing.T) {
	top := newCustomNode(NodeItemHangar)
	a := newCustomNode(NodeCargoBay)
	top.addChild(a)
	b := newCustomNode(NodeFuelBay)
	a.addChild(b)
	c := newCustomNode(NodeDroneBay)
	b.addChild(c)

	xassert.Equal(t, 0, top.AncestorCount())
	xassert.Equal(t, 1, a.AncestorCount())
	xassert.Equal(t, 2, b.AncestorCount())
	xassert.Equal(t, 3, c.AncestorCount())
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

func TestNode_ID(t *testing.T) {
	t.Run("should return item ID of assets", func(t *testing.T) {
		n := newAssetNode(&app.CharacterAsset{
			Asset: app.Asset{
				ItemID: 42,
			},
		})
		xassert.Equal(t, 42, n.ID())
	})
	t.Run("should return location ID of locations", func(t *testing.T) {
		n := newLocationNode(&app.EveLocation{
			ID: 123,
		})
		xassert.Equal(t, 123, n.ID())
	})
	t.Run("should return 0 of other nodes", func(t *testing.T) {
		n := newCustomNode(NodeItemHangar)
		xassert.Equal(t, 0, n.ID())
	})
}

func TestNode_CharacterAsset(t *testing.T) {
	// Setup mock data
	validAsset := &app.CharacterAsset{}
	invalidAsset := &app.CorporationAsset{}

	tests := []struct {
		name      string
		item      Item
		wantAsset *app.CharacterAsset
		wantOk    bool
	}{
		{
			name:      "Nil item returns false",
			item:      nil,
			wantAsset: nil,
			wantOk:    false,
		},
		{
			name:      "Wrong type returns false",
			item:      invalidAsset,
			wantAsset: nil,
			wantOk:    false,
		},
		{
			name:      "Valid CharacterAsset returns true",
			item:      validAsset,
			wantAsset: validAsset,
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				item: tt.item,
			}

			got, ok := n.CharacterAsset()

			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.wantAsset, got)
		})
	}
}

func TestNode_CorporationAsset(t *testing.T) {
	// Setup mock data
	validAsset := &app.CorporationAsset{}
	invalidAsset := &app.CharacterAsset{}

	tests := []struct {
		name      string
		item      Item
		wantAsset *app.CorporationAsset
		wantOk    bool
	}{
		{
			name:      "Nil item returns false",
			item:      nil,
			wantAsset: nil,
			wantOk:    false,
		},
		{
			name:      "Wrong type returns false",
			item:      invalidAsset,
			wantAsset: nil,
			wantOk:    false,
		},
		{
			name:      "Valid CharacterAsset returns true",
			item:      validAsset,
			wantAsset: validAsset,
			wantOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				item: tt.item,
			}

			got, ok := n.CorporationAsset()

			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.wantAsset, got)
		})
	}
}
