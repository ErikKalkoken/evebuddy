package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveRace(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		arg := storage.CreateEveRaceParams{
			ID:          42,
			Description: "description",
			Name:        "name",
		}
		x1, err := st.CreateEveRace(ctx, arg)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 42, x1.ID)
		xassert.Equal(t, "description", x1.Description)
		xassert.Equal(t, "name", x1.Name)
		x2, err := st.GetEveRace(ctx, 42)
		require.NoError(t, err)
		xassert.Equal(t, *x1, *x2)
	})
}
