package charactermanager

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestTraining_Update(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("shows characters sorted by name", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Bravo"})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Alpha"})
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		// when
		cw.characterTraining.update(context.Background())
		// then
		var got []int64
		for _, c := range cw.characterTraining.characters {
			got = append(got, c.ID)
		}
		assert.Equal(t, []int64{c2.ID, c1.ID}, got)
	})
	t.Run("reports error when loading fails", func(t *testing.T) {
		// given
		db2, st2, _ := testutil.NewDBInMemory()
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st2})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		var messages []string
		cw.displaySnackbar = func(s string) {
			messages = append(messages, s)
		}
		db2.Close()
		// when
		cw.characterTraining.update(context.Background())
		// then
		if assert.Len(t, messages, 1) {
			assert.Contains(t, messages[0], "ERROR: Failed to update training")
		}
	})
	t.Run("skips reporting and updates when canceled", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		var messages []string
		cw.displaySnackbar = func(s string) {
			messages = append(messages, s)
		}
		c := factory.CreateCharacter()
		a := cw.characterTraining
		a.update(context.Background())
		factory.CreateCharacter()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		// when
		a.update(ctx)
		// then
		assert.Empty(t, messages)
		if assert.Len(t, a.characters, 1) {
			assert.Equal(t, c.ID, a.characters[0].ID)
		}
	})
	t.Run("reloads when a character was added", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		c := factory.CreateCharacter()
		// when
		u.Signals().CharacterAdded.Emit(context.Background(), c)
		// then
		if assert.Len(t, cw.characterTraining.characters, 1) {
			assert.Equal(t, c.ID, cw.characterTraining.characters[0].ID)
		}
	})
	t.Run("reloads when a character was removed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		c := factory.CreateCharacter()
		cw.characterTraining.update(context.Background())
		err := st.DeleteCharacter(context.Background(), c.ID)
		require.NoError(t, err)
		// when
		u.Signals().CharacterRemoved.Emit(context.Background(), &app.EntityShort{ID: c.ID})
		// then
		assert.Empty(t, cw.characterTraining.characters)
	})
}

func TestTraining_UpdateCharacterWatched(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("stores flag, updates row and emits CharacterChanged", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		c := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: false})
		a := cw.characterTraining
		a.update(context.Background())
		var changed []int64
		u.Signals().CharacterChanged.AddListener(func(_ context.Context, id int64) {
			changed = append(changed, id)
		})
		// when
		a.updateCharacterWatched(context.Background(), 0, true)
		// then
		c2, err := st.GetCharacter(context.Background(), c.ID)
		require.NoError(t, err)
		assert.True(t, c2.IsTrainingWatched)
		assert.True(t, a.characters[0].IsTrainingWatched)
		assert.Equal(t, []int64{c.ID}, changed)
	})
	t.Run("keeps row and emits nothing when storing fails", func(t *testing.T) {
		// given
		db2, st2, factory2 := testutil.NewDBInMemory()
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st2})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		var messages []string
		cw.displaySnackbar = func(s string) {
			messages = append(messages, s)
		}
		factory2.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: false})
		a := cw.characterTraining
		a.update(context.Background())
		var changed []int64
		u.Signals().CharacterChanged.AddListener(func(_ context.Context, id int64) {
			changed = append(changed, id)
		})
		db2.Close()
		// when
		a.updateCharacterWatched(context.Background(), 0, true)
		// then
		assert.False(t, a.characters[0].IsTrainingWatched)
		assert.Empty(t, changed)
		if assert.Len(t, messages, 1) {
			assert.Contains(t, messages[0], "Failed to update training watcher")
		}
	})
	t.Run("updates the right row when the list was reloaded while storing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Bravo"})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		a := cw.characterTraining
		a.update(context.Background())
		var pending []func()
		orig := runAsync
		runAsync = func(f func()) { pending = append(pending, f) }
		t.Cleanup(func() { runAsync = orig })
		a.updateCharacterWatched(context.Background(), 0, true)
		runAsync = orig
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Alpha"})
		factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		a.update(context.Background()) // Bravo moves to row 1
		// when
		pending[0]()
		// then
		assert.False(t, a.characters[0].IsTrainingWatched) // Alpha
		assert.Equal(t, c1.ID, a.characters[1].ID)
		assert.True(t, a.characters[1].IsTrainingWatched) // Bravo
	})
	t.Run("ignores character removed while storing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Alpha"})
		factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Bravo"})
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		a := cw.characterTraining
		a.update(context.Background())
		var pending []func()
		orig := runAsync
		runAsync = func(f func()) { pending = append(pending, f) }
		t.Cleanup(func() { runAsync = orig })
		a.updateCharacterWatched(context.Background(), 1, true)
		runAsync = orig
		err := st.DeleteCharacter(context.Background(), c2.ID)
		require.NoError(t, err)
		a.update(context.Background()) // only Alpha left
		// when
		pending[0]()
		// then
		assert.Len(t, a.characters, 1)
		assert.False(t, a.characters[0].IsTrainingWatched)
	})
	t.Run("ignores unknown row", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		factory.CreateCharacter()
		a := cw.characterTraining
		a.update(context.Background())
		var changed []int64
		u.Signals().CharacterChanged.AddListener(func(_ context.Context, id int64) {
			changed = append(changed, id)
		})
		// when
		a.updateCharacterWatched(context.Background(), 1, true)
		// then
		assert.False(t, a.characters[0].IsTrainingWatched)
		assert.Empty(t, changed)
	})
	t.Run("selecting a character toggles its flag", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		c := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: true})
		a := cw.characterTraining
		a.update(context.Background())
		// when
		a.list.OnSelected(0)
		// then
		c2, err := st.GetCharacter(context.Background(), c.ID)
		require.NoError(t, err)
		assert.False(t, c2.IsTrainingWatched)
		assert.False(t, a.characters[0].IsTrainingWatched)
	})
}

func TestTraining_SetAllWatched(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("enables all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: false})
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: true})
		a := cw.characterTraining
		a.update(context.Background())
		// when
		a.setAllWatched(context.Background(), true)
		// then
		for _, id := range []int64{c1.ID, c2.ID} {
			c, err := st.GetCharacter(context.Background(), id)
			require.NoError(t, err)
			assert.True(t, c.IsTrainingWatched, "character %d", id)
		}
	})
	t.Run("disables all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: false})
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: true})
		a := cw.characterTraining
		a.update(context.Background())
		// when
		a.setAllWatched(context.Background(), false)
		// then
		for _, id := range []int64{c1.ID, c2.ID} {
			c, err := st.GetCharacter(context.Background(), id)
			require.NoError(t, err)
			assert.False(t, c.IsTrainingWatched, "character %d", id)
		}
	})
	t.Run("skips characters already in target state", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: false})
		factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: true})
		var changed []int64
		u.Signals().CharacterChanged.AddListener(func(_ context.Context, id int64) {
			changed = append(changed, id)
		})
		a := cw.characterTraining
		a.update(context.Background())
		// when
		a.setAllWatched(context.Background(), true)
		// then
		assert.Equal(t, []int64{c1.ID}, changed)
	})
}

func TestTraining_SetWatchedToCurrentlyTrained(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("watches only characters in training", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		trained := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: false})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: trained.ID,
			Section:     app.SectionCharacterSkillqueue,
		})
		now := time.Now()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: trained.ID,
			StartDate:   optional.New(now),
			FinishDate:  optional.New(now.Add(2 * time.Hour)),
		})
		idle := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: true})
		a := cw.characterTraining
		a.update(context.Background())
		// when
		a.setWatchedToCurrentlyTrained(context.Background(), []int64{trained.ID, idle.ID})
		// then
		c1, err := st.GetCharacter(context.Background(), trained.ID)
		require.NoError(t, err)
		assert.True(t, c1.IsTrainingWatched)
		c2, err := st.GetCharacter(context.Background(), idle.ID)
		require.NoError(t, err)
		assert.False(t, c2.IsTrainingWatched)
	})
	t.Run("updates the right row when the list was reloaded", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Bravo"})
		trained := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: trained.ID,
			Section:     app.SectionCharacterSkillqueue,
		})
		now := time.Now()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: trained.ID,
			StartDate:   optional.New(now),
			FinishDate:  optional.New(now.Add(2 * time.Hour)),
		})
		a := cw.characterTraining
		a.update(context.Background())
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Alpha"})
		factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		a.update(context.Background()) // Bravo moves to row 1
		// when
		a.setWatchedToCurrentlyTrained(context.Background(), []int64{trained.ID})
		// then
		assert.False(t, a.characters[0].IsTrainingWatched) // Alpha
		assert.True(t, a.characters[1].IsTrainingWatched)  // Bravo
	})
}
