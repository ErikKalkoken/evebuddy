package ui

import (
	"context"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestCorporationMember_CanRenderWithData(t *testing.T) {
	if IsCI() {
		t.Skip("UI tests are currently flaky and therefore only run locally")
	}
	ctx := context.Background()
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	test.ApplyTheme(t, test.Theme())
	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
	a := ui.corporationMember
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
	ui.setCorporation(c)
	err := ui.scs.InitCache(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.update()
	test.AssertImageMatches(t, "corporationmembers/master.png", w.Canvas().Capture())
}
