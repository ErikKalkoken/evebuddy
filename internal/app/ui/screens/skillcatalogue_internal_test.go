package screens

import (
	"bytes"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestSkillsForClipboard(t *testing.T) {
	t.Run("formats sorted active skills as name and level", func(t *testing.T) {
		rows := []skillCatalogueRow{
			{name: "Gunnery", levelActive: 4},
			{name: "Caldari Frigate", levelActive: 5},
			{name: "Spaceship Command", levelActive: 3},
		}
		got, err := skillsForClipboard(rows)
		require.NoError(t, err)
		want := "Caldari Frigate 5\nGunnery 4\nSpaceship Command 3\n"
		assert.Equal(t, want, got)
	})
	t.Run("skips skills that are not active", func(t *testing.T) {
		rows := []skillCatalogueRow{
			{name: "Gunnery", levelActive: 4},
			{name: "Caldari Frigate", levelActive: 0},
		}
		got, err := skillsForClipboard(rows)
		require.NoError(t, err)
		want := "Gunnery 4\n"
		assert.Equal(t, want, got)
	})
	t.Run("returns empty string for no rows", func(t *testing.T) {
		got, err := skillsForClipboard(nil)
		require.NoError(t, err)
		assert.Equal(t, "", got)
	})
}

func TestWriteSkillCatalogueRowsToCSV(t *testing.T) {
	t.Run("writes sorted active skills as CSV", func(t *testing.T) {
		rows := []skillCatalogueRow{
			{name: "Gunnery", levelActive: 4},
			{name: "Caldari Frigate", levelActive: 5},
			{name: "Spaceship Command", levelActive: 3},
		}
		var b bytes.Buffer
		err := writeSkillCatalogueRowsToCSV(&b, rows)
		require.NoError(t, err)
		want := "Name,Level\nCaldari Frigate,5\nGunnery,4\nSpaceship Command,3\n"
		assert.Equal(t, want, b.String())
	})
	t.Run("skips skills that are not active", func(t *testing.T) {
		rows := []skillCatalogueRow{
			{name: "Gunnery", levelActive: 4},
			{name: "Caldari Frigate", levelActive: 0},
		}
		var b bytes.Buffer
		err := writeSkillCatalogueRowsToCSV(&b, rows)
		require.NoError(t, err)
		want := "Name,Level\nGunnery,4\n"
		assert.Equal(t, want, b.String())
	})
	t.Run("no rows returns only header", func(t *testing.T) {
		var b bytes.Buffer
		err := writeSkillCatalogueRowsToCSV(&b, nil)
		require.NoError(t, err)
		assert.Equal(t, "Name,Level\n", b.String())
	})
}

func TestSkillCatalogueFilter_Match(t *testing.T) {
	trained := skillCatalogueRow{groupName: "Gunnery", levelActive: 3, levelTrained: 3}
	untrained := skillCatalogueRow{groupName: "Gunnery", hasPrerequisites: true}
	queued := skillCatalogueRow{groupName: "Gunnery", levelQueued: 1}
	maxed := skillCatalogueRow{groupName: "Gunnery", hasPrerequisites: true, levelActive: 5, levelTrained: 5}
	for _, tc := range []struct {
		name   string
		filter skillCatalogueFilter
		row    skillCatalogueRow
		want   bool
	}{
		{"no filter", skillCatalogueFilter{}, untrained, true},
		{"group matches", skillCatalogueFilter{group: "Gunnery"}, trained, true},
		{"group differs", skillCatalogueFilter{group: "Navigation"}, trained, false},
		{"trained matches", skillCatalogueFilter{training: skillCatalogueTrained}, trained, true},
		{"trained but untrained", skillCatalogueFilter{training: skillCatalogueTrained}, untrained, false},
		{"have prerequisites matches", skillCatalogueFilter{training: skillCatalogueHavePrerequisites}, untrained, true},
		{"have prerequisites but maxed", skillCatalogueFilter{training: skillCatalogueHavePrerequisites}, maxed, false},
		{"queued matches", skillCatalogueFilter{training: skillCatalogueQueued}, queued, true},
		{"queued but not queued", skillCatalogueFilter{training: skillCatalogueQueued}, trained, false},
		{"fully trained matches", skillCatalogueFilter{training: skillCatalogueFullyTrained}, maxed, true},
		{"fully trained but not", skillCatalogueFilter{training: skillCatalogueFullyTrained}, trained, false},
		{"training and group combined", skillCatalogueFilter{group: "Navigation", training: skillCatalogueTrained}, trained, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestSkillCatalogue_Filter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []skillCatalogueRow{
		{typeID: 1, name: "Gunnery", groupName: "Gunnery", levelActive: 1, levelTrained: 1, searchTarget: "gunnery"},
		{typeID: 2, name: "Navigation", groupName: "Navigation", searchTarget: "navigation"},
	}
	newCatalogue := func(t *testing.T, isMobile bool) *SkillCatalogue {
		a := NewSkillCatalogue(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync()
		require.Len(t, a.rowsFiltered, 2) // all skills by default
		return a
	}
	typeIDs := func(a *SkillCatalogue) []int64 {
		return xslices.Map(a.rowsFiltered, func(r skillCatalogueRow) int64 {
			return r.typeID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newCatalogue(t, true)
		a.filterChip.SetSelected(map[string]string{skillCatalogueFilterGroup: "Navigation"})
		assert.ElementsMatch(t, []int64{2}, typeIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newCatalogue(t, false)
		a.selectGroup.SetSelected("Navigation")
		assert.ElementsMatch(t, []int64{2}, typeIDs(a))
	})
	t.Run("can filter training on mobile", func(t *testing.T) {
		a := newCatalogue(t, true)
		a.filterChip.SetSelected(map[string]string{skillCatalogueFilterTraining: skillCatalogueTrained})
		assert.ElementsMatch(t, []int64{1}, typeIDs(a))
	})
	t.Run("can filter training on desktop", func(t *testing.T) {
		a := newCatalogue(t, false)
		a.selectTraining.SetSelected(skillCatalogueTrained)
		assert.ElementsMatch(t, []int64{1}, typeIDs(a))
	})
	t.Run("can search on mobile", func(t *testing.T) {
		a := newCatalogue(t, true)
		a.searchEntry.SetText("navi")
		assert.ElementsMatch(t, []int64{2}, typeIDs(a))
	})
}
