package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
)

// DeleteDictKey deletes a key from the dictionary.
// If the key does not exist no error will be raised.
func (s *Service) DeleteDictKey(key string) error {
	ctx := context.Background()
	return s.r.DeleteDictEntry(ctx, key)
}

// GetDictKeyInt returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (s *Service) GetDictKeyInt(key string) (int, error) {
	ctx := context.Background()
	data, err := s.r.GetDictEntry(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return anyFromBytes[int](data)
}

// GetDictKeyString returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (s *Service) GetDictKeyString(key string) (string, error) {
	ctx := context.Background()
	data, err := s.r.GetDictEntry(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get setting for key %s: %w", key, err)
	}
	return anyFromBytes[string](data)
}

// SetDictKeyInt sets the value for a dictionary key.
func (s *Service) SetDictKeyInt(key string, value int) error {
	ctx := context.Background()
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := s.r.SetDictEntry(ctx, key, bb); err != nil {
		return err
	}
	return nil
}

// SetDictKeyString sets the value for a dictionary key.
func (s *Service) SetDictKeyString(key string, value string) error {
	ctx := context.Background()
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := s.r.SetDictEntry(ctx, key, bb); err != nil {
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
