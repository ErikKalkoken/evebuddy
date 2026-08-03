package optional

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqual(t *testing.T) {
	cases := []struct {
		a, b Optional[int]
		want bool
	}{
		{New(5), New(5), true},
		{Optional[int]{}, Optional[int]{}, true},
		{Optional[int]{value: 1}, Optional[int]{value: 2}, true},
	}
	for _, tc := range cases {
		got := Equal(tc.a, tc.b)
		assert.Equal(t, tc.want, got)
	}
}
