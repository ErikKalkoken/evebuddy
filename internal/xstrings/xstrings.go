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
