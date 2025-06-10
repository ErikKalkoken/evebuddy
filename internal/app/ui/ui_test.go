package ui_test

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

func TestUIStartEmpty(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterNoResponder(httpmock.NewNotFoundResponder(t.Fatal)) // fails on any HTTP request
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	bu := ui.NewFakeBaseUI(st, ui.NewFakeApp(t), true)
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

func TestUIStartWithCharacter(t *testing.T) {
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

	bu := ui.NewFakeBaseUI(st, ui.NewFakeApp(t), true)
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

func TestCanUpdateAllEmpty(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	bu := ui.NewFakeBaseUI(st, test.NewTempApp(t), true)
	bu.UpdateAll()
}

func TestCanUpdateAllWithData(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	test.ApplyTheme(t, test.Theme())
	bu := ui.NewFakeBaseUI(st, ui.NewFakeApp(t), true)
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
	for _, s := range app.CharacterSections {
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: character.ID,
			Section:     s,
		})
	}
	bu.UpdateAll()
	u := ui.NewDesktopUI(bu)
	test.RenderToMarkup(u.MainWindow().Canvas())
}
