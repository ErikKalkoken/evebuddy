package xgoesi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalcTimePeriod(t *testing.T) {
	// Fixed reference time for consistent testing
	referenceTime := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	expectedDay := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		day           time.Time
		start         string
		finish        string
		expectStart   time.Time
		expectFinish  time.Time
		expectError   bool
		errorContains string
	}{
		{
			name:         "valid time period - hours",
			day:          referenceTime,
			start:        "9h",
			finish:       "17h",
			expectStart:  expectedDay.Add(9 * time.Hour),
			expectFinish: expectedDay.Add(17 * time.Hour),
			expectError:  false,
		},
		{
			name:         "valid time period - hours and minutes",
			day:          referenceTime,
			start:        "8h30m",
			finish:       "16h45m",
			expectStart:  expectedDay.Add(8*time.Hour + 30*time.Minute),
			expectFinish: expectedDay.Add(16*time.Hour + 45*time.Minute),
			expectError:  false,
		},
		{
			name:         "valid time period - complex duration",
			day:          referenceTime,
			start:        "1h30m15s",
			finish:       "23h59m59s",
			expectStart:  expectedDay.Add(1*time.Hour + 30*time.Minute + 15*time.Second),
			expectFinish: expectedDay.Add(23*time.Hour + 59*time.Minute + 59*time.Second),
			expectError:  false,
		},
		{
			name:         "zero start time",
			day:          referenceTime,
			start:        "0h",
			finish:       "8h",
			expectStart:  expectedDay,
			expectFinish: expectedDay.Add(8 * time.Hour),
			expectError:  false,
		},
		{
			name:         "midnight to midnight next day",
			day:          referenceTime,
			start:        "0h",
			finish:       "24h",
			expectStart:  expectedDay,
			expectFinish: expectedDay.Add(24 * time.Hour),
			expectError:  false,
		},
		{
			name:          "invalid start time format",
			day:           referenceTime,
			start:         "invalid",
			finish:        "17h",
			expectError:   true,
			errorContains: "parsing start time",
		},
		{
			name:          "invalid finish time format",
			day:           referenceTime,
			start:         "9h",
			finish:        "invalid",
			expectError:   true,
			errorContains: "parsing end time",
		},
		{
			name:          "empty start time",
			day:           referenceTime,
			start:         "",
			finish:        "17h",
			expectError:   true,
			errorContains: "parsing start time",
		},
		{
			name:          "empty finish time",
			day:           referenceTime,
			start:         "9h",
			finish:        "",
			expectError:   true,
			errorContains: "parsing end time",
		},
		{
			name:         "non-UTC timezone is converted",
			day:          time.Date(2024, 3, 15, 14, 30, 45, 0, time.FixedZone("EST", -5*3600)),
			start:        "10h",
			finish:       "18h",
			expectStart:  expectedDay.Add(10 * time.Hour),
			expectFinish: expectedDay.Add(18 * time.Hour),
			expectError:  false,
		},
		{
			name:         "finish before start is allowed",
			day:          referenceTime,
			start:        "22h",
			finish:       "6h",
			expectStart:  expectedDay.Add(22 * time.Hour),
			expectFinish: expectedDay.Add(6 * time.Hour),
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, finish, err := calcTimePeriod(tt.day, tt.start, tt.finish)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectStart, start, "start time mismatch")
				assert.Equal(t, tt.expectFinish, finish, "finish time mismatch")
			}
		})
	}
}

func TestIsInPeriod(t *testing.T) {
	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	startTime := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	finishTime := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		myTime   time.Time
		start    time.Time
		finish   time.Time
		expected bool
	}{
		{
			name:     "time is within period",
			myTime:   baseTime,
			start:    startTime,
			finish:   finishTime,
			expected: true,
		},
		{
			name:     "time is before period",
			myTime:   time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
			start:    startTime,
			finish:   finishTime,
			expected: false,
		},
		{
			name:     "time is after period",
			myTime:   time.Date(2024, 1, 25, 0, 0, 0, 0, time.UTC),
			start:    startTime,
			finish:   finishTime,
			expected: false,
		},
		{
			name:     "time equals start time",
			myTime:   startTime,
			start:    startTime,
			finish:   finishTime,
			expected: true,
		},
		{
			name:     "time equals finish time",
			myTime:   finishTime,
			start:    startTime,
			finish:   finishTime,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInPeriod(tt.myTime, tt.start, tt.finish)
			assert.Equal(t, tt.expected, result)
		})
	}
}
