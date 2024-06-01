package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

func (r *Storage) GetDictEntry(ctx context.Context, key string) ([]byte, bool, error) {
	obj, err := r.q.GetDictEntry(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("failed to get dict entry for key %s: %w", key, err)
	}
	return obj.Value, true, nil
}
func (r *Storage) DeleteDictEntry(ctx context.Context, key string) error {
	err := r.q.DeleteDictEntry(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete setting for key %s: %w", key, err)
	}
	return nil
}

func (r *Storage) SetDictEntry(ctx context.Context, key string, bb []byte) error {
	arg := queries.UpdateOrCreateDictEntryParams{
		Value: bb,
		Key:   key,
	}
	if err := r.q.UpdateOrCreateDictEntry(ctx, arg); err != nil {
		return fmt.Errorf("failed to set setting for key %s: %w", key, err)
	}
	return nil
}
