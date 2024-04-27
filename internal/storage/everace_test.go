package storage_test

import (
	"context"
	"example/evebuddy/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveRace(t *testing.T) {
	db, r, _ := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		x1, err := r.CreateEveRace(ctx, 42, "description", "name")
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
