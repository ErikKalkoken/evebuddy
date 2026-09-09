package app_test

import (
	"fmt"
	"testing"
	"time"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestStatusToImportance(t *testing.T) {
	cases := []struct {
		s    app.Status
		want widget.Importance
	}{
		{app.StatusError, widget.DangerImportance},
		{app.StatusOK, widget.MediumImportance},
		{app.StatusUnknown, widget.LowImportance},
		{app.StatusWorking, widget.MediumImportance},
		{app.StatusPartial, widget.MediumImportance},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.s), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.s.ToImportance())
		})
	}
}

func TestStatusToImportance2(t *testing.T) {
	cases := []struct {
		s    app.Status
		want widget.Importance
	}{
		{app.StatusError, widget.DangerImportance},
		{app.StatusOK, widget.SuccessImportance},
		{app.StatusPartial, widget.SuccessImportance},
		{app.StatusUnknown, widget.LowImportance},
		{app.StatusWorking, widget.MediumImportance},
		{app.Status(99), widget.MediumImportance},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.s), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.s.ToImportance2())
		})
	}
}

func TestStatusSummaryProgressP(t *testing.T) {
	ss := app.StatusSummary{Current: 3, Skipped: 2, Total: 10}
	assert.InDelta(t, 0.5, ss.ProgressP(), 0.001)
}

func TestStatusSummaryStatus(t *testing.T) {
	cases := []struct {
		name string
		ss   app.StatusSummary
		want app.Status
	}{
		{"has errors", app.StatusSummary{Errors: 1, Current: 1, Total: 1}, app.StatusError},
		{"complete with skipped", app.StatusSummary{Current: 4, Skipped: 1, Total: 5}, app.StatusPartial},
		{"complete without skipped", app.StatusSummary{Current: 5, Total: 5}, app.StatusOK},
		{"in progress", app.StatusSummary{Current: 2, Total: 5}, app.StatusWorking},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.ss.Status())
		})
	}
}

func TestStatusSummaryDisplay(t *testing.T) {
	cases := []struct {
		name       string
		ss         app.StatusSummary
		wantShort  string
		wantNormal string
	}{
		{"error", app.StatusSummary{Errors: 2, Current: 1, Total: 5}, "2 ERRORS", "2 ERRORS"},
		{"ok", app.StatusSummary{Current: 5, Total: 5}, "OK", "OK"},
		{"partial", app.StatusSummary{Current: 4, Skipped: 1, Total: 5}, "OK", "Partial"},
		{"working", app.StatusSummary{Current: 2, Total: 4}, "50%", "50%"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			xassert.Equal(t, tc.wantShort, tc.ss.DisplayShort())
			xassert.Equal(t, tc.wantNormal, tc.ss.Display())
		})
	}
}

func TestCacheSectionStatusHasError(t *testing.T) {
	xassert.Equal(t, true, app.CacheSectionStatus{ErrorMessage: "boom"}.HasError())
	xassert.Equal(t, false, app.CacheSectionStatus{}.HasError())
}

func TestCacheSectionStatusHasComment(t *testing.T) {
	xassert.Equal(t, true, app.CacheSectionStatus{Comment: "note"}.HasComment())
	xassert.Equal(t, false, app.CacheSectionStatus{}.HasComment())
}

func TestCacheSectionStatusIsExpired(t *testing.T) {
	t.Run("never completed", func(t *testing.T) {
		xassert.Equal(t, true, app.CacheSectionStatus{}.IsExpired())
	})
	t.Run("expired", func(t *testing.T) {
		ss := app.CacheSectionStatus{CompletedAt: time.Now().Add(-time.Hour), Timeout: time.Minute}
		xassert.Equal(t, true, ss.IsExpired())
	})
	t.Run("not expired", func(t *testing.T) {
		ss := app.CacheSectionStatus{CompletedAt: time.Now(), Timeout: time.Hour}
		xassert.Equal(t, false, ss.IsExpired())
	})
}

func TestCacheSectionStatusIsCurrent(t *testing.T) {
	t.Run("never completed", func(t *testing.T) {
		xassert.Equal(t, false, app.CacheSectionStatus{}.IsCurrent())
	})
	t.Run("current", func(t *testing.T) {
		ss := app.CacheSectionStatus{CompletedAt: time.Now(), Timeout: time.Hour}
		xassert.Equal(t, true, ss.IsCurrent())
	})
	t.Run("stale", func(t *testing.T) {
		ss := app.CacheSectionStatus{CompletedAt: time.Now().Add(-3 * time.Hour), Timeout: time.Minute}
		xassert.Equal(t, false, ss.IsCurrent())
	})
}

func TestCacheSectionStatusIsMissing(t *testing.T) {
	xassert.Equal(t, true, app.CacheSectionStatus{}.IsMissing())
	xassert.Equal(t, false, app.CacheSectionStatus{CompletedAt: time.Now()}.IsMissing())
}

func TestCacheSectionStatusIsRunning(t *testing.T) {
	xassert.Equal(t, false, app.CacheSectionStatus{}.IsRunning())
	xassert.Equal(t, true, app.CacheSectionStatus{StartedAt: time.Now()}.IsRunning())
}

func TestCacheSectionStatusDisplay(t *testing.T) {
	cases := []struct {
		name    string
		ss      app.CacheSectionStatus
		wantStr string
		wantImp widget.Importance
	}{
		{
			"error",
			app.CacheSectionStatus{ErrorMessage: "boom"},
			"ERROR", widget.DangerImportance,
		},
		{
			"missing",
			app.CacheSectionStatus{},
			"Missing", widget.WarningImportance,
		},
		{
			"skipped",
			app.CacheSectionStatus{CompletedAt: time.Now(), Timeout: time.Hour, Comment: "note"},
			"Skipped", widget.MediumImportance,
		},
		{
			"stale",
			app.CacheSectionStatus{CompletedAt: time.Now().Add(-3 * time.Hour), Timeout: time.Minute},
			"Stale", widget.WarningImportance,
		},
		{
			"ok",
			app.CacheSectionStatus{CompletedAt: time.Now(), Timeout: time.Hour},
			"OK", widget.SuccessImportance,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotStr, gotImp := tc.ss.Display()
			xassert.Equal(t, tc.wantStr, gotStr)
			xassert.Equal(t, tc.wantImp, gotImp)
		})
	}
}
