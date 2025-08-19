package ui

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestCorporationWallet_CanRenderWithData(t *testing.T) {
	ctx := context.Background()
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	test.ApplyTheme(t, test.Theme())
	ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
	const d = app.Division1
	a := ui.corporationWallets[d]
	w := test.NewWindow(a)
	defer w.Close()
	w.Resize(fyne.NewSize(1700, 300))

	const balance = 12_345_678.99
	c := factory.CreateCorporation()
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		CorporationID: c.ID,
	})
	factory.CreateCharacter(storage.CreateCharacterParams{
		ID: ec.ID,
	})
	factory.SetCharacterRoles(ec.ID, set.Of(app.RoleAccountant))
	factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
		Amount:        2_345_67.89,
		Balance:       balance,
		CorporationID: c.ID,
		Description:   "Test entry",
		Date:          time.Date(2017, 8, 16, 10, 8, 0, 0, time.UTC),
		DivisionID:    d.ID(),
	})
	factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
		CorporationID: c.ID,
		Section:       app.CorporationSectionWalletJournal(d),
	})
	factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{
		CorporationID: c.ID,
		DivisionID:    d.ID(),
		Balance:       balance,
	})
	factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
		CorporationID: c.ID,
		Section:       app.SectionCorporationWalletBalances,
	})
	ui.setCorporation(c)
	err := ui.scs.InitCache(ctx)
	if err != nil {
		t.Fatal(err)
	}
	a.update()
	test.AssertImageMatches(t, "corporationwallets/master.png", w.Canvas().Capture())
}
