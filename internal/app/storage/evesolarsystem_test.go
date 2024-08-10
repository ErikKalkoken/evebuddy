package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestEveSolarSystem(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateEveConstellation()
		arg := storage.CreateEveSolarSystemParams{
			ID:              42,
			ConstellationID: c.ID,
			Name:            "name",
			SecurityStatus:  -8.5,
		}
		// when
		err := r.CreateEveSolarSystem(ctx, arg)
		// then
		if assert.NoError(t, err) {
			g, err := r.GetEveSolarSystem(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), g.ID)
				assert.Equal(t, "name", g.Name)
				assert.Equal(t, c, g.Constellation)
				assert.Equal(t, float32(-8.5), g.SecurityStatus)
			}
		}
	})
}
