package dictionary

import (
	"bytes"
	"context"
	"encoding/gob"
	"time"
)

type Storage interface {
	GetDictEntry(context.Context, string) ([]byte, bool, error)
	DeleteDictEntry(context.Context, string) error
	SetDictEntry(context.Context, string, []byte) error
}

type Dictionary struct {
	s Storage
}

func New(s Storage) *Dictionary {
	d := &Dictionary{s: s}
	return d
}

// Delete deletes a key from the dictionary.
// If the key does not exist no error will be raised.
func (d *Dictionary) Delete(key string) error {
	ctx := context.Background()
	return d.s.DeleteDictEntry(ctx, key)
}

// Exists reports wether a key exists in the dictionary.
func (d *Dictionary) Exists(key string) (bool, error) {
	ctx := context.Background()
	_, ok, err := d.s.GetDictEntry(ctx, key)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// GetInt returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (d *Dictionary) GetInt(key string) (int, bool, error) {
	ctx := context.Background()
	data, ok, err := d.s.GetDictEntry(ctx, key)
	if err != nil {
		return 0, false, err
	}
	if !ok {
		return 0, false, nil
	}
	return anyFromBytes[int](data)
}

func (d *Dictionary) GetIntWithFallback(key string, fallback int) (int, error) {
	v, found, err := d.GetInt(key)
	if err != nil {
		return 0, err
	}
	if !found {
		return fallback, nil
	}
	return v, nil
}

// GetFloat32 returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (d *Dictionary) GetFloat32(key string) (float32, bool, error) {
	ctx := context.Background()
	data, ok, err := d.s.GetDictEntry(ctx, key)
	if err != nil {
		return 0, false, err
	}
	if !ok {
		return 0, false, nil
	}
	return anyFromBytes[float32](data)
}

// GetString returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (d *Dictionary) GetString(key string) (string, bool, error) {
	ctx := context.Background()
	data, ok, err := d.s.GetDictEntry(ctx, key)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	return anyFromBytes[string](data)
}

// GetTime returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (d *Dictionary) GetTime(key string) (time.Time, bool, error) {
	ctx := context.Background()
	data, ok, err := d.s.GetDictEntry(ctx, key)
	if err != nil {
		return time.Time{}, false, err
	}
	if !ok {
		return time.Time{}, false, nil
	}
	return anyFromBytes[time.Time](data)
}

// SetInt sets the value for a dictionary int entry.
func (d *Dictionary) SetInt(key string, value int) error {
	ctx := context.Background()
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := d.s.SetDictEntry(ctx, key, bb); err != nil {
		return err
	}
	return nil
}

// SetFloat32 sets the value for a dictionary int entry.
func (d *Dictionary) SetFloat32(key string, value float32) error {
	ctx := context.Background()
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := d.s.SetDictEntry(ctx, key, bb); err != nil {
		return err
	}
	return nil
}

// SetString sets the value for a dictionary string entry.
func (d *Dictionary) SetString(key string, value string) error {
	ctx := context.Background()
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := d.s.SetDictEntry(ctx, key, bb); err != nil {
		return err
	}
	return nil
}

// SetTime sets the value for a dictionary time entry.
func (d *Dictionary) SetTime(key string, value time.Time) error {
	ctx := context.Background()
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := d.s.SetDictEntry(ctx, key, bb); err != nil {
		return err
	}
	return nil
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

func bytesFromAny[T any](value T) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
