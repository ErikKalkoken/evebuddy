package xgoesi

import (
	"context"
	"net/http"

	"github.com/gohugoio/httpcache"
)

var contextForceRefresh contextKey = "forceRefresh"

// NewContextWithForceRefresh returns a new context marking that any ESI request made
// with it should bypass the local HTTP cache and force a genuine network fetch,
// instead of potentially returning a cached or 304-revalidated stale response.
func NewContextWithForceRefresh(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextForceRefresh, true)
}

// IsForceRefresh reports whether ctx was marked via [NewContextWithForceRefresh].
func IsForceRefresh(ctx context.Context) bool {
	v, ok := ctx.Value(contextForceRefresh).(bool)
	return ok && v
}

// CacheKeyWithForceRefresh returns the cache key for req for use as a
// [github.com/gohugoio/httpcache.Transport] CacheKey func. It returns "" to
// disable caching for a force-refresh context, a ranged request, or any
// non-GET/HEAD method (ESI's POST responses depend on the body).
func CacheKeyWithForceRefresh(req *http.Request) string {
	if IsForceRefresh(req.Context()) {
		return ""
	}
	if req.Header.Get("Range") != "" {
		return ""
	}
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return ""
	}
	return req.URL.String()
}

// ResponseFromCache reports whether resp was served from the local HTTP cache
// rather than fetched fresh over the network, for use in diagnostic logging.
// It relies on the ESI transport's httpcache.Transport being configured with
// MarkCachedResponses: true. It reports false for a nil response.
func ResponseFromCache(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	return resp.Header.Get(httpcache.XFromCache) != ""
}
