package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/queries"
)

func (st *Storage) GetDictEntry(ctx context.Context, key string) ([]byte, bool, error) {
	obj, err := st.q.GetDictEntry(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("failed to get dict entry for key %s: %w", key, err)
	}
	return obj.Value, true, nil
}
func (st *Storage) DeleteDictEntry(ctx context.Context, key string) error {
	err := st.q.DeleteDictEntry(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete setting for key %s: %w", key, err)
	}
	return nil
}

func (st *Storage) SetDictEntry(ctx context.Context, key string, bb []byte) error {
	arg := queries.UpdateOrCreateDictEntryParams{
		Value: bb,
		Key:   key,
	}
	if err := st.q.UpdateOrCreateDictEntry(ctx, arg); err != nil {
		return fmt.Errorf("failed to set setting for key %s: %w", key, err)
	}
	return nil
}
