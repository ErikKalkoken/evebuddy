package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCacheGet(t *testing.T) {
	db, r, _ := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can get and existing entry", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		expiresAt := time.Now().Add(time.Minute)
		// when
		err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})
		// then
		if assert.NoError(t, err) {
			v, err := r.CacheGet(ctx, key)
			if assert.NoError(t, err) {
				assert.Equal(t, value, v)
			}
		}
	})
	t.Run("should treat expired entries as non existent for get", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		expiresAt := time.Now().Add(-time.Minute)
		err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			t.Fatal(err)
		}
		// when
		_, err = r.CacheGet(ctx, key)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return entries with get which never expiry", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: time.Time{},
		})
		if err != nil {
			t.Fatal(err)
		}
		// when
		x, err := r.CacheGet(ctx, key)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, value, x)
		}
	})
	t.Run("should return error when key is empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := r.CacheGet(ctx, "")
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}

func TestCacheExists(t *testing.T) {
	db, r, _ := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return true when entry exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		expiresAt := time.Now().Add(time.Minute)
		err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			t.Fatal(err)
		}
		// when
		ok, err := r.CacheExists(ctx, key)
		if assert.NoError(t, err) {
			assert.True(t, ok)
		}
	})
	t.Run("should return false when entry expired", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		expiresAt := time.Now().Add(-time.Minute)
		err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			t.Fatal(err)
		}
		// when
		ok, err := r.CacheExists(ctx, key)
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})
	t.Run("should report false when entry does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		expiresAt := time.Now().Add(time.Minute)
		err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			t.Fatal(err)
		}
		// when
		ok, err := r.CacheExists(ctx, "key-2")
		if assert.NoError(t, err) {
			assert.False(t, ok)
		}
	})
	t.Run("should return true when entry has no expiration date", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: time.Time{},
		})
		if err != nil {
			t.Fatal(err)
		}
		// when
		ok, err := r.CacheExists(ctx, key)
		if assert.NoError(t, err) {
			assert.True(t, ok)
		}
	})
	t.Run("should return error when key is empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := r.CacheExists(ctx, "")
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}

func TestCacheSet(t *testing.T) {
	db, r, _ := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can update existing entry", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       key,
			Value:     []byte("old-value"),
			ExpiresAt: time.Now().Add(3 * time.Minute),
		})
		if err != nil {
			t.Fatal(err)
		}
		value := []byte("value")
		expiresAt := time.Now().Add(5 * time.Minute)
		// when
		err = r.CacheSet(ctx, storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})
		// then
		if assert.NoError(t, err) {
			v, err := r.CacheGet(ctx, key)
			if assert.NoError(t, err) {
				assert.Equal(t, value, v)
			}
		}
	})
	t.Run("should return error when key is empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := r.CacheSet(ctx, storage.CacheSetParams{})
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}

func TestCacheOther(t *testing.T) {
	db, r, _ := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can delete entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		expiresAt := time.Now().Add(time.Minute)
		err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			t.Fatal(err)
		}
		// when
		err = r.CacheDelete(ctx, key)
		// then
		if assert.NoError(t, err) {
			ok, err := r.CacheExists(ctx, key)
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		}
	})
	t.Run("should return error when delete and key is empty", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := r.CacheDelete(ctx, "")
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
	t.Run("can clear cache", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		key := "key"
		value := []byte("value")
		expiresAt := time.Now().Add(time.Minute)
		err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		})
		if err != nil {
			t.Fatal(err)
		}
		// when
		err = r.CacheClear(ctx)
		// then
		if assert.NoError(t, err) {
			ok, err := r.CacheExists(ctx, key)
			if assert.NoError(t, err) {
				assert.False(t, ok)
			}
		}
	})
	t.Run("can remove all expired entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		now := time.Now()
		if err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       "k1",
			Value:     []byte("not expired"),
			ExpiresAt: now.Add(time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
		if err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       "k2",
			Value:     []byte("expired"),
			ExpiresAt: now.Add(-time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
		if err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       "k3",
			Value:     []byte("no expireation date"),
			ExpiresAt: time.Time{},
		}); err != nil {
			t.Fatal(err)
		}
		if err := r.CacheSet(ctx, storage.CacheSetParams{
			Key:       "k4",
			Value:     []byte("expired"),
			ExpiresAt: now.Add(-time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
		// when
		n, err := r.CacheCleanUp(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 2, n)
			rows, err := db.Query("SELECT key FROM cache;")
			if err != nil {
				t.Fatal(err)
			}
			var keys []string
			for rows.Next() {
				var k string
				if err := rows.Scan(&k); err != nil {
					t.Fatal(err)
				}
				keys = append(keys, k)
			}
			assert.ElementsMatch(t, []string{"k1", "k3"}, keys)
		}
	})
}
