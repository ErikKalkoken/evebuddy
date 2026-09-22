package xgoesi

import "net/http"

// ShouldCacheGetOrHeadOnly restricts httpcache.Transport to caching GET/HEAD,
// since this app's POST responses depend on the request body, which isn't
// part of the cache key.
func ShouldCacheGetOrHeadOnly(req *http.Request, resp *http.Response, key string) bool {
	return req.Method == http.MethodGet || req.Method == http.MethodHead
}
