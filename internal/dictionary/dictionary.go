// Package dictionary contains the dictionary service.
package dictionary

import (
	"bytes"
	"context"
	"encoding/gob"
	"time"
)

type DictionaryStorage interface {
	GetDictEntry(context.Context, string) ([]byte, bool, error)
	DeleteDictEntry(context.Context, string) error
	SetDictEntry(context.Context, string, []byte) error
}

// DictionaryService is persistent key/value store.
type DictionaryService struct {
	st DictionaryStorage
}

// New creates and returns a new instance of a dictionary service.
func New(st DictionaryStorage) *DictionaryService {
	d := &DictionaryService{st: st}
	return d
}

// Delete deletes a key from the dictionary.
// If the key does not exist no error will be raised.
func (d *DictionaryService) Delete(key string) error {
	ctx := context.TODO()
	return d.st.DeleteDictEntry(ctx, key)
}

// Exists reports wether a key exists in the dictionary.
func (d *DictionaryService) Exists(key string) (bool, error) {
	ctx := context.TODO()
	_, ok, err := d.st.GetDictEntry(ctx, key)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// Int returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (d *DictionaryService) Int(key string) (int, bool, error) {
	ctx := context.TODO()
	data, ok, err := d.st.GetDictEntry(ctx, key)
	if err != nil {
		return 0, false, err
	}
	if !ok {
		return 0, false, nil
	}
	return anyFromBytes[int](data)
}

func (d *DictionaryService) IntWithFallback(key string, fallback int) (int, error) {
	v, found, err := d.Int(key)
	if err != nil {
		return 0, err
	}
	if !found {
		return fallback, nil
	}
	return v, nil
}

// Float32 returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (d *DictionaryService) Float32(key string) (float32, bool, error) {
	ctx := context.TODO()
	data, ok, err := d.st.GetDictEntry(ctx, key)
	if err != nil {
		return 0, false, err
	}
	if !ok {
		return 0, false, nil
	}
	return anyFromBytes[float32](data)
}

// Float64 returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (d *DictionaryService) Float64(key string) (float64, bool, error) {
	ctx := context.TODO()
	data, ok, err := d.st.GetDictEntry(ctx, key)
	if err != nil {
		return 0, false, err
	}
	if !ok {
		return 0, false, nil
	}
	return anyFromBytes[float64](data)
}

// String returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (d *DictionaryService) String(key string) (string, bool, error) {
	ctx := context.TODO()
	data, ok, err := d.st.GetDictEntry(ctx, key)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}
	return anyFromBytes[string](data)
}

// Time returns the value for a dictionary key, when it exists.
// Otherwise it returns it's zero value.
func (d *DictionaryService) Time(key string) (time.Time, bool, error) {
	ctx := context.TODO()
	data, ok, err := d.st.GetDictEntry(ctx, key)
	if err != nil {
		return time.Time{}, false, err
	}
	if !ok {
		return time.Time{}, false, nil
	}
	return anyFromBytes[time.Time](data)
}

// SetInt sets the value for a dictionary int entry.
func (d *DictionaryService) SetInt(key string, value int) error {
	ctx := context.TODO()
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := d.st.SetDictEntry(ctx, key, bb); err != nil {
		return err
	}
	return nil
}

// SetFloat32 sets the value for a dictionary int entry.
func (d *DictionaryService) SetFloat32(key string, value float32) error {
	ctx := context.TODO()
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := d.st.SetDictEntry(ctx, key, bb); err != nil {
		return err
	}
	return nil
}

// SetFloat64 sets the value for a dictionary int entry.
func (d *DictionaryService) SetFloat64(key string, value float64) error {
	ctx := context.TODO()
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := d.st.SetDictEntry(ctx, key, bb); err != nil {
		return err
	}
	return nil
}

// SetString sets the value for a dictionary string entry.
func (d *DictionaryService) SetString(key string, value string) error {
	ctx := context.TODO()
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := d.st.SetDictEntry(ctx, key, bb); err != nil {
		return err
	}
	return nil
}

// SetTime sets the value for a dictionary time entry.
func (d *DictionaryService) SetTime(key string, value time.Time) error {
	ctx := context.TODO()
	bb, err := bytesFromAny(value)
	if err != nil {
		return err
	}
	if err := d.st.SetDictEntry(ctx, key, bb); err != nil {
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
