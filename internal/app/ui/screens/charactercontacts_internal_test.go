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
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCharacterContactsFilter_Match(t *testing.T) {
	r := characterContactRow{
		blockedSelect:    "no",
		category:         "Character",
		labels:           set.Of("friends"),
		npcSelect:        "no",
		standingCategory: app.GoodStanding,
		watchedSelect:    "yes",
	}
	for _, tc := range []struct {
		name   string
		filter characterContactsFilter
		want   bool
	}{
		{"no filter", characterContactsFilter{}, true},
		{"blocked matches", characterContactsFilter{blocked: "no"}, true},
		{"blocked differs", characterContactsFilter{blocked: "yes"}, false},
		{"category differs", characterContactsFilter{category: "Corporation"}, false},
		{"label matches", characterContactsFilter{label: "friends"}, true},
		{"label missing", characterContactsFilter{label: "enemies"}, false},
		{"npc differs", characterContactsFilter{npc: "yes"}, false},
		{"standing matches", characterContactsFilter{standing: app.GoodStanding.String()}, true},
		{"standing differs", characterContactsFilter{standing: app.BadStanding.String()}, false},
		{"watched differs", characterContactsFilter{watched: "no"}, false},
		{"all match", characterContactsFilter{blocked: "no", category: "Character", label: "friends", npc: "no", standing: app.GoodStanding.String(), watched: "yes"}, true},
		{"one of many differs", characterContactsFilter{category: "Character", watched: "no"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(r))
		})
	}
}

func TestCharacterContacts_Filter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []characterContactRow{
		{contact: &app.EveEntity{ID: 1, Name: "Alice"}, category: "Character", searchTarget: "alice"},
		{contact: &app.EveEntity{ID: 2, Name: "Bruce"}, category: "Corporation", searchTarget: "bruce", isWatched: optional.New(true), watchedSelect: "yes"},
	}
	newContacts := func(t *testing.T, isMobile bool) *CharacterContacts {
		a := NewCharacterContacts(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync()
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	contactIDs := func(a *CharacterContacts) []int64 {
		return xslices.Map(a.rowsFiltered, func(r characterContactRow) int64 {
			return r.contact.ID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newContacts(t, true)
		a.filterChip.SetSelected(map[string]string{characterContactsFilterCategory: "Corporation"})
		assert.ElementsMatch(t, []int64{2}, contactIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newContacts(t, false)
		a.selectCategory.SetSelected("Corporation")
		assert.ElementsMatch(t, []int64{2}, contactIDs(a))
	})
	t.Run("can filter by watched on mobile", func(t *testing.T) {
		a := newContacts(t, true)
		a.filterChip.SetSelected(map[string]string{characterContactsFilterWatched: "yes"})
		assert.ElementsMatch(t, []int64{2}, contactIDs(a))
	})
	t.Run("can search on mobile", func(t *testing.T) {
		a := newContacts(t, true)
		a.searchEntry.SetText("ali")
		assert.ElementsMatch(t, []int64{1}, contactIDs(a))
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newContacts(t, true)
		assert.Equal(t, map[string]string{
			characterContactsFilterBlocked:  "",
			characterContactsFilterCategory: "",
			characterContactsFilterLabel:    "",
			characterContactsFilterNPC:      "",
			characterContactsFilterStanding: "",
			characterContactsFilterWatched:  "",
		}, a.filterChip.Selected())
	})
	t.Run("resets filters when character changes on mobile", func(t *testing.T) {
		a := newContacts(t, true)
		a.filterChip.SetSelected(map[string]string{characterContactsFilterCategory: "Corporation"})
		require.True(t, a.filterChip.IsOn())

		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), factory.CreateCharacter())

		assert.False(t, a.filterChip.IsOn())
	})
}
