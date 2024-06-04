package mytypes

import "time"

type OptionalDuration struct {
	Duration time.Duration
	Valid    bool
}

type OptionalInt struct {
	Int   int
	Valid bool
}
