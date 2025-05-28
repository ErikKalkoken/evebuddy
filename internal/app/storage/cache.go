package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) CacheClear(ctx context.Context) error {
	err := st.qRW.CacheClear(ctx)
	if err != nil {
		return fmt.Errorf("cache clear: %w", err)
	}
	return nil
}

func (st *Storage) CacheCleanUp(ctx context.Context) (int, error) {
	n, err := st.qRW.CacheCleanUp(ctx, NewNullTimeFromTime(time.Now().UTC()))
	if err != nil {
		return 0, fmt.Errorf("cache cleanup: %w", err)
	}
	return int(n), nil
}

func (st *Storage) CacheExists(ctx context.Context, key string) (bool, error) {
	_, err := st.CacheGet(ctx, key)
	if errors.Is(err, app.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cache exists: %w", err)
	}
	return true, nil
}

func (st *Storage) CacheGet(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("CacheGet: key can not be empty: %w", app.ErrInvalid)
	}
	arg := queries.CacheGetParams{
		Key: key,
		Now: NewNullTimeFromTime(time.Now().UTC()),
	}
	x, err := st.qRO.CacheGet(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("cache get: %w", convertGetError(err))
	}
	return x.Value, nil
}

func (st *Storage) CacheDelete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("CacheDelete: key can not be empty: %w", app.ErrInvalid)
	}
	err := st.qRW.CacheDelete(ctx, key)
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
	if arg.Key == "" {
		return fmt.Errorf("CacheSet: key can not be empty: %w", app.ErrInvalid)
	}
	var expiresAt time.Time
	if !arg.ExpiresAt.IsZero() {
		expiresAt = arg.ExpiresAt.UTC()
	}
	arg2 := queries.CacheSetParams{
		ExpiresAt: NewNullTimeFromTime(expiresAt),
		Key:       arg.Key,
		Value:     arg.Value,
	}
	err := st.qRW.CacheSet(ctx, arg2)
	if err != nil {
		return fmt.Errorf("cache set %s: %w", arg.Key, err)
	}
	return nil
}
