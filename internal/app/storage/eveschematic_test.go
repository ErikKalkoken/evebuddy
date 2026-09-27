package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveSchematic(t *testing.T) {
	db, r, _ := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		arg := storage.CreateEveSchematicParams{
			ID:        42,
			Name:      "name",
			CycleTime: 7,
		}
		// when
		c1, err := r.CreateEveSchematic(ctx, arg)
		// then
		require.NoError(t, err)
		c2, err := r.GetEveSchematic(ctx, 42)
		require.NoError(t, err)
		xassert.Equal(t, c1, c2)
	})
}
