package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestIndustryJob_Filter(t *testing.T) {
	t.Skip("Temporarily disabled as they are now flaky with filtering running async") // TODO
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	j1 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		ActivityID: int64(app.Manufacturing),
		Status:     app.JobReady,
	})
	j2 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		ActivityID: int64(app.Copying),
		Status:     app.JobReady,
	})
	j3 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		ActivityID: int64(app.Reactions1),
		Status:     app.JobReady,
	})
	j4 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		ActivityID: int64(app.Reactions2),
		Status:     app.JobReady,
	})
	a := NewJobsForOverview(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	}))
	a.update(t.Context())

	t.Run("no filter", func(t *testing.T) {
		a.selectActivity.SetSelected("")

		got := xslices.Map(a.rowsFiltered, func(r industryJobRow) int64 {
			return r.jobID
		})
		want := []int64{j1.JobID, j2.JobID, j3.JobID, j4.JobID}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("can filter manufacturing", func(t *testing.T) {
		a.selectActivity.SetSelected("Manufacturing")

		got := xslices.Map(a.rowsFiltered, func(r industryJobRow) int64 {
			return r.jobID
		})
		want := []int64{j1.JobID}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("can filter reactions", func(t *testing.T) {
		a.selectActivity.SetSelected("Reactions")

		got := xslices.Map(a.rowsFiltered, func(r industryJobRow) int64 {
			return r.jobID
		})
		want := []int64{j3.JobID, j4.JobID}
		assert.ElementsMatch(t, want, got)
	})
}

func TestIndustryJob_FetchJobs(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec := factory.CreateEveCharacter()
	character := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
	corporation := factory.CreateCorporation(ec.Corporation.ID)
	j1 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		ActivityID:  int64(app.Manufacturing),
		CharacterID: character.ID,
		JobID:       1,
		Status:      app.JobReady,
	})
	j2 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
		ActivityID:    int64(app.Copying),
		CorporationID: ec.Corporation.ID,
		JobID:         2,
		Status:        app.JobDelivered,
		InstallerID:   character.ID,
	})
	c2 := factory.CreateEveEntityCharacter()
	j3 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
		ActivityID:    int64(app.Copying),
		CorporationID: ec.Corporation.ID,
		JobID:         3,
		Status:        app.JobDelivered,
		InstallerID:   c2.ID,
	})

	t.Run("can return all character and relevant corporation jobs", func(t *testing.T) {
		a := NewJobsForOverview(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.update(t.Context())
		a.corporation.Store(corporation)
		xx, err := a.fetchCombinedJobs(t.Context())
		require.NoError(t, err)
		want := set.Of(j1.JobID, j2.JobID)
		got := set.Collect(xiter.MapSlice(xx, func(x industryJobRow) int64 {
			return x.jobID
		}))
		xassert.Equal(t, want, got)
	})

	t.Run("can return all jobs for current corporation", func(t *testing.T) {
		a := NewJobsForCorporation(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.corporation.Store(corporation)
		a.update(t.Context())

		xx, err := a.fetchCorporationJobs(t.Context())
		require.NoError(t, err)
		want := set.Of(j2.JobID, j3.JobID)
		got := set.Collect(xiter.MapSlice(xx, func(x industryJobRow) int64 {
			return x.jobID
		}))
		xassert.Equal(t, want, got)
	})
}

func TestIndustryJobsFilter_Match(t *testing.T) {
	r := industryJobRow{
		activity:      app.Manufacturing,
		isInstallerMe: true,
		isOwnerMe:     true,
		status:        app.JobReady,
		tags:          set.Of("alpha"),
	}
	byCorpmate := r
	byCorpmate.isInstallerMe = false
	ownedByCorp := r
	ownedByCorp.isOwnerMe = false
	reaction := r
	reaction.activity = app.Reactions2
	paused := r
	paused.status = app.JobPaused
	delivered := r
	delivered.status = app.JobDelivered
	for _, tc := range []struct {
		name   string
		filter industryJobsFilter
		row    industryJobRow
		want   bool
	}{
		{"no filter", industryJobsFilter{}, r, true},
		{"installed by me matches", industryJobsFilter{installer: industryInstallerMe}, r, true},
		{"installed by me but corpmate", industryJobsFilter{installer: industryInstallerMe}, byCorpmate, false},
		{"installed by corpmates matches", industryJobsFilter{installer: industryInstallerCorpmates}, byCorpmate, true},
		{"installed by corpmates but me", industryJobsFilter{installer: industryInstallerCorpmates}, r, false},
		{"owned by me matches", industryJobsFilter{owner: industryOwnerMe}, r, true},
		{"owned by me but corp", industryJobsFilter{owner: industryOwnerMe}, ownedByCorp, false},
		{"owned by corp matches", industryJobsFilter{owner: industryOwnerCorp}, ownedByCorp, true},
		{"owned by corp but me", industryJobsFilter{owner: industryOwnerCorp}, r, false},
		{"tag matches", industryJobsFilter{tag: "alpha"}, r, true},
		{"tag missing", industryJobsFilter{tag: "bravo"}, r, false},
		{"manufacturing matches", industryJobsFilter{activity: industryActivityManufacturing}, r, true},
		{"copying differs", industryJobsFilter{activity: industryActivityCopying}, r, false},
		{"reactions matches reactions 2", industryJobsFilter{activity: industryActivityReaction}, reaction, true},
		{"reactions but manufacturing", industryJobsFilter{activity: industryActivityReaction}, r, false},
		{"all active matches ready", industryJobsFilter{status: industryStatusActive}, r, true},
		{"all active but delivered", industryJobsFilter{status: industryStatusActive}, delivered, false},
		{"ready matches", industryJobsFilter{status: industryStatusReady}, r, true},
		{"halted matches", industryJobsFilter{status: industryStatusHalted}, paused, true},
		{"halted but ready", industryJobsFilter{status: industryStatusHalted}, r, false},
		{"history matches delivered", industryJobsFilter{status: industryStatusHistory}, delivered, true},
		{"history but ready", industryJobsFilter{status: industryStatusHistory}, r, false},
		{"one of many differs", industryJobsFilter{owner: industryOwnerMe, activity: industryActivityCopying}, r, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestIndustryJobs_FilterWidgets(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []industryJobRow{
		{jobID: 1, activity: app.Manufacturing, isInstallerMe: true, status: app.JobReady},
		{jobID: 2, activity: app.Copying, isInstallerMe: true, status: app.JobReady},
		{jobID: 3, activity: app.Manufacturing, isInstallerMe: false, status: app.JobReady},
		{jobID: 4, activity: app.Manufacturing, isInstallerMe: true, status: app.JobDelivered},
	}
	newIndustryJobs := func(t *testing.T, isMobile, forCorporation bool) *IndustryJobs {
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		})
		var a *IndustryJobs
		if forCorporation {
			a = NewJobsForCorporation(u)
		} else {
			a = NewJobsForOverview(u)
		}
		a.rows = rows
		a.filterRowsAsync("")
		return a
	}
	jobIDs := func(a *IndustryJobs) []int64 {
		return xslices.Map(a.rowsFiltered, func(r industryJobRow) int64 {
			return r.jobID
		})
	}
	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("shows only jobs installed by me outside corporation mode mobile=%v", isMobile), func(t *testing.T) {
			a := newIndustryJobs(t, isMobile, false)
			assert.ElementsMatch(t, []int64{1, 2}, jobIDs(a))
		})
		t.Run(fmt.Sprintf("shows jobs of all installers in corporation mode mobile=%v", isMobile), func(t *testing.T) {
			a := newIndustryJobs(t, isMobile, true)
			assert.ElementsMatch(t, []int64{1, 2, 3}, jobIDs(a))
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newIndustryJobs(t, true, false)
		a.filterChip.SetSelected(map[string]string{industryJobsFilterActivity: industryActivityCopying})
		assert.ElementsMatch(t, []int64{2}, jobIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newIndustryJobs(t, false, false)
		a.selectActivity.SetSelected(industryActivityCopying)
		assert.ElementsMatch(t, []int64{2}, jobIDs(a))
	})
	t.Run("can combine status and filter on mobile", func(t *testing.T) {
		a := newIndustryJobs(t, true, false)
		a.selectStatus.SetSelected(industryStatusHistory)
		a.filterChip.SetSelected(map[string]string{industryJobsFilterActivity: industryActivityManufacturing})
		assert.ElementsMatch(t, []int64{4}, jobIDs(a))
	})
	t.Run("shows tag filter outside corporation mode on mobile", func(t *testing.T) {
		a := newIndustryJobs(t, true, false)
		assert.Equal(t, map[string]string{
			industryJobsFilterActivity: "",
			industryJobsFilterOwner:    "",
			industryJobsFilterTag:      "",
		}, a.filterChip.Selected())
	})
	t.Run("shows installer filter in corporation mode on mobile", func(t *testing.T) {
		a := newIndustryJobs(t, true, true)
		assert.Equal(t, map[string]string{
			industryJobsFilterActivity:  "",
			industryJobsFilterInstaller: "",
			industryJobsFilterOwner:     "",
		}, a.filterChip.Selected())
	})
	t.Run("resets filters when corporation changes on mobile", func(t *testing.T) {
		a := newIndustryJobs(t, true, true)
		a.filterChip.SetSelected(map[string]string{industryJobsFilterActivity: industryActivityCopying})
		require.True(t, a.filterChip.IsOn())

		a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), factory.CreateCorporation())

		assert.False(t, a.filterChip.IsOn())
	})
	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("resets status to default when corporation changes mobile=%v", isMobile), func(t *testing.T) {
			a := newIndustryJobs(t, isMobile, true)
			a.selectStatus.SetSelected(industryStatusHistory)

			a.u.Signals().CurrentCorporationExchanged.Emit(t.Context(), factory.CreateCorporation())

			assert.Equal(t, industryStatusActive, a.selectStatus.Selected)
		})
	}
}
