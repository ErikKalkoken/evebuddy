package industrysync

import (
	"log/slog"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestParseStatus(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	statuses := map[string]app.IndustryJobStatus{
		"active":    app.JobActive,
		"delivered": app.JobDelivered,
		"ready":     app.JobReady,
	}
	cases := []struct {
		name    string
		status  string
		endDate time.Time
		want    app.IndustryJobStatus
	}{
		{"keeps active job which has not ended", "active", future, app.JobActive},
		{"fixes active job which has ended", "active", past, app.JobReady},
		{"keeps active job without end date", "active", time.Time{}, app.JobActive},
		{"keeps active job which ends now", "active", now, app.JobActive},
		{"keeps other status of job which has ended", "delivered", past, app.JobDelivered},
		{"keeps ready job which has ended", "ready", past, app.JobReady},
		{"returns undefined for unknown status", "new_status", future, app.JobUndefined},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseStatus(ESIJob{Status: tc.status, EndDate: tc.endDate}, statuses, now, slog.Int64("ownerID", 42))
			xassert.Equal(t, tc.want, got)
		})
	}
}
