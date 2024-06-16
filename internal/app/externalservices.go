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

type DictionaryService interface {
	Delete(string) error
	Float32(string) (float32, bool, error)
	Int(string) (int, bool, error)
	IntWithFallback(string, int) (int, error)
	SetFloat32(string, float32) error
	Float64(key string) (float64, bool, error)
	SetFloat64(key string, value float64) error
	SetInt(string, int) error
	SetString(string, string) error
	String(string) (string, bool, error)
}
