package xgoesi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/gohugoio/httpcache"
)

var contextForceRefresh contextKey = "forceRefresh"

// NewContextWithForceRefresh marks ctx so ESI requests made with it bypass the local HTTP cache.
func NewContextWithForceRefresh(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextForceRefresh, true)
}

// IsForceRefresh reports whether ctx was marked via [NewContextWithForceRefresh].
func IsForceRefresh(ctx context.Context) bool {
	v, ok := ctx.Value(contextForceRefresh).(bool)
	return ok && v
}

// CacheKeyWithForceRefresh is a [github.com/gohugoio/httpcache.Transport] CacheKey
// func. It returns "" for a force-refreshed request, since ESI has been observed to
// replay stale bodies on 304s for frequently-changing endpoints like the skill queue.
// For non-GET requests it hashes the body into the key, since some ESI POST endpoints
// identify the request by body rather than URL.
func CacheKeyWithForceRefresh(req *http.Request) string {
	if IsForceRefresh(req.Context()) {
		return ""
	}
	if req.Header.Get("Range") != "" {
		return ""
	}
	if req.Method == http.MethodGet {
		return req.URL.String()
	}
	key := req.Method + " " + req.URL.String()
	if req.Body == nil {
		return key
	}
	if req.GetBody == nil {
		// Can't safely re-read the body, so disable caching to avoid a collision.
		return ""
	}
	rc, err := req.GetBody()
	if err != nil {
		return ""
	}
	defer rc.Close()
	body, err := io.ReadAll(rc)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return key + " " + hex.EncodeToString(sum[:])
}

// ResponseFromCache reports whether resp was served from the local HTTP cache.
func ResponseFromCache(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	return resp.Header.Get(httpcache.XFromCache) != ""
}
