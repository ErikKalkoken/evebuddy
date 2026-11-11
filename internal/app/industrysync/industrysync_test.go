package industrysync_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/industrysync"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

var (
	logAttr  = slog.Int64("ownerID", 42)
	statuses = map[string]app.IndustryJobStatus{
		"active":    app.JobActive,
		"delivered": app.JobDelivered,
		"ready":     app.JobReady,
	}
)

func noSave(t *testing.T) func(context.Context, []industrysync.ESIJob, []app.IndustryJobStatus) (bool, error) {
	return func(context.Context, []industrysync.ESIJob, []app.IndustryJobStatus) (bool, error) {
		t.Fatal("save should not be called")
		return false, nil
	}
}

func TestExtractIDs(t *testing.T) {
	completer := int64(3)
	product := int64(20)
	entityIDs, typeIDs := industrysync.ExtractIDs([]industrysync.ESIJob{
		{InstallerId: 1, BlueprintTypeId: 10},
		{InstallerId: 2, BlueprintTypeId: 10, CompletedCharacterId: &completer, ProductTypeId: &product},
	})
	xassert.Equal(t, set.Of[int64](1, 2, 3), entityIDs)
	xassert.Equal(t, set.Of[int64](10, 20), typeIDs)
}

func TestSync(t *testing.T) {
	ctx := context.Background()
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)
	t.Run("adds related data and then saves jobs with their status", func(t *testing.T) {
		// given
		var calls []string
		got := make(map[int64]app.IndustryJobStatus)
		// when
		changed, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{
				{JobId: 1, Status: "active", EndDate: future},
				{JobId: 2, Status: "delivered"},
			},
			statuses,
			func(context.Context) error {
				calls = append(calls, "addRelated")
				return nil
			},
			func(_ context.Context, jobs []industrysync.ESIJob, jobStatuses []app.IndustryJobStatus) (bool, error) {
				calls = append(calls, "save")
				for i, j := range jobs {
					got[j.JobId] = jobStatuses[i]
				}
				return true, nil
			},
			logAttr,
		)
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Equal(t, []string{"addRelated", "save"}, calls)
		want := map[int64]app.IndustryJobStatus{
			1: app.JobActive,
			2: app.JobDelivered,
		}
		xassert.Equal(t, want, got)
	})
	t.Run("reports whether save has changed anything", func(t *testing.T) {
		cases := []struct {
			name    string
			changed bool
		}{
			{"changed", true},
			{"unchanged", false},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// when
				changed, err := industrysync.Sync(
					ctx,
					[]industrysync.ESIJob{{JobId: 1, Status: "active", EndDate: future}},
					statuses,
					func(context.Context) error {
						return nil
					},
					func(context.Context, []industrysync.ESIJob, []app.IndustryJobStatus) (bool, error) {
						return tc.changed, nil
					},
					logAttr,
				)
				// then
				require.NoError(t, err)
				xassert.Equal(t, tc.changed, changed)
			})
		}
	})
	t.Run("saves the fixed status of jobs which have ended", func(t *testing.T) {
		// given
		var got []app.IndustryJobStatus
		// when
		_, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{{JobId: 1, Status: "active", EndDate: past}},
			statuses,
			func(context.Context) error {
				return nil
			},
			func(_ context.Context, _ []industrysync.ESIJob, jobStatuses []app.IndustryJobStatus) (bool, error) {
				got = jobStatuses
				return true, nil
			},
			logAttr,
		)
		// then
		require.NoError(t, err)
		xassert.Equal(t, []app.IndustryJobStatus{app.JobReady}, got)
	})
	t.Run("saves unknown status as undefined and logs a warning", func(t *testing.T) {
		// given
		var buf bytes.Buffer
		old := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
		defer slog.SetDefault(old)
		var got []app.IndustryJobStatus
		// when
		_, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{{JobId: 1, Status: "new_status"}},
			statuses,
			func(context.Context) error {
				return nil
			},
			func(_ context.Context, _ []industrysync.ESIJob, jobStatuses []app.IndustryJobStatus) (bool, error) {
				got = jobStatuses
				return true, nil
			},
			logAttr,
		)
		// then
		require.NoError(t, err)
		xassert.Equal(t, []app.IndustryJobStatus{app.JobUndefined}, got)
		assert.Contains(t, buf.String(), "Unknown industry job status encountered")
		assert.Contains(t, buf.String(), "jobID=1")
		assert.Contains(t, buf.String(), "status=new_status")
	})
	t.Run("saves only the last of duplicate jobs and logs a warning", func(t *testing.T) {
		// given
		var buf bytes.Buffer
		old := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
		defer slog.SetDefault(old)
		got := make(map[int64][]app.IndustryJobStatus)
		// when
		_, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{
				{JobId: 1, Status: "active", EndDate: future},
				{JobId: 2, Status: "active", EndDate: future},
				{JobId: 1, Status: "delivered"},
			},
			statuses,
			func(context.Context) error {
				return nil
			},
			func(_ context.Context, jobs []industrysync.ESIJob, jobStatuses []app.IndustryJobStatus) (bool, error) {
				for i, j := range jobs {
					got[j.JobId] = append(got[j.JobId], jobStatuses[i])
				}
				return true, nil
			},
			logAttr,
		)
		// then
		require.NoError(t, err)
		want := map[int64][]app.IndustryJobStatus{
			1: {app.JobDelivered},
			2: {app.JobActive},
		}
		xassert.Equal(t, want, got)
		assert.Contains(t, buf.String(), "Duplicate industry jobs received from ESI")
	})
	t.Run("returns error and saves nothing when adding related data fails", func(t *testing.T) {
		// given
		wantErr := errors.New("failed")
		// when
		_, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{{JobId: 1, Status: "active", EndDate: future}},
			statuses,
			func(context.Context) error {
				return wantErr
			},
			noSave(t),
			logAttr,
		)
		// then
		assert.ErrorIs(t, err, wantErr)
	})
	t.Run("returns error when saving fails", func(t *testing.T) {
		// given
		wantErr := errors.New("failed")
		// when
		_, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{{JobId: 1, Status: "active", EndDate: future}},
			statuses,
			func(context.Context) error {
				return nil
			},
			func(context.Context, []industrysync.ESIJob, []app.IndustryJobStatus) (bool, error) {
				return false, wantErr
			},
			logAttr,
		)
		// then
		assert.ErrorIs(t, err, wantErr)
	})
}
