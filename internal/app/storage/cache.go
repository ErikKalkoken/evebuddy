package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) CacheClear(ctx context.Context) error {
	err := st.q.CacheClear(ctx)
	if err != nil {
		return fmt.Errorf("cache clear: %w", err)
	}
	return nil
}

func (st *Storage) CacheCleanup(ctx context.Context) error {
	err := st.q.CacheCleanUp(ctx, NewNullTimeFromTime(time.Now().UTC()))
	if err != nil {
		return fmt.Errorf("cache cleanup: %w", err)
	}
	return nil
}

func (st *Storage) CacheExists(ctx context.Context, key string) (bool, error) {
	_, err := st.CacheGet(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache exists: %w", err)
	}
	return true, nil
}

func (st *Storage) CacheGet(ctx context.Context, key string) ([]byte, error) {
	arg := queries.CacheGetParams{
		Key: key,
		Now: NewNullTimeFromTime(time.Now().UTC()),
	}
	x, err := st.q.CacheGet(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("cache get: %w", err)
	}
	return x.Value, nil
}

func (st *Storage) CacheDelete(ctx context.Context, key string) error {
	err := st.q.CacheDelete(ctx, key)
	if err != nil {
		return fmt.Errorf("cache delete %s: %w", key, err)
	}
	return nil
}

type CacheSetParams struct {
	Key       string
	Value     []byte
	ExpiresAt time.Time
}

func (st *Storage) CacheSet(ctx context.Context, arg CacheSetParams) error {
	var expiresAt time.Time
	if !arg.ExpiresAt.IsZero() {
		expiresAt = arg.ExpiresAt.UTC()
	}
	arg2 := queries.CacheSetParams{
		ExpiresAt: NewNullTimeFromTime(expiresAt),
		Key:       arg.Key,
		Value:     arg.Value,
	}
	err := st.q.CacheSet(ctx, arg2)
	if err != nil {
		return fmt.Errorf("cache set %s: %w", arg.Key, err)
	}
	return nil
}
