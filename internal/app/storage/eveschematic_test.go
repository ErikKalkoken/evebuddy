package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestEveSchematic(t *testing.T) {
	db, r, _ := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		arg := storage.CreateEveSchematicParams{
			ID:        42,
			Name:      "name",
			CycleTime: 7,
		}
		// when
		c1, err := r.CreateEveSchematic(ctx, arg)
		// then
		if assert.NoError(t, err) {
			c2, err := r.GetEveSchematic(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, c1, c2)
			}
		}
	})
}
