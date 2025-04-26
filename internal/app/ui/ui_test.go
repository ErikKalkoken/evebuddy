package ui_test

import (
	"testing"
	"time"

	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatusservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
)

func TestUIStartEmpty(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterNoResponder(httpmock.NewNotFoundResponder(t.Fatal)) // fails on any HTTP request
	db, st, _ := testutil.NewWithOptions(false)
	defer db.Close()
	esiClient := goesi.NewAPIClient(nil, "dummy")
	cache := memcache.New()
	scs := statuscacheservice.New(cache, st)
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		ESIClient:          esiClient,
		StatusCacheService: scs,
		Storage:            st,
	})
	eis := eveimageservice.New(ui.NewFakeCache(), nil, true)
	cs := characterservice.New(characterservice.Params{
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
	})
	bu := ui.NewBaseUI(ui.BaseUIParams{
		App:                ui.NewFakeApp(t),
		CharacterService:   cs,
		ESIStatusService:   esistatusservice.New(esiClient),
		EveImageService:    eis,
		EveUniverseService: eus,
		MemCache:           cache,
		StatusCacheService: scs,
		IsOffline:          true,
	})
	u := ui.NewDesktopUI(bu)
	u.Init()
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
	db, st, factory := testutil.NewWithOptions(false)
	defer db.Close()

	character := factory.CreateCharacter()
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

	bu := ui.NewFakeBaseUI(st, ui.NewFakeApp(t))
	u := ui.NewDesktopUI(bu)
	u.Init()
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
