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
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// DictionaryInt returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (s *Service) DictionaryInt(key string) (int, bool, error) {
	ctx := context.Background()
	data, err := s.r.GetDictEntry(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	} else if err != nil {
		return 0, false, err
	}
	return anyFromBytes[int](data)
}

// DictionaryFloat32 returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (s *Service) DictionaryFloat32(key string) (float32, bool, error) {
	ctx := context.Background()
	data, err := s.r.GetDictEntry(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	} else if err != nil {
		return 0, false, err
	}
	return anyFromBytes[float32](data)
}

// DictionaryString returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (s *Service) DictionaryString(key string) (string, bool, error) {
	ctx := context.Background()
	data, err := s.r.GetDictEntry(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	} else if err != nil {
		return "", false, err
	}
	return anyFromBytes[string](data)
}

// DictionaryTime returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (s *Service) DictionaryTime(key string) (time.Time, bool, error) {
	ctx := context.Background()
	data, err := s.r.GetDictEntry(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	} else if err != nil {
		return time.Time{}, false, err
	}
	return anyFromBytes[time.Time](data)
}

func anyFromBytes[T any](bb []byte) (T, bool, error) {
	var t T
	buf := bytes.NewBuffer(bb)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&t); err != nil {
		return t, false, err
	}
	return t, true, nil
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

// DictionarySetFloat32 sets the value for a dictionary int entry.
func (s *Service) DictionarySetFloat32(key string, value float32) error {
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

func bytesFromAny[T any](value T) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
