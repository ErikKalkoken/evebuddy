package eveuniverseservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetOrCreateEveRaceESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing race", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		x1 := factory.CreateEveRace(app.EveRace{ID: 7})
		// when
		x2, err := s.GetOrCreateRaceESI(ctx, 7)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
	t.Run("should create race from ESI when it does not exit in DB", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/races/",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id": 500001,
					"description": "Founded on the tenets of patriotism and hard work...",
					"name":        "Caldari",
					"race_id":     7,
				},
			}))

		// when
		x1, err := s.GetOrCreateRaceESI(ctx, 7)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "Caldari", x1.Name)
			assert.Equal(t, "Founded on the tenets of patriotism and hard work...", x1.Description)
			x2, err := st.GetEveRace(ctx, 7)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
	t.Run("should return specific error when race ID is invalid", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/races/",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"alliance_id": 500001,
					"description": "Founded on the tenets of patriotism and hard work...",
					"name":        "Caldari",
					"race_id":     7,
				},
			}))

		// when
		_, err := s.GetOrCreateRaceESI(ctx, 42)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}
