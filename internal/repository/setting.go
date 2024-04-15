package repository

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"

	"example/evebuddy/internal/repository/sqlc"
)

// GetSetting returns the value for a settings key, when it exists.
// Otherwise it returns it's zero value.
func (r *Repository) GetSettingInt32(key string) (int32, error) {
	obj, err := r.q.GetSetting(context.Background(), key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
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
		return "", err
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

func (s *Repository) setSetting(key string, bb []byte) error {
	ctx := context.Background()
	arg := sqlc.CreateSettingParams{
		Value: bb,
		Key:   key,
	}
	if err := s.q.CreateSetting(ctx, arg); err != nil {
		if !isSqlite3ErrConstraint(err) {
			return err
		}
		arg := sqlc.UpdateSettingParams{
			Value: bb,
			Key:   key,
		}
		if err := s.q.UpdateSetting(ctx, arg); err != nil {
			return err
		}
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

func (s *Repository) DeleteSetting(key string) error {
	return s.q.DeleteSetting(context.Background(), key)
}
