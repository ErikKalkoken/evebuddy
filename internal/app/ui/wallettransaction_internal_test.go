package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func TestWalletTransactions_CanRenderWithData(t *testing.T) {
	test.ApplyTheme(t, test.Theme())
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	t.Run("can show transactions for characters", func(t *testing.T) {
		testutil.TruncateTables(db)
		character := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: character.ID,
			Section:     app.SectionCharacterWalletTransactions,
		})
		et := factory.CreateEveType(storage.CreateEveTypeParams{
			Name: "Merlin",
		})
		client := factory.CreateEveEntityCharacter(app.EveEntity{
			Name: "Peter Paker",
		})
		system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{
			Name:           "Abune",
			SecurityStatus: 0.4,
		})
		location := factory.CreateEveLocationStation(storage.UpdateOrCreateLocationParams{
			Name:          "My Home",
			SolarSystemID: optional.New(system.ID),
		})
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{
			CharacterID: character.ID,
			EveTypeID:   et.ID,
			Quantity:    42,
			Date:        time.Date(2025, 7, 19, 12, 0, 0, 0, time.UTC),
			UnitPrice:   1234.56,
			ClientID:    client.ID,
			LocationID:  location.ID,
		})
		factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{
			CharacterID: character.ID,
			EveTypeID:   et.ID,
			Quantity:    3,
			Date:        time.Date(2025, 7, 19, 13, 0, 0, 0, time.UTC),
			UnitPrice:   2345.67,
			ClientID:    client.ID,
			LocationID:  location.ID,
			IsBuy:       true,
		})
		ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
		x := newCharacterWalletTransaction(ui)
		w := test.NewWindow(x)
		defer w.Close()
		w.Resize(fyne.NewSize(1700, 300))

		ui.setCharacter(character)
		x.update()

		test.AssertImageMatches(t, "wallettransactions/character.png", w.Canvas().Capture())
	})
	t.Run("can show transactions for corporations", func(t *testing.T) {
		testutil.TruncateTables(db)
		character := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			CharacterID: character.ID,
			Scopes:      app.Scopes(),
		})
		factory.SetCharacterRoles(character.ID, app.SectionCorporationWalletTransactions1.Roles())
		ec := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
			ID: character.EveCharacter.Corporation.ID,
		})
		corporation := factory.CreateCorporation(ec.ID)
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: corporation.ID,
			Section:       app.SectionCorporationWalletTransactions1,
		})
		et := factory.CreateEveType(storage.CreateEveTypeParams{
			Name: "Merlin",
		})
		client := factory.CreateEveEntityCharacter(app.EveEntity{
			Name: "Peter Paker",
		})
		system := factory.CreateEveSolarSystem(storage.CreateEveSolarSystemParams{
			Name:           "Abune",
			SecurityStatus: 0.4,
		})
		location := factory.CreateEveLocationStation(storage.UpdateOrCreateLocationParams{
			Name:          "My Home",
			SolarSystemID: optional.New(system.ID),
		})
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: corporation.ID,
			EveTypeID:     et.ID,
			Quantity:      42,
			Date:          time.Date(2025, 7, 19, 12, 0, 0, 0, time.UTC),
			UnitPrice:     1234.56,
			ClientID:      client.ID,
			LocationID:    location.ID,
		})
		factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
			CorporationID: corporation.ID,
			EveTypeID:     et.ID,
			Quantity:      3,
			Date:          time.Date(2025, 7, 19, 13, 0, 0, 0, time.UTC),
			UnitPrice:     2345.67,
			ClientID:      client.ID,
			LocationID:    location.ID,
			IsBuy:         true,
		})
		ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
		x := newCorporationWalletTransactions(ui, app.Division1)
		w := test.NewWindow(x)
		defer w.Close()
		w.Resize(fyne.NewSize(1700, 300))

		ui.setCorporation(corporation)
		x.update()

		// fmt.Println(testutil.DumpTables(db))
		test.AssertImageMatches(t, "wallettransactions/corporation.png", w.Canvas().Capture())
	})
}
