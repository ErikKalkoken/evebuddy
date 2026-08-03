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

type equaler[V any] interface {
	Equal(other V) bool
}

// Equal asserts that two objects are equal.
// This variant is type safe
// and will also compare objects with their Equal() methods if available.
func Equal[V any](t *testing.T, want, got V) bool {
	t.Helper()
	got2, ok := any(got).(equaler[V])
	if ok {
		return assert.Truef(t, got2.Equal(want), "Not equal:\nexpected: %s\nactual  : %s", want, got)
	}
	return assert.Equal(t, want, got)
}

// Empty asserts that an optional is empty.
func Empty[V any](t *testing.T, v optional.Optional[V]) bool {
	t.Helper()
	return assert.Truef(t, v.IsEmpty(), "Variable should be empty:\n%v", v)
}

// NotEmpty asserts that an optional is not empty.
func NotEmpty[V any](t *testing.T, v optional.Optional[V]) bool {
	t.Helper()
	return assert.False(t, v.IsEmpty(), "Variable should not be empty")
}

// EqualOptional asserts that the optional is not empty and equal.
func EqualOptional[V any](t *testing.T, want V, got optional.Optional[V]) bool {
	t.Helper()
	if got.IsEmpty() {
		return assert.Fail(t, "Variable should not be empty")
	}
	return Equal(t, want, got.MustValue())
}
