package screens

import (
	"bytes"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestTraining_Filter(t *testing.T) {
	t.Skip("Temporarily disabled as they are now flaky with filtering running async") // TODO
	now := time.Now().UTC()
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Alpha",
	})
	character1 := factory.CreateCharacter(storage.CreateCharacterParams{
		ID:            ec1.ID,
		TotalSP:       optional.New(10_000_000),
		UnallocatedSP: optional.New(1_000_000),
	})
	factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
		CharacterID: character1.ID,
		StartDate:   optional.New(now.Add(-1 * time.Hour)),
		FinishDate:  optional.New(now.Add(3 * time.Hour)),
	})
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character1.ID,
		Section:     app.SectionCharacterSkillqueue,
		CompletedAt: now,
	})

	ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bravo",
	})
	character2 := factory.CreateCharacter(storage.CreateCharacterParams{
		ID:            ec2.ID,
		TotalSP:       optional.New(10_000_000),
		UnallocatedSP: optional.New(1_000_000),
	})
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character2.ID,
		Section:     app.SectionCharacterSkillqueue,
		CompletedAt: now,
	})
	tag := factory.CreateCharacterTag()
	factory.AddCharacterToTag(tag, character2)
	factory.CreateCharacterTag()
	a := NewTraining(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	}))
	a.update(t.Context())

	t.Run("no filter", func(t *testing.T) {
		a.selectStatus.SetSelected("")
		a.selectTag.SetSelected("")
		got := xslices.Map(a.rowsFiltered, func(r trainingRow) string {
			return r.characterName
		})
		want := []string{"Alpha", "Bravo"}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("filter active", func(t *testing.T) {
		a.selectStatus.SetSelected(trainingStatusActive)
		a.selectTag.SetSelected("")

		got := xslices.Map(a.rowsFiltered, func(r trainingRow) string {
			return r.characterName
		})
		want := []string{"Alpha"}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("filter inactive", func(t *testing.T) {
		a.selectStatus.SetSelected(trainingStatusInActive)
		a.selectTag.SetSelected("")

		got := xslices.Map(a.rowsFiltered, func(r trainingRow) string {
			return r.characterName
		})
		want := []string{"Bravo"}
		assert.ElementsMatch(t, want, got)
	})
	t.Run("filter tag", func(t *testing.T) {
		a.selectStatus.SetSelected("")
		a.selectTag.SetSelected(tag.Name)

		got := xslices.Map(a.rowsFiltered, func(r trainingRow) string {
			return r.characterName
		})
		want := []string{"Bravo"}
		assert.ElementsMatch(t, want, got)
	})
}

func TestTraining_MakeTrainingCSVString(t *testing.T) {
	a := &Training{
		rowsFiltered: []trainingRow{
			{
				characterName:              "Alpha",
				tags:                       set.Of("pilot", "logistics"),
				skillName:                  "Gunnery V",
				isActive:                   false,
				totalRemainingCountDisplay: "3",
				trainedSPDisplay:           "10000000",
				unallocatedSPDisplay:       "1000000",
				totalSPDisplay:             "11000000",
			},
			{
				characterName:              "Bravo",
				tags:                       set.Of[string](),
				skillName:                  "N/A",
				isActive:                   false,
				totalRemainingCountDisplay: "",
				trainedSPDisplay:           "5000000",
				unallocatedSPDisplay:       "0",
				totalSPDisplay:             "5000000",
			},
		},
	}
	var b bytes.Buffer
	err := writeTrainingRowsToCSV(&b, a.rowsFiltered)
	assert.NoError(t, err)
	got := b.String()
	want := "Character,Tags,Current Skill,Current Remaining,Queued,Queue Time,Trained SP,Unallocated SP,Total SP\n" +
		"Alpha,logistics;pilot,Gunnery V,N/A,3,N/A,10000000,1000000,11000000\n" +
		"Bravo,,N/A,N/A,,N/A,5000000,0,5000000\n"
	assert.Equal(t, want, got)
}

func TestTrainingFilter_Match(t *testing.T) {
	r := trainingRow{isActive: true, tags: set.Of("alpha")}
	inactive := r
	inactive.isActive = false
	for _, tc := range []struct {
		name   string
		filter trainingFilter
		row    trainingRow
		want   bool
	}{
		{"no filter", trainingFilter{}, r, true},
		{"active matches", trainingFilter{status: trainingStatusActive}, r, true},
		{"active but inactive", trainingFilter{status: trainingStatusActive}, inactive, false},
		{"inactive matches", trainingFilter{status: trainingStatusInActive}, inactive, true},
		{"inactive but active", trainingFilter{status: trainingStatusInActive}, r, false},
		{"tag matches", trainingFilter{tag: "alpha"}, r, true},
		{"tag missing", trainingFilter{tag: "bravo"}, r, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestTraining_FilterWidgets(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []trainingRow{
		{characterID: 1, isActive: true, searchTarget: "bruce"},
		{characterID: 2, isActive: false, searchTarget: "alice"},
	}
	newTraining := func(t *testing.T, isMobile bool) *Training {
		a := NewTraining(testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		}))
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2)
		return a
	}
	characterIDs := func(a *Training) []int64 {
		return xslices.Map(a.rowsFiltered, func(r trainingRow) int64 {
			return r.characterID
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newTraining(t, true)
		a.filterChip.SetSelected(map[string]string{trainingFilterStatus: trainingStatusInActive})
		assert.ElementsMatch(t, []int64{2}, characterIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newTraining(t, false)
		a.selectStatus.SetSelected(trainingStatusInActive)
		assert.ElementsMatch(t, []int64{2}, characterIDs(a))
	})
	t.Run("can search on mobile", func(t *testing.T) {
		a := newTraining(t, true)
		a.searchEntry.SetText("bru")
		assert.ElementsMatch(t, []int64{1}, characterIDs(a))
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newTraining(t, true)
		assert.Equal(t, map[string]string{
			trainingFilterStatus: "",
			trainingFilterTag:    "",
		}, a.filterChip.Selected())
	})
}
