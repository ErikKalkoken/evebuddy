package characterservice_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestSearchESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("should return search results", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		x1 := factory.CreateEveEntityCharacter()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/search/?categories=character&search=search&strict=false", c.ID),
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
		assert.Equal(t, 1, n)
		assert.Equal(t, map[app.SearchCategory][]*app.EveEntity{app.SearchCharacter: {x1}}, got)
	})
}
