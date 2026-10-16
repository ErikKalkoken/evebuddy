package screens

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// skillQueueStatus records the last status reported via OnUpdate.
type skillQueueStatus struct {
	short, long string
}

func newSkillQueueForTest(t *testing.T, st *storage.Storage, c *app.Character) (*SkillQueue, *testdouble.UIFake, *skillQueueStatus) {
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	var a *SkillQueue
	if c == nil {
		a = NewSkillQueue(u)
	} else {
		a = NewQueueWithCharacter(u, c)
	}
	status := &skillQueueStatus{short: "42%", long: "Skill (42%)"} // status of a previous character
	a.OnUpdate = func(statusShort, statusLong string) {
		status.short, status.long = statusShort, statusLong
	}
	return a, u, status
}

func createSkillqueueSection(factory testutil.Factory, characterID int64) {
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: characterID,
		Section:     app.SectionCharacterSkillqueue,
	})
}

// createActiveSkillqueue creates a queue with one skill 25% into training and one more queued.
func createActiveSkillqueue(factory testutil.Factory, characterID int64) {
	now := time.Now()
	factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
		CharacterID:     characterID,
		EveTypeID:       factory.CreateEveType(storage.CreateEveTypeParams{Name: "Alpha"}).ID,
		FinishedLevel:   3,
		StartDate:       optional.New(now.Add(-1 * time.Hour)),
		FinishDate:      optional.New(now.Add(3 * time.Hour)),
		LevelStartSP:    optional.New[int64](0),
		TrainingStartSP: optional.New[int64](0),
		LevelEndSP:      optional.New[int64](1000),
	})
	factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
		CharacterID: characterID,
		StartDate:   optional.New(now.Add(3 * time.Hour)),
		FinishDate:  optional.New(now.Add(10 * time.Hour)),
	})
}

func TestSkillQueue_Update(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	t.Run("no character", func(t *testing.T) {
		a, _, status := newSkillQueueForTest(t, st, nil)

		a.Update(t.Context())

		assert.Equal(t, "No character", a.topLabel.Text)
		assert.Equal(t, widget.LowImportance, a.topLabel.Importance)
		assert.False(t, a.skillList.Visible())
		assert.False(t, a.emptyLabel.Visible())
		assert.False(t, a.statusIcon.Visible())
		assert.Equal(t, skillQueueStatus{}, *status)
	})
	t.Run("section not loaded", func(t *testing.T) {
		a, _, status := newSkillQueueForTest(t, st, factory.CreateCharacterFull())

		a.Update(t.Context())

		assert.Equal(t, "Waiting for character data to be loaded...", a.topLabel.Text)
		assert.Equal(t, widget.WarningImportance, a.topLabel.Importance)
		assert.False(t, a.skillList.Visible())
		assert.False(t, a.emptyLabel.Visible())
		assert.False(t, a.statusIcon.Visible())
		assert.Equal(t, skillQueueStatus{}, *status)
	})
	t.Run("empty queue and training not watched", func(t *testing.T) {
		c := factory.CreateCharacterFull()
		createSkillqueueSection(factory, c.ID)
		a, _, status := newSkillQueueForTest(t, st, c)

		a.Update(t.Context())

		assert.Equal(t, "Training not active", a.topLabel.Text)
		assert.Equal(t, widget.MediumImportance, a.topLabel.Importance)
		assert.Equal(t, 0, a.skillList.Length())
		assert.True(t, a.emptyLabel.Visible())
		assert.True(t, a.statusIcon.Visible())
		assert.Equal(t, skillQueueStatus{}, *status)
	})
	t.Run("empty queue and training watched", func(t *testing.T) {
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
		createSkillqueueSection(factory, c.ID)
		a, _, status := newSkillQueueForTest(t, st, c)

		a.Update(t.Context())

		assert.Equal(t, "No skill in training", a.topLabel.Text)
		assert.Equal(t, widget.DangerImportance, a.topLabel.Importance)
		assert.True(t, a.emptyLabel.Visible())
		assert.Equal(t, skillQueueStatus{"!", "No skill in training"}, *status)
	})
	t.Run("active training", func(t *testing.T) {
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
		createSkillqueueSection(factory, c.ID)
		createActiveSkillqueue(factory, c.ID)
		a, _, status := newSkillQueueForTest(t, st, c)

		a.Update(t.Context())

		assert.Contains(t, a.topLabel.Text, "Total training time:")
		assert.Equal(t, widget.MediumImportance, a.topLabel.Importance)
		assert.Equal(t, 2, a.skillList.Length())
		assert.True(t, a.skillList.Visible())
		assert.False(t, a.emptyLabel.Visible())
		assert.True(t, a.statusIcon.Visible())
		assert.Equal(t, skillQueueStatus{"25%", "Alpha III (25%)"}, *status)
	})
	t.Run("finished queue and training watched", func(t *testing.T) {
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
		createSkillqueueSection(factory, c.ID)
		now := time.Now()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   optional.New(now.Add(-5 * time.Hour)),
			FinishDate:  optional.New(now.Add(-1 * time.Hour)),
		})
		a, _, status := newSkillQueueForTest(t, st, c)

		a.Update(t.Context())

		assert.Equal(t, "No skill in training", a.topLabel.Text)
		assert.Equal(t, widget.DangerImportance, a.topLabel.Importance)
		assert.Equal(t, 1, a.skillList.Length())
		assert.False(t, a.emptyLabel.Visible())
		assert.Equal(t, skillQueueStatus{"!", "No skill in training"}, *status)
	})
}

func TestSkillQueue_CharacterSectionChanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	setup := func(t *testing.T) (*SkillQueue, *testdouble.UIFake, *app.Character) {
		c := factory.CreateCharacterFull()
		createSkillqueueSection(factory, c.ID)
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		a, u, _ := newSkillQueueForTest(t, st, c)
		a.Update(t.Context())
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		return a, u, c
	}
	t.Run("reloads on skill queue update for the character", func(t *testing.T) {
		a, u, c := setup(t)

		u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkillqueue,
		})

		assert.Equal(t, 2, a.skillList.Length())
	})
	t.Run("ignores other sections", func(t *testing.T) {
		a, u, c := setup(t)

		u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAssets,
		})

		assert.Equal(t, 1, a.skillList.Length())
	})
	t.Run("ignores other characters", func(t *testing.T) {
		a, u, _ := setup(t)
		other := factory.CreateCharacterFull()

		u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: other.ID,
			Section:     app.SectionCharacterSkillqueue,
		})

		assert.Equal(t, 1, a.skillList.Length())
	})
}

func TestSkillQueue_CurrentCharacterExchanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	c := factory.CreateCharacterFull()
	createSkillqueueSection(factory, c.ID)
	createActiveSkillqueue(factory, c.ID)
	t.Run("shows queue of new current character", func(t *testing.T) {
		a, u, _ := newSkillQueueForTest(t, st, nil)

		u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)

		assert.Same(t, c, a.character.Load())
		assert.Equal(t, 2, a.skillList.Length())
	})
	t.Run("queue for fixed character ignores exchange", func(t *testing.T) {
		fixed := factory.CreateCharacterFull()
		a, u, _ := newSkillQueueForTest(t, st, fixed)

		u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)

		assert.Same(t, fixed, a.character.Load())
	})
}

func TestSkillQueue_RefreshTicker(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	t.Run("keeps reset state when section not loaded", func(t *testing.T) {
		a, u, status := newSkillQueueForTest(t, st, factory.CreateCharacterFull())
		a.Update(t.Context())

		u.Signals().RefreshTickerExpired.Emit(t.Context(), struct{}{})

		assert.Equal(t, "Waiting for character data to be loaded...", a.topLabel.Text)
		assert.False(t, a.skillList.Visible())
		assert.False(t, a.statusIcon.Visible())
		assert.Equal(t, skillQueueStatus{}, *status)
	})
	t.Run("does not reload the queue", func(t *testing.T) {
		c := factory.CreateCharacterFull()
		createSkillqueueSection(factory, c.ID)
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		a, u, _ := newSkillQueueForTest(t, st, c)
		a.Update(t.Context())
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})

		u.Signals().RefreshTickerExpired.Emit(t.Context(), struct{}{})

		assert.Equal(t, 1, a.skillList.Length())
	})
	t.Run("keeps time dependent status current", func(t *testing.T) {
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
		createSkillqueueSection(factory, c.ID)
		createActiveSkillqueue(factory, c.ID)
		a, u, status := newSkillQueueForTest(t, st, c)
		a.Update(t.Context())
		assert.Equal(t, skillQueueStatus{"25%", "Alpha III (25%)"}, *status)

		// simulate time passing by moving the loaded items back in time
		shiftQueue := func(d time.Duration) {
			for qi := range a.skillqueue.All() {
				qi.StartDate = optional.New(qi.StartDate.ValueOrZero().Add(-d))
				qi.FinishDate = optional.New(qi.FinishDate.ValueOrZero().Add(-d))
			}
		}
		shiftQueue(2 * time.Hour)
		u.Signals().RefreshTickerExpired.Emit(t.Context(), struct{}{})
		assert.Equal(t, skillQueueStatus{"75%", "Alpha III (75%)"}, *status)
		assert.Contains(t, a.topLabel.Text, "Total training time:")

		shiftQueue(10 * time.Hour)
		u.Signals().RefreshTickerExpired.Emit(t.Context(), struct{}{})
		assert.Equal(t, skillQueueStatus{"!", "No skill in training"}, *status)
		assert.Equal(t, "No skill in training", a.topLabel.Text)
		assert.Equal(t, widget.DangerImportance, a.topLabel.Importance)
	})
	t.Run("keeps error after failed reload", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t) // own DB, since a table is dropped
		defer db.Close()
		c := factory.CreateCharacterFull()
		createSkillqueueSection(factory, c.ID)
		createActiveSkillqueue(factory, c.ID)
		a, u, status := newSkillQueueForTest(t, st, c)
		a.Update(t.Context())
		require.True(t, a.skillList.Visible())
		_, err := db.Exec("DROP TABLE character_skillqueue_items;") // make the reload fail
		require.NoError(t, err)

		u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkillqueue,
		})
		u.Signals().RefreshTickerExpired.Emit(t.Context(), struct{}{})

		assert.Contains(t, a.topLabel.Text, "ERROR:")
		assert.Equal(t, widget.DangerImportance, a.topLabel.Importance)
		assert.False(t, a.skillList.Visible())
		assert.False(t, a.statusIcon.Visible())
		assert.Equal(t, skillQueueStatus{}, *status)
	})
	t.Run("does not show queue again after reset", func(t *testing.T) {
		c := factory.CreateCharacterFull()
		createSkillqueueSection(factory, c.ID)
		createActiveSkillqueue(factory, c.ID)
		a, u, status := newSkillQueueForTest(t, st, nil)
		u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		assert.True(t, a.skillList.Visible())
		u.Signals().CurrentCharacterExchanged.Emit(t.Context(), factory.CreateCharacterFull())

		u.Signals().RefreshTickerExpired.Emit(t.Context(), struct{}{})

		assert.Equal(t, "Waiting for character data to be loaded...", a.topLabel.Text)
		assert.False(t, a.skillList.Visible())
		assert.False(t, a.statusIcon.Visible())
		assert.Equal(t, skillQueueStatus{}, *status)
	})
}

func TestSkillQueue_Stop(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	s := u.Signals()
	listenerCounts := func() []int {
		return []int{
			s.CharacterChanged.Len(),
			s.CharacterSectionChanged.Len(),
			s.CurrentCharacterExchanged.Len(),
			s.RefreshTickerExpired.Len(),
		}
	}
	before := listenerCounts()
	a := NewQueueWithCharacter(u, factory.CreateCharacter())
	assert.NotEqual(t, before, listenerCounts())

	a.Stop()

	assert.Equal(t, before, listenerCounts())
}

func TestSkillQueue_CharacterChanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	newQueue := func(t *testing.T) (*SkillQueue, *testdouble.UIFake, *app.Character) {
		c := factory.CreateCharacterFull()
		a, u, _ := newSkillQueueForTest(t, st, c)
		return a, u, c
	}
	t.Run("reloads the current character", func(t *testing.T) {
		a, u, c := newQueue(t)

		u.Signals().CharacterChanged.Emit(t.Context(), c.ID)

		got := a.character.Load()
		assert.NotSame(t, c, got)
		assert.Equal(t, c.ID, got.ID)
	})
	t.Run("ignores other characters", func(t *testing.T) {
		a, u, c := newQueue(t)
		other := factory.CreateCharacterFull()

		u.Signals().CharacterChanged.Emit(t.Context(), other.ID)

		assert.Same(t, c, a.character.Load())
	})
	t.Run("shows changed training watch status", func(t *testing.T) {
		a, u, c := newQueue(t)
		createSkillqueueSection(factory, c.ID)
		a.Update(t.Context())
		assert.Equal(t, "Training not active", a.topLabel.Text)
		err := st.UpdateCharacterIsTrainingWatched(t.Context(), c.ID, true)
		require.NoError(t, err)

		u.Signals().CharacterChanged.Emit(t.Context(), c.ID)

		assert.Equal(t, "No skill in training", a.topLabel.Text)
	})
}
