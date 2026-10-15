package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestSkillQueue_RefreshTicker(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	a := NewQueueWithCharacter(u, factory.CreateCharacter())

	u.Signals().RefreshTickerExpired.Emit(t.Context(), struct{}{})

	assert.Equal(t, "Waiting for character data to be loaded...", a.top.Text)
}

func TestSkillQueue_Update(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	c := factory.CreateCharacterFull()
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: c.ID,
		Section:     app.SectionCharacterSkillqueue,
	})
	factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
	factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
	a := NewQueueWithCharacter(u, c)

	a.Update(t.Context())

	assert.Equal(t, 2, a.list.Length())
	assert.False(t, a.emptyInfo.Visible())
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
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		c := factory.CreateCharacterFull()
		return NewQueueWithCharacter(u, c), u, c
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
}
