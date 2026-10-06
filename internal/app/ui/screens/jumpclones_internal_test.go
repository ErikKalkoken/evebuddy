package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func makeJumpCloneRow(id int64, character, system, region string, tags ...string) jumpCloneRow {
	return jumpCloneRow{
		jc: &app.CharacterJumpClone2{
			ID:        id,
			Character: &app.EntityShort{Name: character},
			Location: &app.EveLocation{
				SolarSystem: optional.New(&app.EveSolarSystem{
					Name:          system,
					Constellation: &app.EveConstellation{Region: &app.EveRegion{Name: region}},
				}),
			},
		},
		tags: set.Of(tags...),
	}
}

func TestJumpClonesFilter_Match(t *testing.T) {
	r := makeJumpCloneRow(1, "Bruce", "Jita", "The Forge", "alpha")
	for _, tc := range []struct {
		name   string
		filter jumpClonesFilter
		want   bool
	}{
		{"no filter", jumpClonesFilter{}, true},
		{"character matches", jumpClonesFilter{character: "Bruce"}, true},
		{"character differs", jumpClonesFilter{character: "Alice"}, false},
		{"region matches", jumpClonesFilter{region: "The Forge"}, true},
		{"region differs", jumpClonesFilter{region: "Domain"}, false},
		{"system matches", jumpClonesFilter{solarSystem: "Jita"}, true},
		{"system differs", jumpClonesFilter{solarSystem: "Amarr"}, false},
		{"tag matches", jumpClonesFilter{tag: "alpha"}, true},
		{"tag missing", jumpClonesFilter{tag: "bravo"}, false},
		{"all match", jumpClonesFilter{character: "Bruce", region: "The Forge", solarSystem: "Jita", tag: "alpha"}, true},
		{"one of many differs", jumpClonesFilter{character: "Bruce", solarSystem: "Amarr"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(r))
		})
	}
}

func TestJumpClones_Filter(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []jumpCloneRow{
		makeJumpCloneRow(1, "Bruce", "Jita", "The Forge"),
		makeJumpCloneRow(2, "Alice", "Amarr", "Domain"),
	}
	newJumpClones := func(t *testing.T, isMobile bool) *JumpClones {
		a := NewJumpClones(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	cloneIDs := func(a *JumpClones) []int64 {
		return xslices.Map(a.rowsFiltered, func(r jumpCloneRow) int64 {
			return r.jc.ID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newJumpClones(t, true)
		a.filterChip.SetSelected(map[string]string{jumpClonesFilterSystem: "Amarr"})
		assert.ElementsMatch(t, []int64{2}, cloneIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newJumpClones(t, false)
		a.selectSolarSystem.SetSelected("Amarr")
		assert.ElementsMatch(t, []int64{2}, cloneIDs(a))
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newJumpClones(t, true)
		assert.Equal(t, map[string]string{
			jumpClonesFilterCharacter: "",
			jumpClonesFilterRegion:    "",
			jumpClonesFilterSystem:    "",
			jumpClonesFilterTag:       "",
		}, a.filterChip.Selected())
	})
}
