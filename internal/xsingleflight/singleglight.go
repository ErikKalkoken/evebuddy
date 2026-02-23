// Package xsingleflight provides extensions to Google's singleflight package.
package xsingleflight

import (
	"errors"

	"golang.org/x/sync/singleflight"
)

// Do is a type safe variant of Group.Do.
func Do[T any](g *singleflight.Group, key string, fn func() (T, error)) (T, error, bool) {
	if g == nil {
		var z T
		return z, errors.New("missing singleflight group"), false
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
	return x.(T), nil, shared
}
