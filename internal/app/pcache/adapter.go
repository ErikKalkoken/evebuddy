package pcache

import (
	"encoding/binary"
	"time"

	"github.com/gohugoio/httpcache"
)

// HTTPCacheAdapter adopts pcache to be used with httpcache.
type HTTPCacheAdapter struct {
	c       *PCache
	prefix  string
	timeout time.Duration
}

var _ httpcache.Cache = (*HTTPCacheAdapter)(nil)

// NewHTTPCacheAdapter returns a new HTTPCacheAdapter.
// The prefix is added to all cache keys to prevent conflicts.
// Keys are stored with the given cache timeout. A timeout of 0 means that keys never expire.
func NewHTTPCacheAdapter(c *PCache, prefix string, timeout time.Duration) *HTTPCacheAdapter {
	a := &HTTPCacheAdapter{c: c, prefix: prefix, timeout: timeout}
	return a
}

func (a *HTTPCacheAdapter) Get(key string) ([]byte, bool) {
	return a.c.Get(a.makeKey(key))
}

func (a *HTTPCacheAdapter) Set(key string, b []byte) {
	a.c.Set(a.makeKey(key), b, a.timeout)
}

func (a *HTTPCacheAdapter) Delete(key string) {
	a.c.Delete(a.makeKey(key))
}

func (a *HTTPCacheAdapter) makeKey(key string) string {
	return a.prefix + ":" + key
}

// ServiceCacheAdapter adopts pcache to be used with services.
type ServiceCacheAdapter struct {
	cache  *PCache
	prefix string
}

// NewServiceCacheAdapter returns a new ServiceCacheAdapter.
// The prefix is added to all cache keys to prevent conflicts.
func NewServiceCacheAdapter(c *PCache, prefix string) *ServiceCacheAdapter {
	a := &ServiceCacheAdapter{cache: c, prefix: prefix}
	return a
}

func (a *ServiceCacheAdapter) Delete(key string) {
	a.cache.Delete(a.makeKey(key))
}

func (a *ServiceCacheAdapter) GetInt64(key string) (int64, bool) {
	b, ok := a.cache.Get(a.makeKey(key))
	if !ok || len(b) < 8 {
		return 0, false
	}
	v := int64(binary.BigEndian.Uint64(b))
	return v, true
}

func (a *ServiceCacheAdapter) GetString(key string) (string, bool) {
	b, ok := a.cache.Get(a.makeKey(key))
	if !ok {
		return "", false
	}
	return string(b), true
}

func (a *ServiceCacheAdapter) SetInt64(key string, v int64, timeout time.Duration) {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	a.cache.Set(a.makeKey(key), b, timeout)
}

func (a *ServiceCacheAdapter) SetString(key string, v string, timeout time.Duration) {
	a.cache.Set(a.makeKey(key), []byte(v), timeout)
}

func (a *ServiceCacheAdapter) makeKey(key string) string {
	return a.prefix + ":" + key
}
