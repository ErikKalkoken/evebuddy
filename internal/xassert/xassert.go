// Package xassert extends the testify assert package with additional test helpers.
package xassert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// EqualDuration asserts that got is almost equal to want.
func EqualDuration(t *testing.T, want, got, delta time.Duration) {
	t.Helper()
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	assert.True(t, diff <= delta, "%s is not almost equal to %s (+/- %s)", got, want, delta)
}

type equaler[T any] interface {
	Equal(other T) bool
}

// Equal asserts that two objects are equal.
// This variant is type safe
// and will also compare objects with their Equal() methods if available.
func Equal[T any](t *testing.T, want, got T) bool {
	t.Helper()
	got2, ok := any(got).(equaler[T])
	if ok {
		return assert.Truef(t, got2.Equal(want), "Not equal:\nexpected: %s\nactual  : %s", want, got)
	}
	return assert.Equal(t, want, got)
}

// Empty asserts that an optional is empty.
func Empty[T any](t *testing.T, v optional.Optional[T]) bool {
	t.Helper()
	return assert.Truef(t, v.IsEmpty(), "Not empty:\n%v", v)
}
