package industry

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/stretchr/testify/assert"

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
	a.Update(t.Context())

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
		a.Update(t.Context())
		a.corporation.Store(corporation)
		xx, err := a.fetchCombinedJobs(t.Context())
		if !assert.NoError(t, err) {
			t.Fatal()
		}
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
		a.Update(t.Context())

		xx, err := a.fetchCorporationJobs(t.Context())
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		want := set.Of(j2.JobID, j3.JobID)
		got := set.Collect(xiter.MapSlice(xx, func(x industryJobRow) int64 {
			return x.jobID
		}))
		xassert.Equal(t, want, got)
	})
}
