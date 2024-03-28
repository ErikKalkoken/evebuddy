// Package esi contains all logic for communicating with the ESI API.
// This package should not access any other internal packages, except helpers.
package esi

import memcache "example/esiapp/internal/cache"

const (
	esiBaseUrl = "https://esi.evetech.net/latest"
)

var cache = memcache.New()
