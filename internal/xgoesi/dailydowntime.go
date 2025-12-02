package xgoesi

import (
	"fmt"
	"log/slog"
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
