package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestEveType(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		g := factory.CreateEveGroup()
		arg := storage.CreateEveTypeParams{
			ID:          42,
			Description: "description",
			GroupID:     g.ID,
			Name:        "name",
			IsPublished: true,
		}
		// when
		err := r.CreateEveType(ctx, arg)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetEveType(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), x.ID)
				assert.Equal(t, "name", x.Name)
				assert.Equal(t, true, x.IsPublished)
				assert.Equal(t, g, x.Group)
			}
		}
	})
}
