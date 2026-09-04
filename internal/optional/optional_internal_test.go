package optional

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqual_Special(t *testing.T) {
	a := Optional[int]{value: 1, isPresent: false}
	b := Optional[int]{value: 2, isPresent: false}
	assert.True(t, Equal(a, b))
}
