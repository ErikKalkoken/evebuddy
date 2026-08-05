package xstrings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
)

func TestCompareIgnoreCase(t *testing.T) {
	cases := []struct {
		a    string
		b    string
		want int
	}{
		{"alpha", "bravo", -1},
		{"alpha", "alpha", 0},
		{"bravo", "alpha", 1},
		{"alpha", "Bravo", -1},
		{"alpha", "Alpha", 0},
		{"bravo", "Alpha", 1},
	}
	for _, tc := range cases {
		got := xstrings.CompareIgnoreCase(tc.a, tc.b)
		assert.Equal(t, tc.want, got)
	}
}

func TestJoinsOrEmpty(t *testing.T) {
	t.Run("should return joined elements when they exist", func(t *testing.T) {
		got := xstrings.JoinsOrEmpty([]string{"a", "b"}, ",", "?")
		assert.Equal(t, "a,b", got)
	})
	t.Run("should return fallback when elements do not exist", func(t *testing.T) {
		got := xstrings.JoinsOrEmpty([]string{}, ",", "?")
		assert.Equal(t, "?", got)
	})
}

func TestTitle(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"abc", "Abc"},
		{"alpha boy", "Alpha Boy"},
		{"Alpha", "Alpha"},
		{"", ""},
	}
	for _, tc := range cases {
		got := xstrings.Title(tc.in)
		assert.Equal(t, tc.want, got)
	}
}

func TestObfuscate(t *testing.T) {
	cases := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{"normal", "123456789", 4, "XXXXX6789"},
		{"s too short", "123", 4, "XXX"},
		{"n is zero", "123456789", 0, "XXXXXXXXX"},
		{"n is negative", "123456789", -5, "XXXXXXXXX"},
		{"s is empty", "", 4, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := xstrings.Obfuscate(tc.s, tc.n, "X")
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTruncateWithSuffix(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		limit     int
		suffixLen int
		expected  string
	}{
		{
			name:      "Standard truncation",
			input:     "november",
			limit:     7,
			suffixLen: 1,
			expected:  "novem...r",
		},
		{
			name:      "Trailing space removal",
			input:     "november ",
			limit:     8,
			suffixLen: 1,
			expected:  "november",
		},
		{
			name:      "Suffix ends in space",
			input:     "open space ",
			limit:     9,
			suffixLen: 2,             // "e "
			expected:  "open s...ce", // Space trimmed
		},
		{
			name:      "String within limit",
			input:     "hello",
			limit:     10,
			suffixLen: 2,
			expected:  "hello",
		},
		{
			name:      "No suffix",
			input:     "november",
			limit:     7,
			suffixLen: 0,
			expected:  "novemb...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := xstrings.TruncateWithSuffix(tt.input, tt.limit, tt.suffixLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}
