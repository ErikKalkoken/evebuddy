package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestEveDogmaAttribute(t *testing.T) {
	db, r, _ := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		arg := storage.CreateEveDogmaAttributeParams{
			ID:           42,
			DefaultValue: 1.2,
			Description:  "description",
			DisplayName:  "display name",
			IconID:       7,
			Name:         "name",
			IsHighGood:   true,
			IsPublished:  true,
			IsStackable:  true,
			UnitID:       99,
		}
		// when
		x1, err := r.CreateEveDogmaAttribute(ctx, arg)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, int32(42), x1.ID)
			assert.Equal(t, float32(1.2), x1.DefaultValue)
			assert.Equal(t, "description", x1.Description)
			assert.Equal(t, "display name", x1.DisplayName)
			assert.Equal(t, int32(7), x1.IconID)
			assert.Equal(t, "name", x1.Name)
			assert.True(t, x1.IsHighGood)
			assert.True(t, x1.IsPublished)
			assert.True(t, x1.IsStackable)
			assert.Equal(t, int32(99), x1.UnitID)
			x2, err := r.GetEveDogmaAttribute(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, x1, x2)
			}
		}
	})
}
