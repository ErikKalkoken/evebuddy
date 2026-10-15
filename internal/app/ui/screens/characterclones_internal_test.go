package screens

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestCharacterClones_Update(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	c := factory.CreateCharacterFull()
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: c.ID,
		Section:     app.SectionCharacterJumpClones,
	})
	factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{CharacterID: c.ID})
	factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{CharacterID: c.ID})
	newClones := func(t *testing.T) *CharacterClones {
		a := NewCharacterClones(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		a.character.Store(c)
		return a
	}
	t.Run("shows clone count in footer and jump info in top", func(t *testing.T) {
		a := newClones(t)

		a.update(t.Context())

		assert.Equal(t, "2 clones", a.footerLabel.Text)
		assert.Contains(t, a.topText.String(), "Next available jump:")
		assert.NotContains(t, a.topText.String(), "clones")
	})
	t.Run("hides error after successful update", func(t *testing.T) {
		a := newClones(t)
		a.errorLabel.SetText("ERROR: failed")
		a.errorLabel.Show()

		a.update(t.Context())

		assert.False(t, a.errorLabel.Visible())
	})
	t.Run("shows error when update fails and keeps it on refresh", func(t *testing.T) {
		a := newClones(t)
		a.update(t.Context())
		if _, err := db.Exec("DROP TABLE character_jump_clone_implants"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")
		assert.Equal(t, "2 clones", a.footerLabel.Text, "keeps last count")

		a.refreshTop(t.Context())
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.topText.String(), "Next available jump:")
	})
}

func TestCharacterClones_RefreshTop(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	newClones := func(t *testing.T) *CharacterClones {
		return NewCharacterClones(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
	}
	t.Run("shows jump available once cooldown is over", func(t *testing.T) {
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{
			LastCloneJumpAt: optional.New(time.Now().Add(-48 * time.Hour)),
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterJumpClones,
		})
		stale := *c
		stale.NextCloneJump = optional.New(time.Now().Add(time.Hour)) // computed before the cooldown ended
		a := newClones(t)
		a.character.Store(&stale)

		a.refreshTop(t.Context())

		assert.Contains(t, a.topText.String(), "Next available jump: NOW")
	})
	t.Run("shows no character", func(t *testing.T) {
		a := newClones(t)
		a.footerLabel.SetText("2 clones")

		a.update(t.Context())

		assert.Equal(t, "No character", a.topText.String())
		assert.Equal(t, "", a.footerLabel.Text)
	})
}
