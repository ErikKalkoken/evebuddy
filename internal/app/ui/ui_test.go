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

// testApp is an extension of the Fyne test app which also conforms to the desktop app interface.
type testApp struct {
	app fyne.App
}

func newTestApp(t testing.TB) *testApp {
	a := &testApp{app: test.NewTempApp(t)}
	return a
}

func (a *testApp) NewWindow(title string) fyne.Window {
	return a.app.NewWindow(title)
}

func (a *testApp) OpenURL(url *url.URL) error {
	return a.app.OpenURL(url)
}

func (a *testApp) Icon() fyne.Resource {
	return a.app.Icon()
}

func (a *testApp) SetIcon(r fyne.Resource) {
	a.app.SetIcon(r)
}

func (a *testApp) Run() {
	a.app.Run()
}

func (a *testApp) Quit() {
	a.app.Quit()
}

func (a *testApp) Driver() fyne.Driver {
	return a.app.Driver()
}

func (a *testApp) UniqueID() string {
	return a.app.UniqueID()
}

func (a *testApp) SendNotification(n *fyne.Notification) {
	a.app.SendNotification(n)
}

func (a *testApp) Settings() fyne.Settings {
	return a.app.Settings()
}

func (a *testApp) Preferences() fyne.Preferences {
	return a.app.Preferences()
}

func (a *testApp) Storage() fyne.Storage {
	return a.app.Storage()
}

func (a *testApp) Lifecycle() fyne.Lifecycle {
	return a.app.Lifecycle()
}

func (a *testApp) Metadata() fyne.AppMetadata {
	return a.app.Metadata()
}

func (a *testApp) CloudProvider() fyne.CloudProvider {
	return a.app.CloudProvider()
}

func (a *testApp) SetCloudProvider(o fyne.CloudProvider) {
	a.app.SetCloudProvider(o)
}

func (a *testApp) SetSystemTrayMenu(_ *fyne.Menu) {
	// noop
}

func (a *testApp) SetSystemTrayIcon(_ fyne.Resource) {
	// noop
}

func (a *testApp) Clipboard() fyne.Clipboard {
	return a.app.Clipboard()
}

var _ fyne.App = (*testApp)(nil)
var _ desktop.App = (*testApp)(nil)

func TestUIStartEmpty(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterNoResponder(httpmock.NewNotFoundResponder(t.Fatal)) // fails on any HTTP request
	db, st, _ := testutil.NewWithOptions(false)
	defer db.Close()
	esiClient := goesi.NewAPIClient(nil, "dummy")
	cache := memcache.New()
	scs := statuscacheservice.New(cache)
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		ESIClient:          esiClient,
		StatusCacheService: scs,
		Storage:            st,
	})
	eis := eveimageservice.New(newCache(), nil, true)
	cs := characterservice.New(characterservice.Params{
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
	})
	bu := NewBaseUI(BaseUIParams{
		App:                newTestApp(t),
		CharacterService:   cs,
		ESIStatusService:   esistatusservice.New(esiClient),
		EveImageService:    eis,
		EveUniverseService: eus,
		MemCache:           cache,
		StatusCacheService: scs,
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
	scs := statuscacheservice.New(cache)
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		ESIClient:          esiClient,
		StatusCacheService: scs,
		Storage:            st,
	})
	eis := eveimageservice.New(newCache(), nil, true)
	cs := characterservice.New(characterservice.Params{
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
	})
	bu := NewBaseUI(BaseUIParams{
		App:                newTestApp(t),
		CharacterService:   cs,
		ESIStatusService:   esistatusservice.New(esiClient),
		EveImageService:    eis,
		EveUniverseService: eus,
		MemCache:           cache,
		StatusCacheService: scs,
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
