// Package pcache implements a persistent cache.
package pcache

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
)

// PCache is a persistent cache.
// It stores all items in the provided storage and also keeps a copy
// in a synced memory cache for faster retrieval.
type PCache struct {
	closeC chan struct{}
	mc     *memcache.Cache
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
		st:     st,
	}
	if cleanUpTimeout > 0 {
		go func() {
			for {
				select {
				case <-c.closeC:
					slog.Debug("cache closed")
					return
				case <-time.After(cleanUpTimeout):
				}
				c.CleanUp()
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
	if err := c.st.CacheClear(context.Background()); err != nil {
		slog.Error("cache failure", "error", err)
		return
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
	if err := c.st.CacheDelete(context.Background(), key); err != nil {
		slog.Error("cache failure", "error", err)
		return
	}
	c.mc.Delete(key)
}

// Exists reports whether an item exists. Expired items do not exist.
func (c *PCache) Exists(key string) bool {
	if c.mc.Exists(key) {
		return true
	}
	v, expiresAt, err := c.st.CacheGet(context.Background(), key)
	if errors.Is(err, app.ErrNotFound) {
		return false
	}
	if err != nil {
		slog.Error("cache failure", "error", err)
		return false
	}
	if d, ok := timeoutFromExpiresAt(expiresAt); ok {
		c.mc.Set(key, v, d)
	}
	return true
}

// Get returns an item that exists and is not expired.
// It also reports whether the item was found.
func (c *PCache) Get(key string) ([]byte, bool) {
	if x, found := c.mc.Get(key); found {
		return x.([]byte), true
	}
	v, expiresAt, err := c.st.CacheGet(context.Background(), key)
	if errors.Is(err, app.ErrNotFound) {
		return nil, false
	}
	if err != nil {
		slog.Error("Failed to fetch from pcache", "key", key, "error", err)
		return nil, false
	}
	if d, ok := timeoutFromExpiresAt(expiresAt); ok {
		c.mc.Set(key, v, d)
	}
	return v, true
}

// timeoutFromExpiresAt returns the memcache timeout for expiresAt
// and reports whether the item should be cached at all.
func timeoutFromExpiresAt(expiresAt time.Time) (time.Duration, bool) {
	if expiresAt.IsZero() {
		return 0, true
	}
	d := time.Until(expiresAt)
	return d, d > 0
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
	err := c.st.CacheSet(context.Background(), storage.CacheSetParams{
		Key:       key,
		Value:     value,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		slog.Error("Failed to store cache item", "error", err)
		return
	}
	c.mc.Set(key, value, timeout)
}
