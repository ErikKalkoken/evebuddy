package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"

	"example/evebuddy/internal/repository"
)

// GetSetting returns the value for a settings key, when it exists.
// Otherwise it returns it's zero value.
func (s *Service) GetSettingInt32(key string) (int32, error) {
	obj, err := s.queries.GetSetting(context.Background(), key)
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
func (s *Service) GetSettingString(key string) (string, error) {
	obj, err := s.queries.GetSetting(context.Background(), key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return anyFromBytes[string](obj.Value)
}

// SetSetting sets the value for a settings key.
func (s *Service) SetSettingInt32(key string, value int32) error {
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	arg := repository.UpdateOrCreateSettingParams{
		Value: bb,
		Key:   key,
	}
	if err := s.queries.UpdateOrCreateSetting(context.Background(), arg); err != nil {
		return err
	}
	return nil
}

// SetSetting sets the value for a settings key.
func (s *Service) SetSettingString(key string, value string) error {
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	arg := repository.UpdateOrCreateSettingParams{
		Value: bb,
		Key:   key,
	}
	if err := s.queries.UpdateOrCreateSetting(context.Background(), arg); err != nil {
		return err
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

func (s *Service) DeleteSetting(key string) error {
	return s.queries.DeleteSetting(context.Background(), key)
}
