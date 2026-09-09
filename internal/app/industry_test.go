package app_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestIndustryJobTypeActivities(t *testing.T) {
	got := app.ManufacturingJob.Activities()
	assert.True(t, got.Contains(app.Manufacturing))
	assert.False(t, got.Contains(app.Copying))
}

func TestIndustryJobTypeString(t *testing.T) {
	xassert.Equal(t, "manufacturing job", app.ManufacturingJob.String())
	xassert.Equal(t, "", app.UndefinedJob.String())
}

func TestIndustryJobTypeDisplay(t *testing.T) {
	xassert.Equal(t, "Manufacturing Job", app.ManufacturingJob.Display())
}

func TestIndustryActivityString(t *testing.T) {
	xassert.Equal(t, "manufacturing", app.Manufacturing.String())
	xassert.Equal(t, "?", app.IndustryActivity(99).String())
}

func TestIndustryActivityDisplay(t *testing.T) {
	xassert.Equal(t, "Manufacturing", app.Manufacturing.Display())
}

func TestIndustryActivityJobType(t *testing.T) {
	cases := []struct {
		a    app.IndustryActivity
		want app.IndustryJobType
	}{
		{app.Manufacturing, app.ManufacturingJob},
		{app.Copying, app.ScienceJob},
		{app.Reactions1, app.ReactionJob},
		{app.IndustryActivity(99), app.UndefinedJob},
	}
	for _, tc := range cases {
		t.Run(tc.a.String(), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.a.JobType())
		})
	}
}

func TestIndustryJobStatusString(t *testing.T) {
	xassert.Equal(t, "in progress", app.JobActive.String())
	xassert.Equal(t, "?", app.IndustryJobStatus(99).String())
}

func TestIndustryJobStatusIsActive(t *testing.T) {
	cases := []struct {
		s    app.IndustryJobStatus
		want bool
	}{
		{app.JobActive, true},
		{app.JobReady, true},
		{app.JobPaused, true},
		{app.JobDelivered, false},
		{app.JobCancelled, false},
	}
	for _, tc := range cases {
		t.Run(tc.s.String(), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.s.IsActive())
		})
	}
}

func TestIndustryJobStatusDisplay(t *testing.T) {
	xassert.Equal(t, "In Progress", app.JobActive.Display())
}

func TestIndustryJobStatusColor(t *testing.T) {
	cases := []struct {
		s    app.IndustryJobStatus
		want fyne.ThemeColorName
	}{
		{app.JobActive, theme.ColorNameForeground},
		{app.JobCancelled, theme.ColorNameError},
		{app.JobPaused, theme.ColorNameWarning},
		{app.JobReady, theme.ColorNameSuccess},
		{app.JobDelivered, theme.ColorNameForeground},
	}
	for _, tc := range cases {
		t.Run(tc.s.String(), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.s.Color())
		})
	}
}
