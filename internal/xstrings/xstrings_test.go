package xstrings_test

import (
	"strings"
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
			got := xstrings.Obfuscate(tc.s, tc.n, 'X')
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
			expected:  "nov...r",
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
			suffixLen: 2,           // "e "
			expected:  "open...ce", // Space trimmed
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
			expected:  "nove...",
		},
		{
			name:      "suffixLen larger then input",
			input:     "november",
			limit:     7,
			suffixLen: 9,
			expected:  "...mber",
		},
		{
			name:      "empty when limit below 3",
			input:     "november",
			limit:     2,
			suffixLen: 0,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := xstrings.TruncateWithSuffix(tt.input, tt.limit, tt.suffixLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"unchanged for a normal name", "report_2024.txt", "report_2024.txt"},
		{"replaces windows-invalid characters", `a<b>c:d"e/f\g|h?i*j`, "a_b_c_d_e_f_g_h_i_j"},
		{"replaces control characters", "a\x00b\x1fc", "a_b_c"},
		{"trims trailing dots and spaces", "report.txt  ...", "report.txt"},
		{"keeps a leading dot for hidden files", ".gitignore", ".gitignore"},
		{"replaces reserved device name", "CON", "CON_"},
		{"replaces reserved device name case-insensitively", "com1", "com1_"},
		{"replaces reserved device name with extension", "NUL.txt", "NUL_.txt"},
		{"leaves non-reserved name that contains a reserved one", "CONSOLE.txt", "CONSOLE.txt"},
		{"returns fallback for empty string", "", "_"},
		{"replaces path separators", "///", "___"},
		{"returns fallback when only dots and spaces", " . . ", "_"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := xstrings.SanitizeFilename(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
	t.Run("truncates overly long names", func(t *testing.T) {
		in := strings.Repeat("a", 300)
		got := xstrings.SanitizeFilename(in)
		assert.LessOrEqual(t, len([]rune(got)), 255)
		assert.Equal(t, strings.Repeat("a", 255), got)
	})
}

func TestRemoveMultiByte(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ASCII only",
			input:    "Hello, World! 123",
			expected: "Hello, World! 123",
		},
		{
			name:     "Mixed ASCII and multi-byte UTF-8",
			input:    "Hello 世界! 🚀 Test",
			expected: "Hello !  Test",
		},
		{
			name:     "Only multi-byte UTF-8",
			input:    "こんにちは世界",
			expected: "",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Accented characters",
			input:    "Café & Naïve",
			expected: "Caf & Nave",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xstrings.RemoveMultiByte(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
