package storage

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"

	"example/evebuddy/internal/sqlc"
)

func (r *Storage) DeleteSetting(ctx context.Context, key string) error {
	err := r.q.DeleteSetting(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete setting for key %s: %w", key, err)
	}
	return nil
}

// GetSetting returns the value for a settings key, when it exists.
// Otherwise it returns it's zero value.
func (r *Storage) GetSettingInt32(ctx context.Context, key string) (int32, error) {
	obj, err := r.q.GetSetting(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get setting for key %s: %w", key, err)
	}
	return anyFromBytes[int32](obj.Value)
}

// GetSetting returns the value for a settings key, when it exists.
// Otherwise it returns it's zero value.
func (r *Storage) GetSettingInt(ctx context.Context, key string) (int, error) {
	obj, err := r.q.GetSetting(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get setting for key %s: %w", key, err)
	}
	return anyFromBytes[int](obj.Value)
}

// GetSetting returns the value for a settings key, when it exists.
// Otherwise it returns it's zero value.
func (r *Storage) GetSettingString(ctx context.Context, key string) (string, error) {
	obj, err := r.q.GetSetting(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get setting for key %s: %w", key, err)
	}
	return anyFromBytes[string](obj.Value)
}

// SetSetting sets the value for a settings key.
func (r *Storage) SetSettingInt32(ctx context.Context, key string, value int32) error {
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := r.setSetting(ctx, key, bb); err != nil {
		return err
	}
	return nil
}

// SetSetting sets the value for a settings key.
func (r *Storage) SetSettingInt(ctx context.Context, key string, value int) error {
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := r.setSetting(ctx, key, bb); err != nil {
		return err
	}
	return nil
}

// SetSetting sets the value for a settings key.
func (r *Storage) SetSettingString(ctx context.Context, key string, value string) error {
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := r.setSetting(ctx, key, bb); err != nil {
		return err
	}
	return nil
}

func (r *Storage) setSetting(ctx context.Context, key string, bb []byte) error {
	err := func() error {
		tx, err := r.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		qtx := r.q.WithTx(tx)
		arg := sqlc.CreateSettingParams{
			Value: bb,
			Key:   key,
		}
		if err := qtx.CreateSetting(ctx, arg); err != nil {
			if !isSqlite3ErrConstraint(err) {
				return err
			}
			arg := sqlc.UpdateSettingParams{
				Value: bb,
				Key:   key,
			}
			if err := qtx.UpdateSetting(ctx, arg); err != nil {
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

func anyFromBytes[T any](bb []byte) (T, error) {
	var t T
	buf := bytes.NewBuffer(bb)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&t); err != nil {
		return t, err
	}
	return t, nil
}

func bytesFromAny[T any](value T) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
