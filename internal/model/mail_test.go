package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatchLinks(t *testing.T) {
	var testCases = []struct {
		in   string
		want string
	}{
		{"first", "first"},
		{"[link](http://www.google.com)\n\nsecond", "[link](http://www.google.com) \n\nsecond"},
		{"first\n\nsecond", "first\n\nsecond"},
		{"[link](http://www.google.com)", "[link](http://www.google.com)"},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("in: %s out: %s", tc.in, tc.want), func(t *testing.T) {
			got := patchLinks(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}
