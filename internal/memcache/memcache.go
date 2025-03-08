// Package cache implements a simple in-memory cache.
package memcache

import (
	"log/slog"
	"sync"
	"time"
)

const (
	cleanUpTimeOutDefault = time.Minute * 10
)

// An in-memory cache.
type Cache struct {
	items  sync.Map
	closeC chan struct{}
}

type item struct {
	Value     any
	ExpiresAt time.Time
}

// New creates a new cache with default timeout and returns it.
//
// Users can close the cache to free allocated resources when the cache is no longer needed.
func New() *Cache {
	return create(cleanUpTimeOutDefault)
}

// NewWithTimeout creates a new cache with a specific timeout for the regular clean-up and returns it.
//
// A timeout of 0 disables the automatic clean-up and users then need to start clean-up manually.
//
// When automatic clean-up is enabled users can close the cache
// to free allocated resources when the cache is no longer needed.
func NewWithTimeout(cleanUpTimeout time.Duration) *Cache {
	return create(cleanUpTimeout)
}

func create(cleanUpTimeout time.Duration) *Cache {
	c := &Cache{
		closeC: make(chan struct{}),
	}
	if cleanUpTimeout > 0 {
		go func() {
			for {
				select {
				case <-c.closeC:
					slog.Info("cache closed")
					return
				case <-time.After(cleanUpTimeout):
				}
				c.CleanUp()
			}
		}()
	}
	return c
}

// CleanUp removes all expired items.
func (c *Cache) CleanUp() {
	slog.Info("cache clean-up: started")
	n := 0
	c.items.Range(func(key, value any) bool {
		_, found := c.Get(key.(string))
		if !found {
			c.Delete(key.(string))
			n++
		}
		return true
	})
	slog.Info("cache clean-up: completed", "removed", n)
}

// Clear removes all items.
func (c *Cache) Clear() {
	c.items.Range(func(key, value any) bool {
		c.items.Delete(key)
		return true
	})
}

// Close closes the cache and frees allocated resources.
func (c *Cache) Close() {
	close(c.closeC)
}

// Delete deletes an item.
func (c *Cache) Delete(key string) {
	c.items.Delete(key)
}

// Exists reports wether an item exists. Expired items do not exist.
func (c *Cache) Exists(key string) bool {
	_, ok := c.Get(key)
	return ok
}

// Get returns an item that exists and is not expired.
// It also reports whether the item was found.
func (c *Cache) Get(key string) (any, bool) {
	value, ok := c.items.Load(key)
	if !ok {
		return nil, false
	}
	i := value.(item)
	if !i.ExpiresAt.IsZero() && time.Until(i.ExpiresAt) < 0 {
		return nil, false
	}
	return i.Value, ok
}

// Set stores an item in the cache.
//
// If an item with the same key already exists it will be overwritten.
// An item with timeout = 0 never expires
func (c *Cache) Set(key string, value any, timeout time.Duration) {
	var at time.Time
	if timeout > 0 {
		at = time.Now().Add(timeout)
	}
	i := item{Value: value, ExpiresAt: at}
	c.items.Store(key, i)
}
