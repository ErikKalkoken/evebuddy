// Package pcache implements a persistent cache.
package pcache

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
)

// PCache is a persistent cache.
// It stores all items in the provided storage and also keeps a copy
// in a synced memory cache for faster retrieval.
//
// After Close all operations become no-ops, so late calls during shutdown
// do not hit a closed database.
type PCache struct {
	cancel context.CancelFunc
	closed atomic.Bool
	ctx    context.Context
	mc     *memcache.Cache
	st     *storage.Storage
	wg     sync.WaitGroup // cleanup goroutine
}

// New returns a new PCache.
//
// cleanUpTimeout is the timeout between automatic clean-up intervals.
// When set to 0 automatic clean-up is disabled and users need to start clean-ups manually.
//
// Users should close the cache when it is no longer needed,
// e.g. before closing the database.
func New(st *storage.Storage, cleanUpTimeout time.Duration) *PCache {
	ctx, cancel := context.WithCancel(context.Background())
	c := &PCache{
		cancel: cancel,
		ctx:    ctx,
		mc:     memcache.NewWithTimeout(0),
		st:     st,
	}
	if cleanUpTimeout > 0 {
		c.wg.Go(func() {
			ticker := time.NewTicker(cleanUpTimeout)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					slog.Debug("cache closed")
					return
				case <-ticker.C:
				}
				c.CleanUp()
			}
		})
	}
	return c
}

// CleanUp removes all expired items.
func (c *PCache) CleanUp() int {
	if c.closed.Load() {
		return 0
	}
	slog.Debug("pcache clean-up: started")
	n, err := c.st.CacheCleanUp(c.ctx)
	if err != nil {
		c.logError("clean-up", "", err)
		n = -1
	}
	c.mc.CleanUp()
	slog.Debug("pcache clean-up: completed", "removed", n)
	return n
}

// Clear removes all items.
func (c *PCache) Clear() {
	if c.closed.Load() {
		return
	}
	if err := c.st.CacheClear(c.ctx); err != nil {
		c.logError("clear", "", err)
		return
	}
	c.mc.Clear()
}

// Close closes the cache and frees allocated resources.
// It cancels in-flight operations and waits for a running clean-up to finish.
// Close is idempotent.
func (c *PCache) Close() {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}
	c.cancel()
	c.wg.Wait()
	c.mc.Close()
}

// Delete deletes an item.
func (c *PCache) Delete(key string) {
	if c.closed.Load() {
		return
	}
	if err := c.st.CacheDelete(c.ctx, key); err != nil {
		c.logError("delete", key, err)
		return
	}
	c.mc.Delete(key)
}

// Exists reports whether an item exists. Expired items do not exist.
func (c *PCache) Exists(key string) bool {
	if c.closed.Load() {
		return false
	}
	if c.mc.Exists(key) {
		return true
	}
	v, expiresAt, err := c.st.CacheGet(c.ctx, key)
	if errors.Is(err, app.ErrNotFound) {
		return false
	}
	if err != nil {
		c.logError("exists", key, err)
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
	if c.closed.Load() {
		return nil, false
	}
	if x, found := c.mc.Get(key); found {
		return x.([]byte), true
	}
	v, expiresAt, err := c.st.CacheGet(c.ctx, key)
	if errors.Is(err, app.ErrNotFound) {
		return nil, false
	}
	if err != nil {
		c.logError("get", key, err)
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
	if c.closed.Load() {
		return
	}
	var expiresAt time.Time
	if timeout > 0 {
		expiresAt = time.Now().Add(timeout)
	}
	err := c.st.CacheSet(c.ctx, storage.CacheSetParams{
		Key:       key,
		Value:     value,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		c.logError("set", key, err)
		return
	}
	c.mc.Set(key, value, timeout)
}

// logError logs failures, but only at debug level when caused by shutdown.
func (c *PCache) logError(op, key string, err error) {
	if c.closed.Load() {
		slog.Debug("pcache failure after close", "op", op, "key", key, "error", err)
		return
	}
	slog.Error("pcache failure", "op", op, "key", key, "error", err)
}
