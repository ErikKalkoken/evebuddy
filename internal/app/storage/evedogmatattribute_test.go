package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveDogmaAttribute(t *testing.T) {
	db, r, _ := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		unit := app.EveUnitID(99)
		arg := storage.CreateEveDogmaAttributeParams{
			ID:           42,
			DefaultValue: optional.New(1.2),
			Description:  optional.New("description"),
			DisplayName:  optional.New("display name"),
			IconID:       optional.New[int64](7),
			Name:         optional.New("name"),
			IsHighGood:   optional.New(true),
			IsPublished:  optional.New(true),
			IsStackable:  optional.New(true),
			UnitID:       unit,
		}
		// when
		x1, err := r.CreateEveDogmaAttribute(ctx, arg)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, 42, x1.ID)
			xassert.Equal(t, 1.2, x1.DefaultValue.ValueOrZero())
			xassert.Equal(t, "description", x1.Description.ValueOrZero())
			xassert.Equal(t, "display name", x1.DisplayName.ValueOrZero())
			xassert.Equal(t, 7, x1.IconID.ValueOrZero())
			xassert.Equal(t, "name", x1.Name.ValueOrZero())
			assert.True(t, x1.IsHighGood.ValueOrZero())
			assert.True(t, x1.IsPublished.ValueOrZero())
			assert.True(t, x1.IsStackable.ValueOrZero())
			xassert.Equal(t, unit, x1.Unit)
			x2, err := r.GetEveDogmaAttribute(ctx, 42)
			if assert.NoError(t, err) {
				xassert.Equal(t, x1, x2)
			}
		}
	})
}
