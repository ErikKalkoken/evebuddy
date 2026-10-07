package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestAugmentationsFilter_Match(t *testing.T) {
	n := &augmentationNode{implantCount: 2, tags: set.Of("alpha")}
	none := &augmentationNode{tags: set.Of("alpha")}
	for _, tc := range []struct {
		name   string
		filter augmentationsFilter
		node   *augmentationNode
		want   bool
	}{
		{"no filter", augmentationsFilter{}, n, true},
		{"has implants matches", augmentationsFilter{implants: augmentationsImplantsSome}, n, true},
		{"has implants but none", augmentationsFilter{implants: augmentationsImplantsSome}, none, false},
		{"no implants matches", augmentationsFilter{implants: augmentationsImplantsNone}, none, true},
		{"no implants but has", augmentationsFilter{implants: augmentationsImplantsNone}, n, false},
		{"tag matches", augmentationsFilter{tag: "alpha"}, n, true},
		{"tag missing", augmentationsFilter{tag: "bravo"}, n, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.node))
		})
	}
}

func TestAugmentations_Filter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	newAugmentations := func(t *testing.T, isMobile bool) *Augmentations {
		a := NewAugmentations(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		td := xwidget.NewTreeData[augmentationNode]()
		require.NoError(t, td.Add(nil, &augmentationNode{characterID: 1, implantCount: 1}, true))
		require.NoError(t, td.Add(nil, &augmentationNode{characterID: 2}, false))
		a.treeData = td
		a.filterTreeAsync()
		return a
	}
	characterIDs := func(a *Augmentations) []int64 {
		return xslices.Map(a.tree.Data().Children(nil), func(n *augmentationNode) int64 {
			return n.characterID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newAugmentations(t, true)
		require.ElementsMatch(t, []int64{1, 2}, characterIDs(a))
		a.filterChip.SetSelected(map[string]string{augmentationsFilterImplants: augmentationsImplantsNone})
		assert.ElementsMatch(t, []int64{2}, characterIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newAugmentations(t, false)
		require.ElementsMatch(t, []int64{1, 2}, characterIDs(a))
		a.selectImplants.SetSelected(augmentationsImplantsNone)
		assert.ElementsMatch(t, []int64{2}, characterIDs(a))
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newAugmentations(t, true)
		assert.Equal(t, map[string]string{
			augmentationsFilterImplants: "",
			augmentationsFilterTag:      "",
		}, a.filterChip.Selected())
	})
}
