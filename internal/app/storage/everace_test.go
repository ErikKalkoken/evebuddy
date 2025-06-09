package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestEveRace(t *testing.T) {
	db, r, _ := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		arg := storage.CreateEveRaceParams{
			ID:          42,
			Description: "description",
			Name:        "name",
		}
		x1, err := r.CreateEveRace(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(42), x1.ID)
			assert.Equal(t, "description", x1.Description)
			assert.Equal(t, "name", x1.Name)
			x2, err := r.GetEveRace(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, *x1, *x2)
			}
		}
	})

}
