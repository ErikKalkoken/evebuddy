package storage_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestPlanetPinContent(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can get and create", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		pin := factory.CreatePlanetPin()
		input := factory.CreateEveType()
		arg := storage.CreatePlanetPinContentParams{
			Amount:      42,
			EveTypeID:   input.ID,
			PlanetPinID: pin.ID,
		}
		// when
		err := r.CreatePlanetPinContent(ctx, arg)
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetPlanetPinContent(ctx, pin.ID, input.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, input, c2.Type)
			}
		}
	})
	t.Run("can list contents", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		p := factory.CreatePlanetPin()
		x1 := factory.CreatePlanetPinContent(storage.CreatePlanetPinContentParams{PlanetPinID: p.ID})
		x2 := factory.CreatePlanetPinContent(storage.CreatePlanetPinContentParams{PlanetPinID: p.ID})
		// when
		oo, err := r.ListPlanetPinContents(ctx, p.ID)
		// then
		if assert.NoError(t, err) {
			got := make([]int32, 0)
			for _, o := range oo {
				got = append(got, o.Type.ID)
			}
			assert.ElementsMatch(t, []int32{x1.Type.ID, x2.Type.ID}, got)
		}
	})
}
