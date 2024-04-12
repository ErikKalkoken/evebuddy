package model

type setting struct {
	Key   string
	Value []byte
}

func SetSetting(key string, data []byte) error {
	s := setting{Key: key, Value: data}
	_, err := db.NamedExec(`
		INSERT INTO settings (key, value)
		VALUES (:key, :value)
		ON CONFLICT (key) DO
		UPDATE SET value = :value;`,
		s,
	)
	return err
}

func GetSetting(key string) ([]byte, error) {
	var s setting
	err := db.Get(&s, "SELECT * FROM settings WHERE key = ?;", key)
	if err != nil {
		return nil, err
	}
	return s.Value, nil
}

func DeleteSetting(key string) error {
	_, err := db.Exec("DELETE FROM settings WHERE key = ?", key)
	return err
}
