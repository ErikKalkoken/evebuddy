package screens

import (
	"bytes"
	"database/sql"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestTraining_Filter(t *testing.T) {
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

func TestTraining_CharacterRemoved(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	NewTraining(u)
	removed := factory.CreateCharacterFull()
	other := factory.CreateCharacterFull()
	u.GetOrCreateWindow(trainingQueueWindowID(removed.ID))
	u.GetOrCreateWindow(trainingQueueWindowID(other.ID))

	u.Signals().CharacterRemoved.Emit(t.Context(), &app.EntityShort{ID: removed.ID})

	_, created := u.GetOrCreateWindow(trainingQueueWindowID(removed.ID))
	assert.True(t, created, "queue window of removed character is closed")
	_, created = u.GetOrCreateWindow(trainingQueueWindowID(other.ID))
	assert.False(t, created, "queue window of other character stays open")
}

func TestTraining_ErrorDisplay(t *testing.T) {
	newScreen := func(t *testing.T) (*Training, *sql.DB) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		a := NewTraining(u)
		return a, db
	}
	t.Run("shows error and clears rows when update fails and keeps error on filtering", func(t *testing.T) {
		a, db := newScreen(t)
		a.rows = []trainingRow{{isWatched: true}}
		a.filterRowsAsync("")
		expired := optional.New(-1)
		a.OnUpdate = func(n optional.Optional[int]) {
			expired = n
		}
		if _, err := db.Exec("DROP TABLE characters"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")
		assert.Empty(t, a.rows)
		assert.Empty(t, a.rowsFiltered)
		assert.True(t, expired.IsEmpty())

		a.filterRowsAsync("")
		assert.True(t, a.errorLabel.Visible())
	})
	t.Run("hides error after successful update", func(t *testing.T) {
		a, _ := newScreen(t)
		a.errorLabel.SetText("ERROR: failed")
		a.errorLabel.Show()

		a.update(t.Context())
		assert.False(t, a.errorLabel.Visible())
	})
}

func TestTrainingRow_Status(t *testing.T) {
	cases := []struct {
		name           string
		row            trainingRow
		wantText       string
		wantImportance widget.Importance
	}{
		{"active", trainingRow{isActive: true, isWatched: true}, "Active", widget.SuccessImportance},
		{"expired", trainingRow{isWatched: true}, "Expired", widget.DangerImportance},
		{"inactive", trainingRow{}, "Inactive", widget.LowImportance},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			text, importance := tc.row.status()
			// then
			xassert.Equal(t, tc.wantText, text)
			xassert.Equal(t, tc.wantImportance, importance)
		})
	}
}

func TestTimeUntil(t *testing.T) {
	t.Run("returns empty without time", func(t *testing.T) {
		// when
		got := timeUntil(optional.Optional[time.Time]{})
		// then
		xassert.Empty(t, got)
	})
	t.Run("returns zero for time in the past", func(t *testing.T) {
		// when
		got := timeUntil(optional.New(time.Now().Add(-time.Hour)))
		// then
		xassert.EqualOptional(t, 0, got)
	})
	t.Run("returns duration until time in the future", func(t *testing.T) {
		// when
		got := timeUntil(optional.New(time.Now().Add(time.Hour)))
		// then
		xassert.EqualDuration(t, time.Hour, got.ValueOrZero(), time.Minute)
	})
}

func TestTrainingRow_RemainingTimeString(t *testing.T) {
	t.Run("shows not applicable when inactive", func(t *testing.T) {
		// given
		r := trainingRow{}
		// when
		got := r.remainingTimeString(optional.New(time.Hour))
		// then
		xassert.Equal(t, "N/A", got)
	})
	t.Run("shows question mark when time is unknown", func(t *testing.T) {
		// given
		r := trainingRow{isActive: true}
		// when
		got := r.remainingTimeString(optional.Optional[time.Duration]{})
		// then
		xassert.Equal(t, "?", got)
	})
	t.Run("shows remaining time", func(t *testing.T) {
		// given
		r := trainingRow{isActive: true}
		// when
		got := r.remainingTimeString(optional.New(2 * time.Hour))
		// then
		xassert.Equal(t, ihumanize.Duration(2*time.Hour), got)
	})
}

func TestTraining_Update(t *testing.T) {
	t.Run("shows training of characters", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewTraining(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		training := factory.CreateCharacterFull()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID:   training.ID,
			EveTypeID:     factory.CreateEveType(storage.CreateEveTypeParams{Name: "Gunnery"}).ID,
			FinishedLevel: 3,
			StartDate:     optional.New(time.Now().Add(-time.Hour)),
			FinishDate:    optional.New(time.Now().Add(time.Hour)),
		})
		idle := factory.CreateCharacterFull()
		// when
		a.u.Signals().AppInit.Emit(t.Context(), struct{}{})
		// then
		got := make(map[int64]string)
		for _, r := range a.rows {
			s, _ := r.status()
			got[r.characterID] = s + ": " + r.skillName
		}
		want := map[int64]string{
			training.ID: "Active: Gunnery III",
			idle.ID:     "Inactive: N/A",
		}
		xassert.Equal(t, want, got)
	})
	t.Run("reports characters with expired training", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewTraining(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[int]
		a.OnUpdate = func(n optional.Optional[int]) {
			reported = append(reported, n)
		}
		factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
		factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
		factory.CreateCharacterFull()
		// when
		a.u.Signals().AppInit.Emit(t.Context(), struct{}{})
		// then
		xassert.Equal(t, []optional.Optional[int]{optional.New(2)}, reported)
	})
}

func TestTraining_UpdateItem(t *testing.T) {
	t.Run("replaces row of character when skill queue changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewTraining(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		var reported []optional.Optional[int]
		a.OnUpdate = func(n optional.Optional[int]) {
			reported = append(reported, n)
		}
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
		a.rows = []trainingRow{
			{characterID: c.ID, isActive: true, skillName: "old"},
			{characterID: 42, isActive: true, skillName: "other"},
		}
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkillqueue,
		})
		// then
		got := xslices.Map(a.rows, func(r trainingRow) string {
			return r.skillName
		})
		xassert.Equal(t, []string{"N/A", "other"}, got)
		xassert.Equal(t, []optional.Optional[int]{optional.New(1)}, reported)
	})
	t.Run("replaces row of character when character changed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewTraining(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		a.rows = []trainingRow{{characterID: c.ID, skillName: "old"}}
		// when
		a.u.Signals().CharacterChanged.Emit(t.Context(), c.ID)
		// then
		require.Len(t, a.rows, 1)
		xassert.Equal(t, "N/A", a.rows[0].skillName)
	})
	t.Run("ignores character not in list", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewTraining(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		a.rows = []trainingRow{{characterID: 42, skillName: "other"}}
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkillqueue,
		})
		// then
		xassert.Equal(t, []trainingRow{{characterID: 42, skillName: "other"}}, a.rows)
	})
	t.Run("ignores change of other section", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewTraining(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		a.rows = []trainingRow{{characterID: c.ID, skillName: "old"}}
		// when
		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterImplants,
		})
		// then
		require.Len(t, a.rows, 1)
		xassert.Equal(t, "old", a.rows[0].skillName)
	})
}

func TestTraining_ShowTrainingQueueWindow(t *testing.T) {
	t.Run("can show queue of character", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewTraining(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		// when
		a.showTrainingQueueWindow(trainingRow{characterID: c.ID, characterName: "Bruce"})
		// then
		w, created := a.u.GetOrCreateWindow(trainingQueueWindowID(c.ID))
		assert.False(t, created)
		assert.NotNil(t, w.Content())
	})
	t.Run("closes window when character can not be loaded", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewTraining(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		// when
		a.showTrainingQueueWindow(trainingRow{characterID: 42, characterName: "Bruce"})
		// then
		_, created := a.u.GetOrCreateWindow(trainingQueueWindowID(42))
		assert.True(t, created)
	})
	t.Run("stops queue when window is closed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewTraining(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		c := factory.CreateCharacterFull()
		n := a.u.Signals().RefreshTickerExpired.Len()
		a.showTrainingQueueWindow(trainingRow{characterID: c.ID, characterName: "Bruce"})
		require.Equal(t, n+1, a.u.Signals().RefreshTickerExpired.Len())
		w, _ := a.u.GetOrCreateWindow(trainingQueueWindowID(c.ID))
		// when
		w.Close()
		// then
		xassert.Equal(t, n, a.u.Signals().RefreshTickerExpired.Len())
		_, created := a.u.GetOrCreateWindow(trainingQueueWindowID(c.ID))
		assert.True(t, created)
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := NewTraining(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		w, _ := a.u.GetOrCreateWindow(trainingQueueWindowID(42))
		// when
		a.showTrainingQueueWindow(trainingRow{characterID: 42, characterName: "Bruce"})
		// then
		assert.Nil(t, w.Content())
	})
}
