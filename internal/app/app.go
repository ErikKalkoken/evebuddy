// Package app is the root package of all domain related packages.
//
// All entity types are defined in this package.
package app

import (
	"errors"
	"net"
	"net/url"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"github.com/fnt-eve/goesi-openapi/esi"
	"github.com/mattn/go-sqlite3"
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
type EntityShort struct {
	ID   int64
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
	ErrAlreadyExists = errors.New("object already exists")
	ErrInvalid       = errors.New("invalid parameters")
	ErrNotFound      = errors.New("object not found")
)

var (
	isDeveloperMode bool
	isMobile        bool
)

func SetDeveloperMode(b bool) {
	isDeveloperMode = b
}

func IsDeveloperMode() bool {
	return isDeveloperMode
}

func SetIsMobile(b bool) {
	isMobile = b
}

func IsMobile() bool {
	return isMobile
}

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
// Or returns the full error when in developer mode.
func ErrorDisplay(err error) string {
	if IsDeveloperMode() {
		return err.Error()
	}
	if err == nil {
		return "No error"
	}
	if errors.Is(err, ErrTokenError) {
		return "token error"
	}
	switch x := err.(type) {
	case sqlite3.Error:
		return "database error"
	case *esi.GenericOpenAPIError:
		msg := x.Error()
		if x, ok := x.Model().(esi.Error); ok {
			msg += ": " + x.Error
		}
		return msg
	case *net.OpError:
		switch x.Op {
		case "dial":
			return "unknown host"
		case "read":
			return "connection refused"
		}
		return "network error"
	case syscall.Errno:
		if x == syscall.ECONNREFUSED {
			return "connection refused"
		}
	case *url.Error:
		return "network error"
	case net.Error:
		if x.Timeout() {
			return "timeout"
		}
		return "network error"
	}
	return "general error"
}

// Name returns the name for this app.
func Name() string {
	info := fyne.CurrentApp().Metadata()
	name := info.Name
	if name == "" {
		return "EVE Buddy"
	}
	return name
}
