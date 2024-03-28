// Package cache implements a simple in-memory cache.
package cache

import (
	"sync"
	"time"
)

const NoTimeout = -1

type Cache struct {
	items sync.Map
}

type item struct {
	Value        any
	ExpiresAt    time.Time
	NeverExpires bool
}

// Clear removes all cache keys.
func (c *Cache) Clear() {
	c.items.Range(func(key, value any) bool {
		c.items.Delete(key)
		return true
	})
}

// Exists reports wether a key exists
func (c *Cache) Exists(key string) bool {
	_, ok := c.Get(key)
	return ok
}

// Get returns an item if it exits
func (c *Cache) Get(key string) (any, bool) {
	value, ok := c.items.Load(key)
	if !ok {
		return nil, false
	}
	i := value.(item)
	if !i.NeverExpires && time.Until(i.ExpiresAt) < 0 {
		return nil, false
	}
	return i.Value, ok
}

// Set stores an item in the cache.
//
// If an item with the same key already exists it will be overwritten.
// An item with timeout = cache.NoTimeout never expires
func (c *Cache) Set(key string, value any, timeout int) {
	expires := timeout == NoTimeout
	if timeout < 0 {
		timeout = 0
	}
	at := time.Now().Add(time.Second * time.Duration(timeout))
	i := item{Value: value, ExpiresAt: at, NeverExpires: expires}
	c.items.Store(key, i)
}

// Delete deletes an item from the cache
func (c *Cache) Delete(key string) {
	c.items.Delete(key)
}

// New creates a new cache and returns it
func New() *Cache {
	c := Cache{}
	return &c
}
