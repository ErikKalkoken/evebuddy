package ui_test

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestUI_StartEmpty(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterNoResponder(httpmock.NewNotFoundResponder(t.Fatal)) // fails on any HTTP request
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	bu := ui.MakeFakeBaseUI(st, ui.NewFakeApp(t), true)
	u := ui.NewDesktopUI(bu)
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		for {
			<-ticker.C
			if u.IsStartupCompleted() {
				u.App().Quit()
			}
		}
	}()
	u.ShowAndRun()
	assert.Equal(t, 0, httpmock.GetTotalCallCount())
}

func TestUI_StartWithCharacter(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterNoResponder(httpmock.NewNotFoundResponder(t.Fatal)) // fails on any HTTP request
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	character := factory.CreateCharacterFull()
	factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{CharacterID: character.ID})
	factory.CreateCharacterAttributes(storage.UpdateOrCreateCharacterAttributesParams{CharacterID: character.ID})
	factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: character.ID})
	factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: character.ID})
	factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{CharacterID: character.ID})
	factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{CharacterID: character.ID})
	factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: character.ID})
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: character.ID})
	factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: character.ID})
	factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: character.ID})
	factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{CharacterID: character.ID})
	factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: character.ID})
	factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: character.ID})

	bu := ui.MakeFakeBaseUI(st, ui.NewFakeApp(t), true)
	u := ui.NewDesktopUI(bu)
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		for {
			<-ticker.C
			if u.IsStartupCompleted() {
				u.App().Quit()
			}
		}
	}()
	u.ShowAndRun()
	assert.Equal(t, 0, httpmock.GetTotalCallCount())
}

func TestUI_CanUpdateAllEmpty(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	test.ApplyTheme(t, test.Theme())
	bu := ui.MakeFakeBaseUI(st, ui.NewFakeApp(t), true)

	du := ui.NewDesktopUI(bu)
	w := test.NewWindow(du.MainWindow().Content())
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 600))
	du.Start()
	time.Sleep(1 * time.Second)
	// test.AssertImageMatches(t, "ui/empty.png", w.Canvas().Capture())
	test.RenderToMarkup(du.MainWindow().Canvas())
}

// TODO: Extend test to cover all tabs with data

func TestUI_CanUpdateAllWithData(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	test.ApplyTheme(t, test.Theme())
	bu := ui.MakeFakeBaseUI(st, ui.NewFakeApp(t), true)

	er := factory.CreateEveCorporation(storage.UpdateOrCreateEveCorporationParams{
		Name: "Wayne Technology",
	})
	factory.CreateEveEntityCorporation(*er.ToEveEntity())
	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name:          "Bruce Wayne",
		CorporationID: er.ID,
	})
	character := factory.CreateCharacter(storage.CreateCharacterParams{
		ID: ec.ID,
	})
	factory.SetCharacterRoles(character.ID, set.Collect(app.RolesAll()))
	corporation := factory.CreateCorporation(character.EveCharacter.Corporation.ID)

	et := factory.CreateEveType()
	factory.CreateEveMarketPrice(storage.UpdateOrCreateEveMarketPriceParams{
		TypeID:       et.ID,
		AveragePrice: 120000.0,
	})
	factory.CreateCharacterAsset(storage.CreateCharacterAssetParams{
		CharacterID: character.ID,
		EveTypeID:   et.ID,
		Quantity:    1,
	})

	factory.CreateCharacterAttributes(storage.UpdateOrCreateCharacterAttributesParams{CharacterID: character.ID})
	factory.CreateCharacterContract(storage.CreateCharacterContractParams{CharacterID: character.ID})
	factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: character.ID})
	factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{CharacterID: character.ID})
	factory.CreateCharacterJumpClone(storage.CreateCharacterJumpCloneParams{CharacterID: character.ID})
	factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: character.ID})
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{CharacterID: character.ID})
	factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{CharacterID: character.ID})
	factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: character.ID})
	factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{CharacterID: character.ID})
	factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: character.ID})
	factory.CreateCharacterWalletTransaction(storage.CreateCharacterWalletTransactionParams{CharacterID: character.ID})
	for _, s := range app.CharacterSections {
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: character.ID,
			Section:     s,
		})
	}
	factory.CreateCorporationWalletBalance(storage.UpdateOrCreateCorporationWalletBalanceParams{CorporationID: corporation.ID})
	factory.CreateCorporationMember(storage.CorporationMemberParams{CorporationID: corporation.ID})
	factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{CorporationID: corporation.ID})
	factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
		CorporationID: corporation.ID,
		DivisionID:    app.Division1.ID(),
	})
	factory.CreateCorporationWalletTransaction(storage.CreateCorporationWalletTransactionParams{
		CorporationID: corporation.ID,
		DivisionID:    app.Division1.ID(),
	})
	for _, s := range app.CorporationSections {
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: corporation.ID,
			Section:       s,
		})
	}
	du := ui.NewDesktopUI(bu)
	w := test.NewWindow(du.MainWindow().Content())
	defer w.Close()
	w.Resize(fyne.NewSize(1000, 600))
	du.Start()
	time.Sleep(1 * time.Second)
	// test.AssertImageMatches(t, "ui/full.png", w.Canvas().Capture())
	test.RenderToMarkup(du.MainWindow().Canvas())
}
