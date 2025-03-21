package app

import "time"

// Defines a cache service
type CacheService interface {
	Clear()
	Delete(string)
	Exists(string) bool
	Get(string) (any, bool)
	Set(string, any, time.Duration)
}
