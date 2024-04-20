package storage

import (
	"context"
	"fmt"

	"example/evebuddy/internal/storage/sqlc"

	"github.com/mattn/go-sqlite3"
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
	err := func() error {
		tx, err := r.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		qtx := r.q.WithTx(tx)
		arg := sqlc.CreateDictEntryParams{
			Value: bb,
			Key:   key,
		}
		if err := qtx.CreateDictEntry(ctx, arg); err != nil {
			sqlErr, ok := err.(sqlite3.Error)
			if !ok || sqlErr.ExtendedCode != sqlite3.ErrConstraintPrimaryKey {
				return err
			}
			arg := sqlc.UpdateDictEntryParams{
				Value: bb,
				Key:   key,
			}
			if err := qtx.UpdateDictEntry(ctx, arg); err != nil {
				return err
			}
		}
		return tx.Commit()
	}()
	if err != nil {
		return fmt.Errorf("failed to set setting for key %s: %w", key, err)
	}
	return nil
}
