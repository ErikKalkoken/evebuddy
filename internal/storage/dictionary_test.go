package storage_test

import (
	"context"
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
			d, ok, err := r.GetDictEntry(ctx, "key")
			if assert.NoError(t, err) {
				assert.True(t, ok)
				assert.Equal(t, []byte("value"), d)
			}
		}
	})
	t.Run("should report false when entry does not exist", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, ok, err := r.GetDictEntry(ctx, "key")
		// then
		if assert.NoError(t, err) {
			assert.False(t, ok)
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
			d, _, err := r.GetDictEntry(ctx, "key")
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
			_, ok, err := r.GetDictEntry(ctx, "key")
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		}
	})
}
