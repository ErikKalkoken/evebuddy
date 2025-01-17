package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	db, r, _ := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can set and get entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
	t.Run("can update existing entry", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
	t.Run("should return true when entry exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
	t.Run("should report false when entry does not exist", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
	t.Run("should return false when entry expired", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
	t.Run("should treat expired entries as non existent for get", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})

	t.Run("can delete entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
	t.Run("can clear cache", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
}
