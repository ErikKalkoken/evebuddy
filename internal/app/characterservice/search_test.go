package characterservice_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestSearchESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("should return search results", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		x1 := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/search?categories=character&search=search&strict=false", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string][]int{
				"agent":          {},
				"alliance":       {},
				"character":      {int(x1.ID)},
				"constellation":  {},
				"corporation":    {},
				"faction":        {},
				"inventory_type": {},
				"region":         {},
				"solar_system":   {},
				"station":        {},
				"structure":      {},
			}),
		)
		// when
		got, n, err := s.SearchESI(ctx, "search", []app.SearchCategory{app.SearchCharacter}, false)
		// then
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		xassert.Equal(t, 1, n)
		xassert.Equal(t, map[app.SearchCategory][]*app.EveEntity{app.SearchCharacter: {x1}}, got)
	})
}

func TestAddEveEntitiesFromSearchESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("should add missing entities found via search", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		const characterID2 = 3007
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/search?categories=corporation&categories=character&categories=alliance&search=abc", c.ID),
			httpmock.NewJsonResponderOrPanic(200, map[string][]int{
				"character": {characterID2},
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"id":       characterID2,
				"name":     "Bruce Wayne",
				"category": "character",
			}}),
		)
		// when
		got, err := s.AddEveEntitiesFromSearchESI(ctx, c.ID, "abc")
		// then
		require.NoError(t, err)
		assert.True(t, got.Contains(characterID2))
	})
}
