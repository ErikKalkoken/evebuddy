package service

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"errors"

	"example/evebuddy/internal/model"
)

// GetSetting returns the value for a settings key, when it exists.
// Otherwise it returns it's zero value.
func GetSetting[T any](key string) (T, error) {
	var t T
	bb, err := model.GetSetting(key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return t, nil
		}
		return t, err
	}
	buf := bytes.NewBuffer(bb)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&t); err != nil {
		return t, err
	}
	return t, nil
}

// SetSetting sets the value for a settings key.
func SetSetting[T any](key string, value T) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		return err
	}
	err := model.SetSetting(key, buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func DeleteSetting(key string) error {
	return model.DeleteSetting(key)
}
