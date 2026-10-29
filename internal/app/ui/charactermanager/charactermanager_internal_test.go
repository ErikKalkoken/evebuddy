package charactermanager

import (
	"context"
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestMain(m *testing.M) {
	// Leftover goroutines from one test can render text concurrently with the next,
	// which is not safe in Fyne's shared text shaper.
	runAsync = func(f func()) { f() }
	os.Exit(m.Run())
}

// uiFake adds the methods this package needs to the shared UI fake.
type uiFake struct {
	*testdouble.UIFake

	currentCharacter     *app.Character
	isOffline            bool
	setAnyCharacterCalls int
}

func (u *uiFake) CurrentCharacter() *app.Character               { return u.currentCharacter }
func (u *uiFake) IsOffline() bool                                { return u.isOffline }
func (u *uiFake) LoadCharacter(_ context.Context, _ int64) error { return nil }

func (u *uiFake) SetAnyCharacter(_ context.Context) error {
	u.setAnyCharacterCalls++
	return nil
}

func TestShow(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("can render window with data", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		tag := factory.CreateCharacterTag("Alpha")
		c := factory.CreateCharacter()
		factory.AddCharacterToTag(tag, c)
		// when
		Show(u)
		// then
		w, created := u.GetOrCreateWindow("characterWindow")
		assert.False(t, created)
		assert.NotNil(t, w.Canvas().Capture())
		w.Close()
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		Show(u)
		n := len(fyne.CurrentApp().Driver().AllWindows())
		// when
		Show(u)
		// then
		assert.Equal(t, n, len(fyne.CurrentApp().Driver().AllWindows()))
		w, _ := u.GetOrCreateWindow("characterWindow")
		w.Close()
	})
}
