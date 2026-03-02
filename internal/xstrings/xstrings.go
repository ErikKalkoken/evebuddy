// Package xstrings provides helpers for strings.
package xstrings

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// CompareIgnoreCase works like [strings.Compare], but is case insensitive.
func CompareIgnoreCase(a, b string) int {
	return strings.Compare(strings.ToLower(a), strings.ToLower(b))
}

// JoinsOrEmpty joins strings together like [strings.Join],
// but returns a fallback when the elem slice is empty.
func JoinsOrEmpty(elems []string, sep, empty string) string {
	if len(elems) == 0 {
		return empty
	}
	return strings.Join(elems, sep)
}

// Title returns the a string with it's first letter upper cased.
func Title(s string) string {
	return cases.Title(language.English).String(s)
}

// Obfuscate returns a new string of the same length as s with all characters replaced
// with a placeholder, except for the last n characters.
func Obfuscate(s string, n int, placeholder string) string {
	if n > len(s) || n < 0 {
		return strings.Repeat(placeholder, len(s))
	}
	return strings.Repeat(placeholder, len(s)-n) + s[len(s)-n:]
}
