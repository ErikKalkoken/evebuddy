package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"time"
)

// DictionaryDelete deletes a key from the dictionary.
// If the key does not exist no error will be raised.
func (s *Service) DictionaryDelete(key string) error {
	ctx := context.Background()
	return s.r.DeleteDictEntry(ctx, key)
}

// DictionaryExists reports wether a key exists in the dictionary.
func (s *Service) DictionaryExists(key string) (bool, error) {
	ctx := context.Background()
	_, err := s.r.GetDictEntry(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// DictionaryInt returns the int value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (s *Service) DictionaryInt(key string) (int, error) {
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

// DictionaryString returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (s *Service) DictionaryString(key string) (string, error) {
	ctx := context.Background()
	data, err := s.r.GetDictEntry(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return anyFromBytes[string](data)
}

// DictionaryTime returns the time value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (s *Service) DictionaryTime(key string) (time.Time, error) {
	ctx := context.Background()
	data, err := s.r.GetDictEntry(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	return anyFromBytes[time.Time](data)
}

// DictionarySetInt sets the value for a dictionary int entry.
func (s *Service) DictionarySetInt(key string, value int) error {
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

// DictionarySetString sets the value for a dictionary string entry.
func (s *Service) DictionarySetString(key string, value string) error {
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

// DictionarySetTime sets the value for a dictionary time entry.
func (s *Service) DictionarySetTime(key string, value time.Time) error {
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