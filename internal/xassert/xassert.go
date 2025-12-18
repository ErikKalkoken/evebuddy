// Package xassert extends the testify assert package with additional test helpers.
package xassert

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
)

// EqualDuration asserts that got is almost equal to want.
func EqualDuration(t *testing.T, got, want, delta time.Duration) {
	t.Helper()
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	assert.True(t, diff <= delta, "%s is not almost equal to %s (+/- %s)", got, want, delta)
}

// EqualSet asserts that two sets are equal.
func EqualSet[T comparable](t *testing.T, got, want set.Set[T]) {
	t.Helper()
	assert.Truef(t, got.Equal(want), "Not equal:\nexpected: %s\nactual  : %s", want, got)
}
