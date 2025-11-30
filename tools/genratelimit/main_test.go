package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToAnchor(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Alpha", "alpha"},
		{"alpha", "alpha"},
		{"Alpha Delta", "alpha-delta"},
	}
	for _, tc := range cases {
		got := toAnchor(tc.in)
		assert.Equal(t, tc.want, got)
	}
}
