package corporations

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

func TestCorporationMember_CanRenderWithData(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	test.ApplyTheme(t, test.Theme())
	a := NewMembers(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	}))
	w := test.NewWindow(a)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 300))

	c := factory.CreateCorporation()
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		CorporationID: c.ID,
	})
	factory.CreateCharacter(storage.CreateCharacterParams{
		ID: ec.ID,
	})
	ee := factory.CreateEveEntityCharacter(app.EveEntity{
		Name: "Bruce Wayne",
	})
	factory.CreateCorporationMember(storage.CorporationMemberParams{
		CorporationID: c.ID,
		CharacterID:   ee.ID,
	})
	factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
		CorporationID: c.ID,
		Section:       app.SectionCorporationMembers,
	})
	a.corporation.Store(c)
	a.update(t.Context())
	test.AssertImageMatches(t, "corporationmembers/master.png", w.Canvas().Capture())
}
