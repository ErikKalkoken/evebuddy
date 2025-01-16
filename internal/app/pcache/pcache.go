// Package pcache implements a persistent cache.
package pcache

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

// PCache is a persistent cache. It can automatically remove expired items.
type PCache struct {
	st     *storage.Storage
	closeC chan struct{}
}

// New returns a new PCache.
//
// cleanUpTimeout is the timeout between automatic clean-up intervals. When set to 0 no cleanUp will be done.
// Make sure to close this object again to free all it's resources.
func New(st *storage.Storage, cleanUpTimeout time.Duration) *PCache {
	c := &PCache{
		st:     st,
		closeC: make(chan struct{}),
	}
	if cleanUpTimeout > 0 {
		ticker := time.NewTicker(cleanUpTimeout)
		go func() {
			for {
				select {
				case <-c.closeC:
					slog.Info("cache closed")
					return
				case <-ticker.C:
					c.CleanUp()
				}
			}
		}()
	}
	return c
}

func (c *PCache) CleanUp() {
	err := c.st.CacheCleanUp(context.Background())
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
}

func (c *PCache) Clear() {
	err := c.st.CacheClear(context.Background())
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
}

// Close closes the cache and frees allocated resources.
func (c *PCache) Close() {
	close(c.closeC)
}

func (c *PCache) Delete(key string) {
	err := c.st.CacheDelete(context.Background(), key)
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
}

func (c *PCache) Exists(key string) bool {
	found, err := c.st.CacheExists(context.Background(), key)
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
	return found
}

func (c *PCache) Get(key string) ([]byte, bool) {
	v, err := c.st.CacheGet(context.Background(), key)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, false
	}
	if err != nil {
		slog.Error("cache failure", "error", err)
		return nil, false
	}
	return v, true
}

func (c *PCache) Set(key string, value []byte, timeout time.Duration) {
	var expiresAt time.Time
	if timeout > 0 {
		expiresAt = time.Now().Add(timeout)
	}
	arg := storage.CacheSetParams{
		Key:       key,
		Value:     value,
		ExpiresAt: expiresAt,
	}
	err := c.st.CacheSet(context.Background(), arg)
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
}
