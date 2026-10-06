package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCharacterLoyaltyPointsFilter_Match(t *testing.T) {
	r := characterLoyaltyPointsRow{factionName: "Caldari State"}
	assert.True(t, characterLoyaltyPointsFilter{}.match(r))
	assert.True(t, characterLoyaltyPointsFilter{faction: "Caldari State"}.match(r))
	assert.False(t, characterLoyaltyPointsFilter{faction: "Amarr Empire"}.match(r))
}

func TestCharacterLoyaltyPoints_Filter(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []characterLoyaltyPointsRow{
		{corporationID: 1, corporationName: "Caldari Navy", factionName: "Caldari State", searchTarget: "caldari navy"},
		{corporationID: 2, corporationName: "Amarr Navy", factionName: "Amarr Empire", searchTarget: "amarr navy"},
	}
	newLoyaltyPoints := func(t *testing.T, isMobile bool) *CharacterLoyaltyPoints {
		a := NewCharacterLoyaltyPoints(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync()
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	corporationIDs := func(a *CharacterLoyaltyPoints) []int64 {
		return xslices.Map(a.rowsFiltered, func(r characterLoyaltyPointsRow) int64 {
			return r.corporationID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newLoyaltyPoints(t, true)
		a.filterChip.SetSelected(map[string]string{characterLoyaltyPointsFilterFaction: "Amarr Empire"})
		assert.ElementsMatch(t, []int64{2}, corporationIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newLoyaltyPoints(t, false)
		a.selectFaction.SetSelected("Amarr Empire")
		assert.ElementsMatch(t, []int64{2}, corporationIDs(a))
	})
	t.Run("can search on mobile", func(t *testing.T) {
		a := newLoyaltyPoints(t, true)
		a.searchEntry.SetText("calda")
		assert.ElementsMatch(t, []int64{1}, corporationIDs(a))
	})
	t.Run("resets filter when character changes on mobile", func(t *testing.T) {
		a := newLoyaltyPoints(t, true)
		a.filterChip.SetSelected(map[string]string{characterLoyaltyPointsFilterFaction: "Amarr Empire"})
		require.True(t, a.filterChip.IsOn())

		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), factory.CreateCharacter())

		assert.False(t, a.filterChip.IsOn())
	})
}
