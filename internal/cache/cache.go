// Package cache implements a simple in-memory cache.
package cache

import (
	"log/slog"
	"sync"
	"time"
)

const NoTimeout = -1
const defaultCleanupDuration = time.Minute * 10

type Cache struct {
	items sync.Map

	m           sync.Mutex
	lastCleanup time.Time
}

type item struct {
	Value        any
	ExpiresAt    time.Time
	NeverExpires bool
}

// New creates a new cache and returns it
func New() *Cache {
	c := Cache{}
	c.lastCleanup = time.Now()
	return &c
}

// Clear removes all cache keys.
func (c *Cache) Clear() {
	c.items.Range(func(key, value any) bool {
		c.items.Delete(key)
		return true
	})
}

// Delete deletes an item from the cache
func (c *Cache) Delete(key any) {
	c.items.Delete(key)
}

// Exists reports wether a key exists
func (c *Cache) Exists(key any) bool {
	_, ok := c.Get(key)
	return ok
}

// Get returns an item if it exits
func (c *Cache) Get(key any) (any, bool) {
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
func (c *Cache) Set(key any, value any, timeout int) {
	// store the item
	expires := timeout == NoTimeout
	if timeout < 0 {
		timeout = 0
	}
	at := time.Now().Add(time.Second * time.Duration(timeout))
	i := item{Value: value, ExpiresAt: at, NeverExpires: expires}
	c.items.Store(key, i)
	// run cleanup when due
	c.m.Lock()
	defer c.m.Unlock()
	if time.Since(c.lastCleanup) > defaultCleanupDuration {
		c.lastCleanup = time.Now()
		go c.cleanup()
	}
}

// cleanup removes all expires keys
func (c *Cache) cleanup() {
	slog.Debug("Started cleanup")
	c.items.Range(func(key, value any) bool {
		_, found := c.Get(key)
		if !found {
			c.Delete(key)
		}
		return true
	})
}
