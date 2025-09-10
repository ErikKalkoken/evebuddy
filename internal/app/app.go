// Package app is the root package of all domain related packages.
//
// All entity types are defined in this package.
package app

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"syscall"
	"time"

	"github.com/antihax/goesi/esi"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Default formats and sizes
const (
	DateTimeFormat            = "2006.01.02 15:04"
	DateTimeFormatWithSeconds = "2006.01.02 15:04:05"
	DateFormat                = "2006.01.02"
	DateTimeFormatESI         = "2006-01-02T15:04:05Z"
	FloatFormat               = "#,###.##"
	IconPixelSize             = 64
	IconUnitSize              = 28
)

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

// App errors
var (
	ErrAborted       = errors.New("process aborted prematurely")
	ErrAlreadyExists = errors.New("object already exists")
	ErrInvalid       = errors.New("invalid parameters")
	ErrNotFound      = errors.New("object not found")
)

// VariableDateFormat returns a variable format for [time.Time] values.
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

// ErrorDisplay returns a user friendly error message for an error.
func ErrorDisplay(err error) string {
	if err == nil {
		return "No error"
	}
	if errors.Is(err, ErrTokenError) {
		return "token error"
	}
	switch t := err.(type) {
	case sqlite3.Error:
		return "database error"
	case esi.GenericSwaggerError:
		var detail string
		switch t2 := t.Model().(type) {
		case esi.BadRequest:
			detail = t2.Error_
		case esi.ErrorLimited:
			detail = t2.Error_
		case esi.GatewayTimeout:
			detail = t2.Error_
		case esi.Forbidden:
			detail = t2.Error_
		case esi.InternalServerError:
			detail = t2.Error_
		case esi.ServiceUnavailable:
			detail = t2.Error_
		case esi.Unauthorized:
			detail = t2.Error_
		default:
			detail = "General swagger error"
		}
		return fmt.Sprintf("%s: %s", err.Error(), detail)
	case *net.OpError:
		switch t.Op {
		case "dial":
			return "unknown host"
		case "read":
			return "connection refused"
		}
		return "network error"
	case syscall.Errno:
		if t == syscall.ECONNREFUSED {
			return "connection refused"
		}
	case *url.Error:
		return "network error"
	case net.Error:
		if t.Timeout() {
			return "timeout"
		}
		return "network error"
	}
	return "general error"
}

func stringTitle(s string) string {
	titler := cases.Title(language.English)
	return titler.String(s)
}
