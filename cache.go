package main

import (
	"encoding/binary"
	"time"

	"github.com/gohugoio/httpcache"

	"github.com/ErikKalkoken/evebuddy/internal/app/pcache"
)

// httpCacheAdapter adopts pcache to be used with httpcache.
type httpCacheAdapter struct {
	c       *pcache.PCache
	prefix  string
	timeout time.Duration
}

var _ httpcache.Cache = (*httpCacheAdapter)(nil)

// newHTTPCacheAdapter returns a new cacheAdapter.
// The prefix is added to all cache keys to prevent conflicts.
// Keys are stored with the given cache timeout. A timeout of 0 means that keys never expire.
func newHTTPCacheAdapter(c *pcache.PCache, prefix string, timeout time.Duration) *httpCacheAdapter {
	ca := &httpCacheAdapter{c: c, prefix: prefix, timeout: timeout}
	return ca
}

func (ca *httpCacheAdapter) Get(key string) ([]byte, bool) {
	return ca.c.Get(ca.makeKey(key))
}

func (ca *httpCacheAdapter) Set(key string, b []byte) {
	ca.c.Set(ca.makeKey(key), b, ca.timeout)
}

func (ca *httpCacheAdapter) Delete(key string) {
	ca.c.Delete(ca.makeKey(key))
}

func (ca *httpCacheAdapter) makeKey(key string) string {
	return ca.prefix + key
}

// serviceCacheAdapter adopts pcache to be used with services.
type serviceCacheAdapter struct {
	cache   *pcache.PCache
	prefix  string
	timeout time.Duration
}

// newServiceCacheAdapter returns a new cacheAdapter2.
// The prefix is added to all cache keys to prevent conflicts.
func newServiceCacheAdapter(c *pcache.PCache, prefix string) *serviceCacheAdapter {
	a := &serviceCacheAdapter{cache: c, prefix: prefix}
	return a
}

func (a *serviceCacheAdapter) Delete(key string) {
	a.cache.Delete(key)
}

func (a *serviceCacheAdapter) GetInt64(key string) (int64, bool) {
	b, ok := a.cache.Get(key)
	if !ok {
		return 0, false
	}
	v := int64(binary.BigEndian.Uint64(b))
	return v, true
}

func (a *serviceCacheAdapter) GetString(key string) (string, bool) {
	b, ok := a.cache.Get(key)
	if !ok {
		return "", false
	}
	return string(b), true
}

func (a *serviceCacheAdapter) SetInt64(key string, v int64, timeout time.Duration) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	a.cache.Set(key, b, timeout)
}

func (a *serviceCacheAdapter) SetString(key string, v string, timeout time.Duration) {
	a.cache.Set(key, []byte(v), timeout)
}
