// Package pcache implements a persistent cache.
package pcache

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
)

// PCache is a persistent cache.
// It stores all items in the provided storage and also keeps a copy in a synced memory cache for faster retrieval.
type PCache struct {
	closeC chan struct{}
	mc     *memcache.Cache
	sfg    *singleflight.Group
	st     *storage.Storage
}

// New returns a new PCache.
//
// cleanUpTimeout is the timeout between automatic clean-up intervals.
// When set to 0 automatic clean-up is disabled and users need to start clean-ups manually.
//
// When automatic clean-up is enabled users can close the cache
// to free allocated resources when the cache is no longer needed.
func New(st *storage.Storage, cleanUpTimeout time.Duration) *PCache {
	c := &PCache{
		closeC: make(chan struct{}),
		mc:     memcache.NewWithTimeout(0),
		sfg:    new(singleflight.Group),
		st:     st,
	}
	if cleanUpTimeout > 0 {
		go func() {
			for {
				c.CleanUp()
				select {
				case <-c.closeC:
					slog.Debug("cache closed")
					return
				case <-time.After(cleanUpTimeout):
				}
			}
		}()
	}
	return c
}

// CleanUp removes all expired items.
func (c *PCache) CleanUp() int {
	slog.Debug("pcache clean-up: started")
	n, err := c.st.CacheCleanUp(context.Background())
	if err != nil {
		slog.Error("cache failure", "error", err)
		n = -1
	}
	c.mc.CleanUp()
	slog.Debug("pcache clean-up: completed", "removed", n)
	return n
}

// Clear removes all items.
func (c *PCache) Clear() {
	err := c.st.CacheClear(context.Background())
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
	c.mc.Clear()
}

// Close closes the cache and frees allocated resources.
func (c *PCache) Close() {
	close(c.closeC)
	c.mc.Close()
}

// Delete deletes an item.
func (c *PCache) Delete(key string) {
	err := c.st.CacheDelete(context.Background(), key)
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
	c.mc.Delete(key)
}

// Exists reports whether an item exists. Expired items do not exist.
func (c *PCache) Exists(key string) bool {
	if c.mc.Exists(key) {
		return true
	}
	found, err := c.st.CacheExists(context.Background(), key)
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
	return found
}

// Get returns an item that exists and is not expired.
// It also reports whether the item was found.
func (c *PCache) Get(key string) ([]byte, bool) {
	x, found := c.mc.Get(key)
	if found {
		return x.([]byte), true
	}
	v, err := c.st.CacheGet(context.Background(), key)
	if errors.Is(err, app.ErrNotFound) {
		return nil, false
	}
	if err != nil {
		slog.Error("cache failure", "error", err)
		return nil, false
	}
	return v, true
}

// Set stores an item in the cache.
//
// If an item with the same key already exists it will be overwritten.
// An item with timeout = 0 never expires
func (c *PCache) Set(key string, value []byte, timeout time.Duration) {
	var expiresAt time.Time
	if timeout > 0 {
		expiresAt = time.Now().Add(timeout)
	}
	c.mc.Set(key, value, timeout)
	_, err, _ := c.sfg.Do(key, func() (any, error) {
		arg := storage.CacheSetParams{
			Key:       key,
			Value:     value,
			ExpiresAt: expiresAt,
		}
		err := c.st.CacheSet(context.Background(), arg)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		slog.Error("store cache item", "error", err)
	}
}
