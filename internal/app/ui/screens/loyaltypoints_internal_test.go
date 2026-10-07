package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestLoyaltyPointsFilter_Match(t *testing.T) {
	corp := &loyaltyPointsNode{factionName: "Caldari State", isCorporation: true}
	character := &loyaltyPointsNode{characterName: "Bruce", tags: set.Of("alpha")}
	t.Run("corporation", func(t *testing.T) {
		assert.True(t, loyaltyPointsFilter{}.matchCorporation(corp))
		assert.True(t, loyaltyPointsFilter{faction: "Caldari State"}.matchCorporation(corp))
		assert.False(t, loyaltyPointsFilter{faction: "Amarr Empire"}.matchCorporation(corp))
	})
	t.Run("character", func(t *testing.T) {
		assert.True(t, loyaltyPointsFilter{}.matchCharacter(character))
		assert.True(t, loyaltyPointsFilter{character: "Bruce", tag: "alpha"}.matchCharacter(character))
		assert.False(t, loyaltyPointsFilter{character: "Alice"}.matchCharacter(character))
		assert.False(t, loyaltyPointsFilter{tag: "bravo"}.matchCharacter(character))
	})
}

func TestLoyaltyPoints_Filter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	newData := func() map[*loyaltyPointsNode][]*loyaltyPointsNode {
		c1 := &loyaltyPointsNode{corporationID: 1, corporationName: "Caldari Navy", factionName: "Caldari State", isCorporation: true, searchTarget: "caldari navy"}
		c2 := &loyaltyPointsNode{corporationID: 2, corporationName: "Amarr Navy", factionName: "Amarr Empire", isCorporation: true, searchTarget: "amarr navy"}
		return map[*loyaltyPointsNode][]*loyaltyPointsNode{
			c1: {{characterID: 1, corporationID: 1, characterName: "Bruce", points: 10}},
			c2: {{characterID: 2, corporationID: 2, characterName: "Alice", points: 20}},
		}
	}
	newLoyaltyPoints := func(t *testing.T, isMobile bool) *LoyaltyPoints {
		a := NewLoyaltyPoints(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.data = newData()
		a.filterTreeAsync()
		assert.Equal(t, "Showing 2 / 2 corporations", a.footer.Text)
		return a
	}
	t.Run("can filter by faction on mobile", func(t *testing.T) {
		a := newLoyaltyPoints(t, true)
		a.filterChip.SetSelected(map[string]string{loyaltyPointsFilterFaction: "Amarr Empire"})
		assert.Equal(t, "Showing 1 / 2 corporations", a.footer.Text)
	})
	t.Run("can filter by character on mobile", func(t *testing.T) {
		a := newLoyaltyPoints(t, true)
		a.filterChip.SetSelected(map[string]string{loyaltyPointsFilterCharacter: "Bruce"})
		assert.Equal(t, "Showing 1 / 2 corporations", a.footer.Text)
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newLoyaltyPoints(t, false)
		a.selectFaction.SetSelected("Amarr Empire")
		assert.Equal(t, "Showing 1 / 2 corporations", a.footer.Text)
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newLoyaltyPoints(t, true)
		assert.Equal(t, map[string]string{
			loyaltyPointsFilterCharacter: "",
			loyaltyPointsFilterFaction:   "",
			loyaltyPointsFilterTag:       "",
		}, a.filterChip.Selected())
	})
}
