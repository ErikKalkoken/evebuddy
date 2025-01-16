// Package pcache implements a persistent cache.
package pcache

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

type PCache struct {
	st     *storage.Storage
	closeC chan struct{}
}

func New(st *storage.Storage, cleanUpTimeout time.Duration) *PCache {
	c := &PCache{
		st:     st,
		closeC: make(chan struct{}),
	}
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

func (c *PCache) Delete(key any) {
	err := c.st.CacheDelete(context.Background(), key.(string))
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
}

func (c *PCache) Exists(key any) bool {
	found, err := c.st.CacheExists(context.Background(), key.(string))
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
	return found
}

func (c *PCache) Get(key any) (any, bool) {
	v, err := c.st.CacheGet(context.Background(), key.(string))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, false
	}
	if err != nil {
		slog.Error("cache failure", "error", err)
		return nil, false
	}
	return v, true
}

func (c *PCache) Set(key, value any, timeout time.Duration) {
	var expiresAt time.Time
	if timeout > 0 {
		expiresAt = time.Now().Add(timeout)
	}
	arg := storage.CacheSetParams{
		Key:       key.(string),
		Value:     value.([]byte),
		ExpiresAt: expiresAt,
	}
	err := c.st.CacheSet(context.Background(), arg)
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
}
