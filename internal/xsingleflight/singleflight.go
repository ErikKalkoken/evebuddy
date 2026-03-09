// Package xsingleflight provides extensions to Google's singleflight package.
package xsingleflight

import (
	"fmt"

	"golang.org/x/sync/singleflight"
)

// Do is a type safe and panic free variant of Group.Do().
func Do[T any](g *singleflight.Group, key string, fn func() (T, error)) (T, error, bool) {
	if g == nil {
		var z T
		return z, fmt.Errorf("xsingleflight: group missing: %s", key), false
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
	v, ok := x.(T)
	if !ok {
		var z T
		return z, fmt.Errorf("xsingleflight: type conversion failed: %s: got %T, want %T", key, x, z), false
	}
	return v, nil, shared
}
