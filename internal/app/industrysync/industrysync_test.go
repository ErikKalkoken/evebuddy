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
	noAddRelated := func(context.Context, []industrysync.ESIJob) error {
		t.Fatal("addRelated should not be called")
		return nil
	}
	noStore := func(context.Context, industrysync.ESIJob, app.IndustryJobStatus) error {
		t.Fatal("store should not be called")
		return nil
	}
	noMarkUnknown := func(context.Context, set.Set[int64]) error {
		t.Fatal("markUnknown should not be called")
		return nil
	}
	t.Run("reports no change when nothing changed and there are no orphans", func(t *testing.T) {
		changed, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{{JobId: 1, Status: "active", EndDate: future}},
			map[int64]app.IndustryJobStatus{1: app.JobActive},
			statuses,
			noAddRelated,
			noStore,
			noMarkUnknown,
			logAttr,
		)
		require.NoError(t, err)
		assert.False(t, changed)
	})
	t.Run("adds related data and stores new job", func(t *testing.T) {
		var calls []string
		got := make(map[int64]app.IndustryJobStatus)
		changed, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{{JobId: 1, Status: "active", EndDate: future}},
			nil,
			statuses,
			func(_ context.Context, jobs []industrysync.ESIJob) error {
				calls = append(calls, "addRelated")
				assert.Len(t, jobs, 1)
				return nil
			},
			func(_ context.Context, j industrysync.ESIJob, status app.IndustryJobStatus) error {
				calls = append(calls, "store")
				got[j.JobId] = status
				return nil
			},
			noMarkUnknown,
			logAttr,
		)
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Equal(t, []string{"addRelated", "store"}, calls)
		xassert.Equal(t, map[int64]app.IndustryJobStatus{1: app.JobActive}, got)
	})
	t.Run("stores all jobs when the status of one has changed", func(t *testing.T) {
		got := make(map[int64]app.IndustryJobStatus)
		changed, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{
				{JobId: 1, Status: "delivered"},
				{JobId: 2, Status: "active", EndDate: future},
			},
			map[int64]app.IndustryJobStatus{
				1: app.JobActive,
				2: app.JobActive,
			},
			statuses,
			func(context.Context, []industrysync.ESIJob) error {
				return nil
			},
			func(_ context.Context, j industrysync.ESIJob, status app.IndustryJobStatus) error {
				got[j.JobId] = status
				return nil
			},
			noMarkUnknown,
			logAttr,
		)
		require.NoError(t, err)
		assert.True(t, changed)
		want := map[int64]app.IndustryJobStatus{
			1: app.JobDelivered,
			2: app.JobActive,
		}
		xassert.Equal(t, want, got)
	})
	t.Run("compares against the fixed status of jobs which have ended", func(t *testing.T) {
		changed, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{{JobId: 1, Status: "active", EndDate: past}},
			map[int64]app.IndustryJobStatus{1: app.JobReady},
			statuses,
			noAddRelated,
			noStore,
			noMarkUnknown,
			logAttr,
		)
		require.NoError(t, err)
		assert.False(t, changed)
	})
	t.Run("stores the fixed status of jobs which have ended", func(t *testing.T) {
		var got app.IndustryJobStatus
		changed, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{{JobId: 1, Status: "active", EndDate: past}},
			map[int64]app.IndustryJobStatus{1: app.JobActive},
			statuses,
			func(context.Context, []industrysync.ESIJob) error {
				return nil
			},
			func(_ context.Context, _ industrysync.ESIJob, status app.IndustryJobStatus) error {
				got = status
				return nil
			},
			noMarkUnknown,
			logAttr,
		)
		require.NoError(t, err)
		assert.True(t, changed)
		xassert.Equal(t, app.JobReady, got)
	})
	t.Run("marks only active orphans as unknown", func(t *testing.T) {
		var got set.Set[int64]
		changed, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{{JobId: 1, Status: "delivered"}},
			map[int64]app.IndustryJobStatus{
				1: app.JobDelivered,
				2: app.JobActive,
				3: app.JobReady,
				4: app.JobPaused,
				5: app.JobDelivered,
				6: app.JobCancelled,
				7: app.JobReverted,
				8: app.JobUnknown,
			},
			statuses,
			noAddRelated,
			noStore,
			func(_ context.Context, ids set.Set[int64]) error {
				got = ids
				return nil
			},
			logAttr,
		)
		require.NoError(t, err)
		assert.True(t, changed, "should report change when only orphans were marked")
		xassert.Equal(t, set.Of[int64](2, 3, 4), got)
	})
	t.Run("does not mark new jobs as orphans", func(t *testing.T) {
		changed, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{{JobId: 1, Status: "active", EndDate: future}},
			map[int64]app.IndustryJobStatus{},
			statuses,
			func(context.Context, []industrysync.ESIJob) error {
				return nil
			},
			func(context.Context, industrysync.ESIJob, app.IndustryJobStatus) error {
				return nil
			},
			noMarkUnknown,
			logAttr,
		)
		require.NoError(t, err)
		assert.True(t, changed)
	})
	t.Run("stores unknown status as undefined and logs a warning", func(t *testing.T) {
		var buf bytes.Buffer
		old := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
		defer slog.SetDefault(old)
		got := app.JobActive
		_, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{{JobId: 1, Status: "new_status"}},
			nil,
			statuses,
			func(context.Context, []industrysync.ESIJob) error {
				return nil
			},
			func(_ context.Context, _ industrysync.ESIJob, status app.IndustryJobStatus) error {
				got = status
				return nil
			},
			noMarkUnknown,
			logAttr,
		)
		require.NoError(t, err)
		xassert.Equal(t, app.JobUndefined, got)
		assert.Contains(t, buf.String(), "Unknown industry job status encountered")
	})
	t.Run("returns error and stores nothing when adding related data fails", func(t *testing.T) {
		wantErr := errors.New("failed")
		_, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{{JobId: 1, Status: "active", EndDate: future}},
			nil,
			statuses,
			func(context.Context, []industrysync.ESIJob) error {
				return wantErr
			},
			noStore,
			noMarkUnknown,
			logAttr,
		)
		assert.ErrorIs(t, err, wantErr)
	})
	t.Run("returns error when storing fails", func(t *testing.T) {
		wantErr := errors.New("failed")
		_, err := industrysync.Sync(
			ctx,
			[]industrysync.ESIJob{{JobId: 1, Status: "active", EndDate: future}},
			nil,
			statuses,
			func(context.Context, []industrysync.ESIJob) error {
				return nil
			},
			func(context.Context, industrysync.ESIJob, app.IndustryJobStatus) error {
				return wantErr
			},
			noMarkUnknown,
			logAttr,
		)
		assert.ErrorIs(t, err, wantErr)
	})
	t.Run("returns error when marking orphans fails", func(t *testing.T) {
		wantErr := errors.New("failed")
		_, err := industrysync.Sync(
			ctx,
			nil,
			map[int64]app.IndustryJobStatus{1: app.JobActive},
			statuses,
			noAddRelated,
			noStore,
			func(context.Context, set.Set[int64]) error {
				return wantErr
			},
			logAttr,
		)
		assert.ErrorIs(t, err, wantErr)
	})
}
