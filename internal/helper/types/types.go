package types

import "time"

type NullDuration struct {
	Duration time.Duration
	Valid    bool
}
