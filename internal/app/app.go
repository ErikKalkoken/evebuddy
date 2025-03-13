// Package app is the root package of all domain related packages.
//
// All entity types are defined in this package.
package app

import (
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Default formats and sizes
const (
	DateTimeFormat = "2006.01.02 15:04"
	FloatFormat    = "#,###.##"
	IconPixelSize  = 64
	IconUnitSize   = 32
)

// Titler converts a string into a title for english language.
var Titler = cases.Title(language.English)

// EntityShort is a short representation of an entity.
type EntityShort[T comparable] struct {
	ID   T
	Name string
}

// Position is a position in 3D space.
type Position struct {
	X float64
	Y float64
	Z float64
}

// VariableDateFormat returns a variable dateformat.
func VariableDateFormat(t time.Time) string {
	var dateFormat string
	if isToday(t) {
		dateFormat = "15:04"
	} else if t.Year() == time.Now().UTC().Year() {
		dateFormat = "Jan 2"
	} else {
		dateFormat = "2006.01.02"
	}
	return dateFormat
}

func isToday(t time.Time) bool {
	n := time.Now().UTC()
	return t.Day() == n.Day() && t.Month() == n.Month() && t.Year() == n.Year()
}
