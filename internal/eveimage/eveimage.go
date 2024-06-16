// package eveimage provides cached access to images on the Eve Online image server.
package eveimage

import "errors"

var (
	ErrHttpError   = errors.New("http error")
	ErrNoImage     = errors.New("no image from API")
	ErrInvalidSize = errors.New("invalid size")
)
