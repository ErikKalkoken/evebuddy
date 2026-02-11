package xassert_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEqual2(t *testing.T) {
	s1 := set.Of(1)
	s2 := set.Of(1)
	assert.True(t, xassert.Equal2(t, s1, s2))
}

func TestEmptyOptional(t *testing.T) {
	assert.True(t, xassert.Empty(t, optional.Optional[int]{}))
}
