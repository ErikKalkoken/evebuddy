package ui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/antihax/goesi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatusservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
)

type cache map[string][]byte

func newCache() cache {
	return make(cache)
}

func (c cache) Get(k string) ([]byte, bool) {
	v, ok := c[k]
	return v, ok
}

func (c cache) Set(k string, v []byte, d time.Duration) {
	c[k] = v
}

func (c cache) Clear() {
	for k := range c {
		delete(c, k)
	}
}

func TestUIStartEmpty(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterNoResponder(httpmock.NewNotFoundResponder(t.Fatal)) // fails on any HTTP request
	db, st, _ := testutil.NewWithOptions(false)
	defer db.Close()
	esiClient := goesi.NewAPIClient(nil, "dummy")
	cache := memcache.New()
	sc := statuscacheservice.New(cache)
	eus := eveuniverseservice.New(st, esiClient)
	eus.StatusCacheService = sc
	eis := eveimageservice.New(newCache(), nil, true)
	cs := characterservice.New(st, nil, esiClient)
	cs.EveUniverseService = eus
	cs.StatusCacheService = sc
	u := NewBaseUI(BaseUIParams{
		App:                test.NewTempApp(t),
		CharacterService:   cs,
		ESIStatusService:   esistatusservice.New(esiClient),
		EveImageService:    eis,
		EveUniverseService: eus,
		MemCache:           cache,
		StatusCacheService: sc,
		IsOffline:          true,
	})
	// u := NewDesktopUI(bu)
	u.Init()
	for _, f := range u.updateCharacterMap() {
		f()
	}
	for _, f := range u.updateCrossPagesMap() {
		f()
	}
	u.MainWindow().Show()
	assert.False(t, u.HasCharacter())
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

	esiClient := goesi.NewAPIClient(nil, "dummy")
	cache := memcache.New()
	sc := statuscacheservice.New(cache)
	eus := eveuniverseservice.New(st, esiClient)
	eus.StatusCacheService = sc
	eis := eveimageservice.New(newCache(), nil, true)
	cs := characterservice.New(st, nil, esiClient)
	cs.EveUniverseService = eus
	cs.StatusCacheService = sc
	u := NewBaseUI(BaseUIParams{
		App:                test.NewTempApp(t),
		CharacterService:   cs,
		ESIStatusService:   esistatusservice.New(esiClient),
		EveImageService:    eis,
		EveUniverseService: eus,
		MemCache:           cache,
		StatusCacheService: sc,
		IsOffline:          true,
	})
	// u := NewDesktopUI(bu)
	u.Init()
	for _, f := range u.updateCharacterMap() {
		f()
	}
	for _, f := range u.updateCrossPagesMap() {
		f()
	}
	u.MainWindow().Show()
	assert.Equal(t, character, u.CurrentCharacter())
}
