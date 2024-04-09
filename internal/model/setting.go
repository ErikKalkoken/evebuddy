package model

import "database/sql"

type Setting struct {
	Key   string
	Value string
}

func GetSetting(key string) (string, error) {
	var s Setting
	err := db.Get(&s, "SELECT * FROM settings WHERE key = ?;", key)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return s.Value, nil

}

func SetSetting(key, value string) error {
	s := Setting{Key: key, Value: value}
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
