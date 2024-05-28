package storage_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
)

func TestDictionary(t *testing.T) {
	db, r, _ := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		err := r.SetDictEntry(ctx, "key", []byte("value"))
		// then
		if assert.NoError(t, err) {
			d, err := r.GetDictEntry(ctx, "key")
			if assert.NoError(t, err) {
				assert.Equal(t, []byte("value"), d)
			}
		}
	})
	t.Run("can overwrite existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		err := r.SetDictEntry(ctx, "key", []byte("value1"))
		if err != nil {
			t.Fatal(err)
		}
		// when
		err = r.SetDictEntry(ctx, "key", []byte("value2"))
		// then
		if assert.NoError(t, err) {
			d, err := r.GetDictEntry(ctx, "key")
			if assert.NoError(t, err) {
				assert.Equal(t, []byte("value2"), d)
			}
		}
	})
	t.Run("can delete entry", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		err := r.SetDictEntry(ctx, "key", []byte("value1"))
		if err != nil {
			t.Fatal(err)
		}
		// when
		err = r.DeleteDictEntry(ctx, "key")
		// then
		if assert.NoError(t, err) {
			_, err := r.GetDictEntry(ctx, "key")
			assert.ErrorIs(t, err, sql.ErrNoRows)
		}
	})
}
