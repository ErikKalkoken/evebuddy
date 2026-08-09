package storage_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCacheGet(t *testing.T) {
	db, r, _ := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can get and existing entry", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		expiresAt := time.Now().Add(time.Minute)

		// when
		err := r.CacheSet(t.Context(), storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})

		// then
		require.NoError(t, err)
		got1, got2, err := r.CacheGet(t.Context(), key)
		require.NoError(t, err)
		xassert.Equal(t, value, got1)
		xassert.Equal(t, expiresAt, got2)
	})

	t.Run("should treat expired entries as non existent for get", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		expiresAt := time.Now().Add(-time.Minute)
		err := r.CacheSet(t.Context(), storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})
		require.NoError(t, err)
		// when
		_, _, err = r.CacheGet(t.Context(), key)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})

	t.Run("should return entries with get which never expiry", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		err := r.CacheSet(t.Context(), storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: time.Time{},
		})
		require.NoError(t, err)
		// when
		got1, got2, err := r.CacheGet(t.Context(), key)
		// then
		require.NoError(t, err)
		xassert.Equal(t, value, got1)
		assert.True(t, got2.IsZero())
	})
	t.Run("should return error when key is empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, _, err := r.CacheGet(t.Context(), "")
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}

func TestCacheSet(t *testing.T) {
	db, r, _ := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can update existing entry", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		err := r.CacheSet(t.Context(), storage.CacheSetParams{
			Key:       key,
			Value:     []byte("old-value"),
			ExpiresAt: time.Now().Add(3 * time.Minute),
		})
		require.NoError(t, err)
		value := []byte("value")
		expiresAt := time.Now().Add(5 * time.Minute)

		// when
		err = r.CacheSet(t.Context(), storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})

		// then
		require.NoError(t, err)
		got1, got2, err := r.CacheGet(t.Context(), key)
		require.NoError(t, err)
		xassert.Equal(t, value, got1)
		xassert.Equal(t, expiresAt, got2)
	})
	t.Run("should return error when key is empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := r.CacheSet(t.Context(), storage.CacheSetParams{})
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}

func TestCacheOther(t *testing.T) {
	db, r, _ := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("can delete entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		expiresAt := time.Now().Add(time.Minute)
		err := r.CacheSet(t.Context(), storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})
		require.NoError(t, err)
		// when
		err = r.CacheDelete(t.Context(), key)
		// then
		require.NoError(t, err)
		_, _, err2 := r.CacheGet(t.Context(), key)
		assert.Error(t, err2, app.ErrNotFound)
	})

	t.Run("should return error when delete and key is empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := r.CacheDelete(t.Context(), "")
		assert.ErrorIs(t, err, app.ErrInvalid)
	})

	t.Run("can clear cache", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		expiresAt := time.Now().Add(time.Minute)
		err := r.CacheSet(t.Context(), storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})
		require.NoError(t, err)
		// when
		err = r.CacheClear(t.Context())
		// then
		require.NoError(t, err)
		_, _, err2 := r.CacheGet(t.Context(), key)
		assert.Error(t, err2, app.ErrNotFound)
	})

	t.Run("can remove all expired entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		now := time.Now()
		if err := r.CacheSet(t.Context(), storage.CacheSetParams{
			Key:       "k1",
			Value:     []byte("not expired"),
			ExpiresAt: now.Add(time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
		if err := r.CacheSet(t.Context(), storage.CacheSetParams{
			Key:       "k2",
			Value:     []byte("expired"),
			ExpiresAt: now.Add(-time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
		if err := r.CacheSet(t.Context(), storage.CacheSetParams{
			Key:       "k3",
			Value:     []byte("no expireation date"),
			ExpiresAt: time.Time{},
		}); err != nil {
			t.Fatal(err)
		}
		if err := r.CacheSet(t.Context(), storage.CacheSetParams{
			Key:       "k4",
			Value:     []byte("expired"),
			ExpiresAt: now.Add(-time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
		// when
		n, err := r.CacheCleanUp(t.Context())
		// then
		require.NoError(t, err)
		xassert.Equal(t, 2, n)
		rows, err := db.Query("SELECT key FROM cache;")
		require.NoError(t, err)
		var keys []string
		for rows.Next() {
			var k string
			if err := rows.Scan(&k); err != nil {
				t.Fatal(err)
			}
			keys = append(keys, k)
		}
		assert.ElementsMatch(t, []string{"k1", "k3"}, keys)
	})
}
