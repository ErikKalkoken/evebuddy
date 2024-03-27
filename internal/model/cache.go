package model

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type CacheEntry struct {
	Key       string
	Value     []byte
	ExpiresAt time.Time `db:"expires_at"`
}

var ErrCacheMiss = errors.New("cache miss")

func CacheGet(key string) ([]byte, error) {
	var v CacheEntry
	err := db.Get(&v, "SELECT * FROM cache_keys WHERE key = ? AND expires_at > ?;", key, time.Now())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrCacheMiss
		}
		return nil, err
	}
	return v.Value, nil
}

func CacheSet(key string, value []byte, timeout int) error {
	if timeout < 0 {
		return fmt.Errorf("timeout invalid: %d", timeout)
	}
	e := time.Now().Add(time.Second * time.Duration(timeout))
	c := CacheEntry{Key: key, Value: value, ExpiresAt: e}
	_, err := db.NamedExec(`
		INSERT INTO cache_keys (key, value, expires_at)
		VALUES (:key, :value, :expires_at)
		ON CONFLICT (key) DO
		UPDATE SET value=:value, expires_at=:expires_at;`,
		c,
	)
	if err != nil {
		return err
	}
	return nil
}
