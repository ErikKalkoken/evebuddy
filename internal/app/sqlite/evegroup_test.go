package sqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/testutil"
)

func TestEveGroup(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateEveCategory()
		arg := sqlite.CreateEveGroupParams{
			ID:          42,
			Name:        "name",
			CategoryID:  c.ID,
			IsPublished: true,
		}
		// when
		err := r.CreateEveGroup(ctx, arg)
		// then
		if assert.NoError(t, err) {
			g, err := r.GetEveGroup(ctx, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, int32(42), g.ID)
				assert.Equal(t, "name", g.Name)
				assert.Equal(t, true, g.IsPublished)
				assert.Equal(t, c, g.Category)
			}
		}
	})
}
