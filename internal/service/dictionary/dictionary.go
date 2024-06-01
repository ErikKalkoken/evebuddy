package dictionary

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

type Dictionary struct {
	r *storage.Storage
}

func New(r *storage.Storage) *Dictionary {
	d := &Dictionary{r: r}
	return d
}

// DictionaryDelete deletes a key from the dictionary.
// If the key does not exist no error will be raised.
func (s *Dictionary) DictionaryDelete(key string) error {
	ctx := context.Background()
	return s.r.DeleteDictEntry(ctx, key)
}

// DictionaryExists reports wether a key exists in the dictionary.
func (s *Dictionary) DictionaryExists(key string) (bool, error) {
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
func (s *Dictionary) DictionaryInt(key string) (int, bool, error) {
	ctx := context.Background()
	data, err := s.r.GetDictEntry(ctx, key)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	} else if err != nil {
		return 0, false, err
	}
	return anyFromBytes[int](data)
}

func (s *Dictionary) DictionaryIntWithFallback(key string, fallback int) (int, error) {
	v, found, err := s.DictionaryInt(key)
	if err != nil {
		return 0, err
	}
	if !found {
		return fallback, nil
	}
	return v, nil
}

// DictionaryFloat32 returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (s *Dictionary) DictionaryFloat32(key string) (float32, bool, error) {
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
func (s *Dictionary) DictionaryString(key string) (string, bool, error) {
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
func (s *Dictionary) DictionaryTime(key string) (time.Time, bool, error) {
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
func (s *Dictionary) DictionarySetInt(key string, value int) error {
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
func (s *Dictionary) DictionarySetFloat32(key string, value float32) error {
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
func (s *Dictionary) DictionarySetString(key string, value string) error {
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
func (s *Dictionary) DictionarySetTime(key string, value time.Time) error {
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
