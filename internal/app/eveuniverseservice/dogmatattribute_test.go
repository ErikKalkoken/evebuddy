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

func TestGetOrCreateEveDogmaAttributeESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing object", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		x1 := factory.CreateEveDogmaAttribute()
		// when
		x2, err := s.GetOrCreateDogmaAttributeESI(ctx, x1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x2, x1)
		}
	})
	t.Run("should create new object from ESI when it does not exist", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/dogma/attributes/20/",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"attribute_id":  20,
				"default_value": 1,
				"description":   "Factor by which top speed increases.",
				"display_name":  "Maximum Velocity Bonus",
				"high_is_good":  true,
				"icon_id":       1389,
				"name":          "speedFactor",
				"published":     true,
				"unit_id":       124,
			}))
		// when
		x1, err := s.GetOrCreateDogmaAttributeESI(ctx, 20)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(20), x1.ID)
			assert.Equal(t, float32(1), x1.DefaultValue)
			assert.Equal(t, "Factor by which top speed increases.", x1.Description)
			assert.Equal(t, "Maximum Velocity Bonus", x1.DisplayName)
			assert.Equal(t, int32(1389), x1.IconID)
			assert.Equal(t, "speedFactor", x1.Name)
			assert.True(t, x1.IsHighGood)
			assert.True(t, x1.IsPublished)
			assert.False(t, x1.IsStackable)
			assert.Equal(t, app.EveUnitID(124), x1.Unit)
			x2, err := st.GetEveDogmaAttribute(ctx, 20)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}
