package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestIndustryJob_CanRenderWithData(t *testing.T) {
	if testing.Short() {
		t.Skip(SkipUIReason)
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	er := factory.CreateEveRegion(storage.CreateEveRegionParams{Name: "Black Rise"})
	con := factory.CreateEveConstellation(storage.CreateEveConstellationParams{RegionID: er.ID})
	system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{
		SecurityStatus:  0.3,
		ConstellationID: con.ID,
	})
	location := factory.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{
		Name:          "Batcave",
		SolarSystemID: optional.New(system.ID),
	})
	ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
	})
	character1 := factory.CreateCharacter(storage.CreateCharacterParams{
		ID: ec1.ID,
	})
	bp1 := factory.CreateEveType(storage.CreateEveTypeParams{Name: "Merlin Blueprint"})
	factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		CharacterID:     character1.ID,
		BlueprintTypeID: bp1.ID,
		ActivityID:      int32(app.Manufacturing),
		StationID:       location.ID,
		Status:          app.JobReady,
		Runs:            3,
		EndDate:         time.Date(2025, 6, 9, 12, 15, 0, 0, time.UTC),
	})
	ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Clark Kent",
	})
	character2 := factory.CreateCharacter(storage.CreateCharacterParams{
		ID: ec2.ID,
	})
	bp2 := factory.CreateEveType(storage.CreateEveTypeParams{Name: "Caracal Blueprint"})
	factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		CharacterID:     character2.ID,
		BlueprintTypeID: bp2.ID,
		ActivityID:      int32(app.Copying),
		StationID:       location.ID,
		Status:          app.JobReady,
		Runs:            100,
		EndDate:         time.Date(2025, 3, 3, 10, 15, 0, 0, time.UTC),
	})
	cases := []struct {
		name      string
		isDesktop bool
		filename  string
		size      fyne.Size
	}{
		{"desktop", true, "desktop_full", fyne.NewSize(1700, 300)},
		{"mobile", false, "mobile_full", fyne.NewSize(500, 800)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			test.ApplyTheme(t, test.Theme())
			ui := MakeFakeBaseUI(st, test.NewTempApp(t), tc.isDesktop)
			w := test.NewWindow(ui.industryJobs)
			defer w.Close()
			w.Resize(tc.size)

			ui.industryJobs.update()

			test.AssertImageMatches(t, "industryjobs/"+tc.filename+".png", w.Canvas().Capture())
		})
	}
}

func TestIndustryJob_CanRenderEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip(SkipUIReason)
	}
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	cases := []struct {
		name      string
		isDesktop bool
		filename  string
		size      fyne.Size
	}{
		{"desktop", true, "desktop_empty", fyne.NewSize(1700, 300)},
		{"mobile", false, "mobile_empty", fyne.NewSize(500, 800)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			test.ApplyTheme(t, test.Theme())
			ui := MakeFakeBaseUI(st, test.NewTempApp(t), tc.isDesktop)
			w := test.NewWindow(ui.industryJobs)
			defer w.Close()
			w.Resize(tc.size)

			ui.industryJobs.update()

			test.AssertImageMatches(t, "industryjobs/"+tc.filename+".png", w.Canvas().Capture())
		})
	}
}

func TestIndustryJob_Filter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	j1 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		ActivityID: int32(app.Manufacturing),
		Status:     app.JobReady,
	})
	j2 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		ActivityID: int32(app.Copying),
		Status:     app.JobReady,
	})
	j3 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		ActivityID: int32(app.Reactions1),
		Status:     app.JobReady,
	})
	j4 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		ActivityID: int32(app.Reactions2),
		Status:     app.JobReady,
	})
	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
	ui.industryJobs.update()

	t.Run("no filter", func(t *testing.T) {
		ui.industryJobs.selectActivity.SetSelected("")

		got := xslices.Map(ui.industryJobs.rowsFiltered, func(r industryJobRow) int32 {
			return r.jobID
		})
		want := []int32{j1.JobID, j2.JobID, j3.JobID, j4.JobID}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("can filter manufacturing", func(t *testing.T) {
		ui.industryJobs.selectActivity.SetSelected("Manufacturing")

		got := xslices.Map(ui.industryJobs.rowsFiltered, func(r industryJobRow) int32 {
			return r.jobID
		})
		want := []int32{j1.JobID}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("can filter reactions", func(t *testing.T) {
		ui.industryJobs.selectActivity.SetSelected("Reactions")

		got := xslices.Map(ui.industryJobs.rowsFiltered, func(r industryJobRow) int32 {
			return r.jobID
		})
		want := []int32{j3.JobID, j4.JobID}
		assert.ElementsMatch(t, want, got)
	})
}

func TestIndustryJob_FetchJobs(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec := factory.CreateEveCharacter()
	character := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
	corporation := factory.CreateCorporation(ec.Corporation.ID)
	j1 := factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
		ActivityID:  int32(app.Manufacturing),
		CharacterID: character.ID,
		JobID:       1,
		Status:      app.JobReady,
	})
	j2 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
		ActivityID:    int32(app.Copying),
		CorporationID: ec.Corporation.ID,
		JobID:         2,
		Status:        app.JobDelivered,
		InstallerID:   character.ID,
	})
	c2 := factory.CreateEveEntityCharacter()
	j3 := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
		ActivityID:    int32(app.Copying),
		CorporationID: ec.Corporation.ID,
		JobID:         3,
		Status:        app.JobDelivered,
		InstallerID:   c2.ID,
	})
	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)

	t.Run("can return all character and relevant corporation jobs", func(t *testing.T) {
		ui.industryJobs.corporation.Store(corporation)
		xx, err := ui.industryJobs.fetchCombinedJobs()
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		want := set.Of(j1.JobID, j2.JobID)
		got := set.Collect(xiter.MapSlice(xx, func(x industryJobRow) int32 {
			return x.jobID
		}))
		xassert.EqualSet(t, want, got)
	})

	t.Run("can return all jobs for current corporation", func(t *testing.T) {
		ui.corporationIndyJobs.corporation.Store(corporation)
		xx, err := ui.corporationIndyJobs.fetchCorporationJobs()
		if !assert.NoError(t, err) {
			t.Fatal()
		}
		want := set.Of(j2.JobID, j3.JobID)
		got := set.Collect(xiter.MapSlice(xx, func(x industryJobRow) int32 {
			return x.jobID
		}))
		xassert.EqualSet(t, want, got)
	})
}
