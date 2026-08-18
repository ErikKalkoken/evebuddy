// Package xstrings provides helpers for strings.
package xstrings

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var folder = cases.Fold(cases.NoLower)

// CompareIgnoreCase works like [strings.Compare], but is case insensitive.
func CompareIgnoreCase(a, b string) int {
	return strings.Compare(folder.String(a), folder.String(b))
}

// JoinsOrEmpty joins strings together like [strings.Join],
// but returns a fallback when the elem slice is empty.
func JoinsOrEmpty(elems []string, sep, empty string) string {
	if len(elems) == 0 {
		return empty
	}
	return strings.Join(elems, sep)
}

// Obfuscate returns a new string of the same length as s with all characters replaced
// with a placeholder, except for the last n characters.
func Obfuscate(s string, n int, placeholder rune) string {
	if placeholder > 127 {
		placeholder = 'X'
	}
	s2 := RemoveMultiByte(s)
	if n > len(s2) || n < 0 {
		return strings.Repeat(string(placeholder), len(s2))
	}
	return strings.Repeat(string(placeholder), len(s2)-n) + s2[len(s2)-n:]
}

// RemoveMultiByte returns a new string where all multi-byte UTF-8 characters have been removed.
func RemoveMultiByte(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] <= 127 {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// Title returns a string with the first letter of every word upper cased.
func Title(s string) string {
	return cases.Title(language.English).String(s)
}

// TruncateWithSuffix shortens the length of s to limit amount of characters
// and adds an ellipsis when the string was shortened.
// It can optionally keep suffixLen characters at the end.
func TruncateWithSuffix(s string, limit int, suffixLen int) string {
	if limit < 3 {
		return ""
	}
	runes := []rune(strings.TrimRight(s, " "))
	if len(runes) <= limit {
		return string(runes)
	}
	suffixLen2 := min(limit-3, suffixLen)
	prefixLen := max(limit-3-suffixLen2, 0) // ellipsis counts as 3
	prefix := runes[:prefixLen]
	suffix := runes[len(runes)-suffixLen2:]
	strSuffix := strings.TrimRight(string(suffix), " ")
	return string(prefix) + "..." + strSuffix
}
