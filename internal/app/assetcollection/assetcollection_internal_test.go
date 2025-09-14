package assetcollection

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/stretchr/testify/assert"
)

func TestBaseNode(t *testing.T) {
	a := &app.CharacterAsset{ItemID: 1, LocationID: 100000}
	b := &app.CharacterAsset{ItemID: 2, LocationID: 100000}
	c := &app.CharacterAsset{ItemID: 3, LocationID: 100000}
	d := &app.CharacterAsset{ItemID: 4, LocationID: 100000}
	el := &app.EveLocation{ID: 100000, Name: "Alpha"}
	ln := newLocationNode(el)
	n1 := ln.add(a)
	ln.add(b)
	n1.add(c)
	n1.add(d)
	t.Run("can return nodes", func(t *testing.T) {
		got := ln.Children()
		assert.ElementsMatch(t, []int{1, 2}, xslices.Map(got, func(a *AssetNode) int {
			return int(a.Asset.ItemID)
		}))
	})
	t.Run("can count size", func(t *testing.T) {
		got := ln.Size()
		assert.Equal(t, 3, got)
	})
}
