package repository

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"

	"example/evebuddy/internal/sqlc"
)

func (r *Repository) DeleteSetting(key string) error {
	err := r.q.DeleteSetting(context.Background(), key)
	if err != nil {
		return fmt.Errorf("failed to delete setting for key %s: %w", key, err)
	}
	return nil
}

// GetSetting returns the value for a settings key, when it exists.
// Otherwise it returns it's zero value.
func (r *Repository) GetSettingInt32(key string) (int32, error) {
	obj, err := r.q.GetSetting(context.Background(), key)
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
func (r *Repository) GetSettingString(key string) (string, error) {
	obj, err := r.q.GetSetting(context.Background(), key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get setting for key %s: %w", key, err)
	}
	return anyFromBytes[string](obj.Value)
}

// SetSetting sets the value for a settings key.
func (r *Repository) SetSettingInt32(key string, value int32) error {
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := r.setSetting(key, bb); err != nil {
		return err
	}
	return nil
}

// SetSetting sets the value for a settings key.
func (r *Repository) SetSettingString(key string, value string) error {
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := r.setSetting(key, bb); err != nil {
		return err
	}
	return nil
}

func (r *Repository) setSetting(key string, bb []byte) error {
	err := func() error {
		ctx := context.Background()
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
