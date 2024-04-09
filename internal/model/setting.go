package model

import (
	"bytes"
	"database/sql"
	"encoding/gob"
)

type setting struct {
	Key   string
	Value []byte
}

// GetSetting returns the value for a settings key, when it exists.
// Otherwise it returns it's zero value.
func GetSetting[T any](key string) (T, error) {
	var t T
	var s setting
	err := db.Get(&s, "SELECT * FROM settings WHERE key = ?;", key)
	if err != nil {
		if err == sql.ErrNoRows {
			return t, nil
		}
		return t, err
	}
	buf := bytes.NewBuffer(s.Value)
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
	s := setting{Key: key, Value: buf.Bytes()}
	_, err := db.NamedExec(`
		INSERT INTO settings (key, value)
		VALUES (:key, :value)
		ON CONFLICT (key) DO
		UPDATE SET value = :value;`,
		s,
	)
	if err != nil {
		return err
	}
	return nil
}

func DeleteSetting(key string) error {
	_, err := db.Exec("DELETE FROM settings WHERE key = ?", key)
	return err
}
