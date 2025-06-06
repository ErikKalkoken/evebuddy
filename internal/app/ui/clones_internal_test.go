package ui

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestClones(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t.TempDir())
	defer db.Close()
	bu := NewFakeBaseUI(st, test.NewTempApp(t))
	t.Run("can handle empty clone location", func(t *testing.T) {
		character := factory.CreateCharacter()
		factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: character.ID,
		})
		location := factory.CreateEveLocationEmptyStructure()
		factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{
			CharacterID: character.ID,
			LocationID:  location.ID,
		})
		bu.clones.update()
	})
}
