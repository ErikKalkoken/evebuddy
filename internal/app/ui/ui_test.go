package ui

import (
	"net/url"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
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

type myTestApp struct {
	app fyne.App
}

func newMyTestApp(t testing.TB) *myTestApp {
	a := &myTestApp{app: test.NewTempApp(t)}
	return a
}

func (a *myTestApp) NewWindow(title string) fyne.Window {
	return a.app.NewWindow(title)
}

func (a *myTestApp) OpenURL(url *url.URL) error {
	return a.app.OpenURL(url)
}

func (a *myTestApp) Icon() fyne.Resource {
	return a.app.Icon()
}

func (a *myTestApp) SetIcon(r fyne.Resource) {
	a.app.SetIcon(r)
}

func (a *myTestApp) Run() {
	a.app.Run()
}

func (a *myTestApp) Quit() {
	a.app.Quit()
}

func (a *myTestApp) Driver() fyne.Driver {
	return a.app.Driver()
}

func (a *myTestApp) UniqueID() string {
	return a.app.UniqueID()
}

func (a *myTestApp) SendNotification(n *fyne.Notification) {
	a.app.SendNotification(n)
}

func (a *myTestApp) Settings() fyne.Settings {
	return a.app.Settings()
}

func (a *myTestApp) Preferences() fyne.Preferences {
	return a.app.Preferences()
}

func (a *myTestApp) Storage() fyne.Storage {
	return a.app.Storage()
}

func (a *myTestApp) Lifecycle() fyne.Lifecycle {
	return a.app.Lifecycle()
}

func (a *myTestApp) Metadata() fyne.AppMetadata {
	return a.app.Metadata()
}

func (a *myTestApp) CloudProvider() fyne.CloudProvider {
	return a.app.CloudProvider()
}

func (a *myTestApp) SetCloudProvider(o fyne.CloudProvider) {
	a.app.SetCloudProvider(o)
}

func (a *myTestApp) SetSystemTrayMenu(_ *fyne.Menu) {
	// noop
}

func (a *myTestApp) SetSystemTrayIcon(_ fyne.Resource) {
	// noop
}

var _ desktop.App = (*myTestApp)(nil)

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
	bu := NewBaseUI(BaseUIParams{
		App:                newMyTestApp(t),
		CharacterService:   cs,
		ESIStatusService:   esistatusservice.New(esiClient),
		EveImageService:    eis,
		EveUniverseService: eus,
		MemCache:           cache,
		StatusCacheService: sc,
		IsOffline:          true,
	})
	u := NewDesktopUI(bu)
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
	bu := NewBaseUI(BaseUIParams{
		App:                newMyTestApp(t),
		CharacterService:   cs,
		ESIStatusService:   esistatusservice.New(esiClient),
		EveImageService:    eis,
		EveUniverseService: eus,
		MemCache:           cache,
		StatusCacheService: sc,
		IsOffline:          true,
	})
	u := NewDesktopUI(bu)
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
