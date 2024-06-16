// Package optional provides optional types.
package optional

import "time"

// TODO: Implement other types to substitute sql.NullX types

type Duration struct {
	Duration time.Duration
	Valid    bool
}

type Int struct {
	Int   int
	Valid bool
}
