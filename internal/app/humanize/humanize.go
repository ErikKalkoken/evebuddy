// Package humanize transforms values into more user friendly representations.
package humanize

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"syscall"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/sso"
	"github.com/antihax/goesi/esi"
	"github.com/dustin/go-humanize"
	"github.com/mattn/go-sqlite3"
)

// Number returns a humanized number, e.g. 1234 becomes 1.23K
func Number(value float64, decimals int) string {
	var s int
	var a string
	v2 := math.Abs(value)
	switch {
	case v2 >= 1000000000000:
		s = 12
		a = " T"
	case v2 >= 1000000000:
		s = 9
		a = " B"
	case v2 >= 1000000:
		s = 6
		a = " M"
	case v2 >= 1000:
		s = 3
		a = " K"
	default:
		s = 0
		a = ""
	}
	x := value / math.Pow10(s)
	var f string
	switch {
	case decimals == 3:
		f = "%.3f"
	case decimals == 2:
		f = "%.2f"
	case decimals == 1:
		f = "%.1f"
	case decimals == 0:
		f = "%.0f"
	default:
		panic(fmt.Sprintf("Undefined decimals: %d", decimals))
	}
	r := fmt.Sprintf(f, x) + a
	return r
}

// Duration returns a humanized duration, e.g. 22d 10h 5m.
func Duration(duration time.Duration) string {
	m := int(math.Round(duration.Abs().Minutes()))
	d := m / 60 / 24
	m -= d * 60 * 24
	h := m / 60
	m -= h * 60
	if d > 0 {
		return fmt.Sprintf("%dd %dh", d, h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

func Optional[T time.Duration | time.Time | string | int | int32 | int64](o optional.Optional[T], fallback string) string {
	if o.IsNone() {
		return fallback
	}
	v := o.ValueOrZero()
	switch x := any(v).(type) {
	case time.Duration:
		return Duration(x)
	case time.Time:
		return humanize.RelTime(x, time.Now(), "", "")
	case string:
		return x
	case int:
		return Number(float64(x), 0)
	case int32:
		return Number(float64(x), 0)
	case int64:
		return Number(float64(x), 0)
	}
	panic("not implemented")
}

func OptionalFloat[T float32 | float64](o optional.Optional[T], decimals int, fallback string) string {
	if o.IsNone() {
		return fallback
	}
	v := o.ValueOrZero()
	switch x := any(v).(type) {
	case float32:
		return Number(float64(x), decimals)
	case float64:
		return Number(float64(x), decimals)
	}
	panic("not implemented")
}

// Error returns a user friendly error message for an error.
func Error(err error) string {
	if err == nil {
		return "No error"
	}
	if errors.Is(err, sso.ErrTokenError) {
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

// ToRomanLetter returns a number as roman letters.
func ToRomanLetter[N int | int32 | int64 | uint | uint32 | uint64](v N) string {
	m := map[int]string{
		1: "I",
		2: "II",
		3: "III",
		4: "IV",
		5: "V",
	}
	r, ok := m[int(v)]
	if !ok {
		panic(fmt.Sprintf("invalid value: %d", v))
	}
	return r
}

func Time(v time.Time, fallback string) string {
	if v.IsZero() {
		return fallback
	}
	return humanize.Time(v)
}