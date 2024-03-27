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

// CacheDelete deletes the item for the given key. ErrCacheMiss is returned if the specified item can not be found.
func CacheDelete(key string) error {
	r, err := db.Exec("DELETE FROM cache_keys WHERE key=?", key)
	if err != nil {
		return err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrCacheMiss
	}
	return nil
}
