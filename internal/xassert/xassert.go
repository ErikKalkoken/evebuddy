// Package xassert extends the testify assert package with additional test helpers.
package xassert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

// Equal asserts that two objects are equal.
// This variant is type safe.
func Equal[T any](t *testing.T, want, got T) bool {
	t.Helper()
	return assert.Equal(t, want, got)
}

type Equaler[T any] interface {
	Equal(other T) bool
}

// Equal2 asserts that two values which satisfy the equaler interface are equal .
func Equal2[T Equaler[T]](t *testing.T, want, got T) {
	t.Helper()
	assert.Truef(t, got.Equal(want), "Not equal:\nexpected: %s\nactual  : %s", want, got)
}
