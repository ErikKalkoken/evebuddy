package main

import (
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/pcache"
	"github.com/gregjones/httpcache"
)

// cacheAdapter enabled the use of pcache with httpcache
type cacheAdapter struct {
	c       *pcache.PCache
	prefix  string
	timeout time.Duration
}

var _ httpcache.Cache = (*cacheAdapter)(nil)

// newCacheAdapter returns a new cacheAdapter.
// The prefix is added to all cache keys to prevent conflicts.
// Keys are stored with the given cache timeout. A timeout of 0 means that keys never expire.
func newCacheAdapter(c *pcache.PCache, prefix string, timeout time.Duration) *cacheAdapter {
	ca := &cacheAdapter{c: c, prefix: prefix, timeout: timeout}
	return ca
}

func (ca *cacheAdapter) Get(key string) ([]byte, bool) {
	return ca.c.Get(ca.makeKey(key))
}

func (ca *cacheAdapter) Set(key string, b []byte) {
	ca.c.Set(ca.makeKey(key), b, ca.timeout)
}

func (ca *cacheAdapter) Delete(key string) {
	ca.c.Delete(ca.makeKey(key))
}

func (ca *cacheAdapter) makeKey(key string) string {
	return ca.prefix + key
}
