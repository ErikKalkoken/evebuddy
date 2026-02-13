package xassert_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEqual(t *testing.T) {
	t.Run("should use Equal() method to compare when type has it", func(t *testing.T) {
		a := set.Of[int]()
		b := set.Set[int]{}
		xassert.Equal(t, a, b)
	})
	t.Run("should compare normally when type has no Equal method", func(t *testing.T) {
		a := 1
		b := 1
		xassert.Equal(t, a, b)
	})
}

func TestEmptyOptional(t *testing.T) {
	assert.True(t, xassert.Empty(t, optional.Optional[int]{}))
}
