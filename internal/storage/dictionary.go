package storage

import (
	"context"
	"fmt"

	"example/evebuddy/internal/storage/queries"
)

func (r *Storage) GetDictEntry(ctx context.Context, key string) ([]byte, error) {
	obj, err := r.q.GetDictEntry(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get dict entry for key %s: %w", key, err)
	}
	return obj.Value, nil
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
