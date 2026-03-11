package clones

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

func TestCharacterAugmentations_CanRenderWithData(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUIReason)
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	character := factory.CreateCharacter()
	et := factory.CreateEveType(storage.CreateEveTypeParams{
		Name: "Dummy Implant",
	})
	da := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{ID: app.EveDogmaAttributeImplantSlot})
	factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
		DogmaAttributeID: da.ID,
		EveTypeID:        et.ID,
		Value:            3,
	})
	factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{
		CharacterID: character.ID,
		TypeID:      et.ID,
	})
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character.ID,
		Section:     app.SectionCharacterImplants,
	})
	test.ApplyTheme(t, test.Theme())
	a := NewCharacterAugmentations(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	}))
	w := test.NewWindow(a)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 300))

	a.character.Store(character)
	a.update(t.Context())

	test.AssertImageMatches(t, "characteraugmentations/master.png", w.Canvas().Capture())
}
