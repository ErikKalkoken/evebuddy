package app

import "time"

// Defines a cache service
type CacheService interface {
	Clear()
	Delete(any)
	Exists(any) bool
	Get(any) (any, bool)
	Set(any, any, time.Duration)
}
