package xgoesi

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// daily downtime. Time given in UTC.
const (
	dailyDowntimeStart  = "11h0m"
	dailyDowntimeFinish = "11h15m"
)

var (
	TimeNow = time.Now
)

// IsDailyDowntime reports whether the daily downtime is currently planned to happen.
func IsDailyDowntime() bool {
	start, finish, err := calcTimePeriod(TimeNow(), dailyDowntimeStart, dailyDowntimeFinish)
	if err != nil {
		slog.Error("Failed to determine daily downtime", "error", err)
		return false
	}
	return isInPeriod(TimeNow(), start, finish)
}

// DailyDowntime returns the daily downtime period.
func DailyDowntime() (start, finish time.Time) {
	var err error
	start, finish, err = calcTimePeriod(TimeNow(), dailyDowntimeStart, dailyDowntimeFinish)
	if err != nil {
		slog.Error("Failed to determine daily downtime", "error", err)
	}
	return
}

func isInPeriod(t, start, finish time.Time) bool {
	if t.Before(start) {
		return false
	}
	if t.After(finish) {
		return false
	}
	return true
}

func calcTimePeriod(day time.Time, start string, finish string) (time.Time, time.Time, error) {
	day2 := day.UTC().Truncate(24 * time.Hour)
	d1, err := time.ParseDuration(start)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parsing start time %s: %w", start, err)
	}
	start2 := day2.Add(d1)
	d2, err := time.ParseDuration(finish)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parsing end time %s: %w", finish, err)
	}
	finish2 := day2.Add(d2)
	return start2, finish2, nil
}

// DowntimeBlocker is a HTTP transport that blocks all requests during the planned daily downtime.
//
// When blocking it returns a response with the status "503 Service Unavailable"
// and a retry after header which points to the end of the planned downtime.
//
// This transport helps to prevent generating a lot of unnecessary errors
// from the ESI servers when trying to access ESI during the daily downtime.
type DowntimeBlocker struct {
	// The RoundTripper interface actually used to make requests
	// If nil, http.DefaultTransport is used
	Transport http.RoundTripper
}

var _ http.RoundTripper = (*DowntimeBlocker)(nil)

func (rl *DowntimeBlocker) RoundTrip(req *http.Request) (*http.Response, error) {
	transport := rl.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	start, finish := DailyDowntime()
	now := TimeNow()
	if isInPeriod(now, start, finish) {
		retryAfter := int(finish.Sub(now).Seconds()) + 1
		resp, err := createErrorResponse(req, http.StatusServiceUnavailable, retryAfter, "ESI requests are blocked during daily downtime")
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
