package eveuniverseservice_test

import (
	"context"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestGetOrCreateEveSchematicESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := eveuniverseservice.NewTestService(st)
	ctx := context.Background()
	t.Run("should return existing schematic", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		x1 := factory.CreateEveSchematic()
		// when
		x2, err := s.GetOrCreateSchematicESI(ctx, x1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
	t.Run("should fetch schematic from ESI and create it", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			"https://esi.evetech.net/v1/universe/schematics/3/",
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"cycle_time":     1800,
				"schematic_name": "Bacteria",
			}))

		// when
		x1, err := s.GetOrCreateSchematicESI(ctx, 3)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(3), x1.ID)
			assert.Equal(t, "Bacteria", x1.Name)
			assert.Equal(t, 1800, x1.CycleTime)
			x2, err := st.GetEveSchematic(ctx, 3)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}
