// Package pcache implements a persistent cache.
package pcache

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/cache"
	"golang.org/x/sync/singleflight"
)

// PCache is a 2-level persistent cache.
// 1st level is in memory. 2nd level is on a persistent storage.
type PCache struct {
	closeC chan struct{}
	mc     *cache.Cache
	sfg    *singleflight.Group
	st     *storage.Storage
}

// New returns a new PCache.
//
// cleanUpTimeout is the timeout between automatic clean-up intervals.
// When set to 0 automatic clean-up is disabled and cache users need to start clean-ups manually.
//
// When automatic clean-up is enabled users can close the cache
// to free allocated resources when the cache is no longer needed.
func New(st *storage.Storage, cleanUpTimeout time.Duration) *PCache {
	c := &PCache{
		closeC: make(chan struct{}),
		mc:     cache.NewWithTimeout(0),
		sfg:    new(singleflight.Group),
		st:     st,
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
	c.mc.CleanUp()
}

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
}

func (c *PCache) Delete(key string) {
	err := c.st.CacheDelete(context.Background(), key)
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
	c.mc.Delete(key)
}

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

func (c *PCache) Get(key string) ([]byte, bool) {
	x, found := c.mc.Get(key)
	if found {
		return x.([]byte), true
	}
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
	c.mc.Set(key, value, timeout)
	_, err, _ := c.sfg.Do(key, func() (interface{}, error) {
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
