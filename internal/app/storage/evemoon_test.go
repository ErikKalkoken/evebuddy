package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveMoon(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		system := factory.CreateEveSolarSystem()
		arg := storage.CreateEveMoonParams{
			ID:            42,
			Name:          "name",
			SolarSystemID: system.ID,
		}
		// when
		err := st.CreateEveMoon(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetEveMoon(ctx, 42)
			if assert.NoError(t, err) {
				xassert.Equal(t, 42, o.ID)
				xassert.Equal(t, "name", o.Name)
				xassert.Equal(t, system, o.SolarSystem)
			}
		}
	})
}
