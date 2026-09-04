// Package xsingleflight provides extensions to Google's singleflight package.
package xsingleflight

import (
	"fmt"

	"golang.org/x/sync/singleflight"
)

// Do is a type safe variant of singleflight.Group.Do().
func Do[T any](g *singleflight.Group, key string, fn func() (T, error)) (T, error, bool) {
	if g == nil {
		var z T
		return z, fmt.Errorf("xsingleflight: missing group: %s", key), false
	}
	x, err, shared := g.Do(key, func() (any, error) {
		v, err := fn()
		if err != nil {
			return nil, err
		}
		return v, nil
	})
	if err != nil {
		var z T
		return z, err, false
	}
	if x == nil {
		var z T
		return z, nil, shared
	}
	v, ok := x.(T)
	if !ok {
		var z T
		return z, fmt.Errorf("xsingleflight: type conversion failed: %s: got %T, want %T", key, x, z), false
	}
	return v, nil, shared
}
