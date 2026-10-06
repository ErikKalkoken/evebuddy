package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestSkillNameCondensed(t *testing.T) {
	tests := []struct {
		name string
		row  skillSearchRow
		want string
	}{
		{
			name: "active level zero and trained level greater than zero",
			row: skillSearchRow{
				typeName:          "Gunnery",
				activeLevel:       0,
				activeLevelRoman:  "0",
				trainedLevel:      3,
				trainedLevelRoman: "III",
			},
			want: "Gunnery [III]",
		},
		{
			name: "active level equal to trained level",
			row: skillSearchRow{
				typeName:          "Gunnery",
				activeLevel:       5,
				activeLevelRoman:  "V",
				trainedLevel:      5,
				trainedLevelRoman: "V",
			},
			want: "Gunnery V",
		},
		{
			name: "active level different from trained level",
			row: skillSearchRow{
				typeName:          "Gunnery",
				activeLevel:       3,
				activeLevelRoman:  "III",
				trainedLevel:      5,
				trainedLevelRoman: "V",
			},
			want: "Gunnery III [V]",
		},
		{
			name: "both levels zero",
			row: skillSearchRow{
				typeName:          "Gunnery",
				activeLevel:       0,
				activeLevelRoman:  "-",
				trainedLevel:      0,
				trainedLevelRoman: "-",
			},
			want: "Gunnery -",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.row.skillNameCondensed()
			xassert.Equal(t, tt.want, got)
		})
	}
}

func TestSkillSearchFilter_Match(t *testing.T) {
	r := skillSearchRow{
		activeLevel:   3,
		characterName: "Bruce",
		groupName:     "Gunnery",
		trainedLevel:  3,
		typeName:      "Small Hybrid Turret",
	}
	restricted := r
	restricted.activeLevel = 1 // e.g. alpha clone
	inactive := r
	inactive.activeLevel = 0
	for _, tc := range []struct {
		name   string
		filter skillSearchFilter
		row    skillSearchRow
		want   bool
	}{
		{"no filter", skillSearchFilter{}, r, true},
		{"character matches", skillSearchFilter{character: "Bruce"}, r, true},
		{"character differs", skillSearchFilter{character: "Alice"}, r, false},
		{"group differs", skillSearchFilter{group: "Navigation"}, r, false},
		{"type differs", skillSearchFilter{typeName: "Afterburner"}, r, false},
		{"active matches", skillSearchFilter{skill: searchSkillActive}, r, true},
		{"active but inactive", skillSearchFilter{skill: searchSkillActive}, inactive, false},
		{"restricted matches", skillSearchFilter{skill: searchSkillRestricted}, restricted, true},
		{"restricted but not", skillSearchFilter{skill: searchSkillRestricted}, r, false},
		{"all includes inactive", skillSearchFilter{skill: searchSkillAll}, inactive, true},
		{"all match", skillSearchFilter{character: "Bruce", group: "Gunnery", skill: searchSkillActive, typeName: "Small Hybrid Turret"}, r, true},
		{"one of many differs", skillSearchFilter{character: "Bruce", group: "Navigation"}, r, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestSkillSearch_Filter(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []skillSearchRow{
		{typeID: 1, activeLevel: 1, trainedLevel: 1, groupName: "Gunnery"},
		{typeID: 2, activeLevel: 1, trainedLevel: 1, groupName: "Navigation"},
		{typeID: 3, activeLevel: 0, trainedLevel: 1, groupName: "Gunnery"},
	}
	newSkillSearch := func(t *testing.T, isMobile bool) *SkillSearch {
		a := NewSkillSearch(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2) // active by default
		return a
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newSkillSearch(t, true)
		a.filterChip.SetSelected(map[string]string{skillSearchFilterGroup: "Gunnery"})
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 1, a.rowsFiltered[0].typeID)
		}
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newSkillSearch(t, false)
		a.selectGroup.SetSelected("Gunnery")
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 1, a.rowsFiltered[0].typeID)
		}
	})
	t.Run("can combine skill mode and filter on mobile", func(t *testing.T) {
		a := newSkillSearch(t, true)
		a.selectSkill.SetSelected(searchSkillAll)
		a.filterChip.SetSelected(map[string]string{skillSearchFilterGroup: "Gunnery"})
		assert.Len(t, a.rowsFiltered, 2)
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newSkillSearch(t, true)
		assert.Equal(t, map[string]string{
			skillSearchFilterCharacter: "",
			skillSearchFilterGroup:     "",
			skillSearchFilterType:      "",
		}, a.filterChip.Selected())
	})
}
