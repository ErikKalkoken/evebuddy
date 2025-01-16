// Package pcache provides a persistent cache.
package pcache

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

type PCache struct {
	st *storage.Storage
}

func New(st *storage.Storage) *PCache {
	pc := &PCache{st: st}
	return pc
}

func (pc *PCache) Clear() {
	err := pc.st.CacheClear(context.Background())
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
}

func (pc *PCache) Delete(key any) {
	err := pc.st.CacheDelete(context.Background(), key.(string))
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
}

func (pc *PCache) Exists(key any) bool {
	found, err := pc.st.CacheExists(context.Background(), key.(string))
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
	return found
}

func (pc *PCache) Get(key any) (any, bool) {
	v, err := pc.st.CacheGet(context.Background(), key.(string))
	if errors.Is(err, storage.ErrNotFound) {
		return nil, false
	}
	if err != nil {
		slog.Error("cache failure", "error", err)
		return nil, false
	}
	return v, true
}

func (pc *PCache) Set(key, value any, timeout time.Duration) {
	var expiresAt time.Time
	if timeout > 0 {
		expiresAt = time.Now().Add(timeout)
	}
	arg := storage.CacheSetParams{
		Key:       key.(string),
		Value:     value.([]byte),
		ExpiresAt: expiresAt,
	}
	err := pc.st.CacheSet(context.Background(), arg)
	if err != nil {
		slog.Error("cache failure", "error", err)
	}
}
