// Package cache implements a simple in-memory cache.
package cache

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

// New creates a new cache and returns it.
func New() *Cache {
	return create(cleanUpTimeOutDefault)
}

// NewWithTimeout creates a new cache with a specific timeout for the regular clean up and returns it.
func NewWithTimeout(timeout time.Duration) *Cache {
	if timeout == 0 {
		panic("timeout can not be zero")
	}
	return create(timeout)
}

func create(timeout time.Duration) *Cache {
	c := Cache{
		closeC: make(chan struct{}),
	}
	ticker := time.NewTicker(timeout)
	go func() {
		for {
			select {
			case <-c.closeC:
				slog.Info("cache closed")
				return
			case <-ticker.C:
				c.cleanUp()
			}
		}
	}()
	return &c
}

// Close closes the cache and frees allocated resources.
func (c *Cache) Close() {
	close(c.closeC)
}

// Clear removes all cache keys.
func (c *Cache) Clear() {
	c.items.Range(func(key, value any) bool {
		c.items.Delete(key)
		return true
	})
}

// Delete deletes an item from the cache
func (c *Cache) Delete(key string) {
	c.items.Delete(key)
}

// Exists reports wether a key exists
func (c *Cache) Exists(key string) bool {
	_, ok := c.Get(key)
	return ok
}

// Get returns an item if it exits and it not stale.
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

// cleanUp removes all expired keys
func (c *Cache) cleanUp() {
	slog.Info("cache clean up: started")
	count := 0
	c.items.Range(func(key, value any) bool {
		_, found := c.Get(key.(string))
		if !found {
			c.Delete(key.(string))
			count++
		}
		return true
	})
	slog.Info("cache clean up: completed", "removed", count)
}
