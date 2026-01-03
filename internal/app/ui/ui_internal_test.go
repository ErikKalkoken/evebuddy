package ui

import (
	"context"
	"net/url"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/antihax/goesi"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatusservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
)

// type FakeCache map[string][]byte

// func NewFakeCache() FakeCache {
// 	return make(FakeCache)
// }

// func (c FakeCache) Get(k string) ([]byte, bool) {
// 	v, ok := c[k]
// 	return v, ok
// }

// func (c FakeCache) Set(k string, v []byte, d time.Duration) {
// 	c[k] = v
// }

// func (c FakeCache) Clear() {
// 	for k := range c {
// 		delete(c, k)
// 	}
// }

// FakeApp is an extension of the Fyne test app which also conforms to the desktop app interface.
type FakeApp struct {
	app fyne.App
}

func NewFakeApp(t testing.TB) *FakeApp {
	a := &FakeApp{app: test.NewTempApp(t)}
	return a
}

func (a *FakeApp) NewWindow(title string) fyne.Window {
	return a.app.NewWindow(title)
}

func (a *FakeApp) Clipboard() fyne.Clipboard {
	return a.app.Clipboard()
}

func (a *FakeApp) OpenURL(url *url.URL) error {
	return a.app.OpenURL(url)
}

func (a *FakeApp) Icon() fyne.Resource {
	return a.app.Icon()
}

func (a *FakeApp) SetIcon(r fyne.Resource) {
	a.app.SetIcon(r)
}

func (a *FakeApp) Run() {
	a.app.Run()
}

func (a *FakeApp) Quit() {
	a.app.Quit()
}

func (a *FakeApp) Driver() fyne.Driver {
	return a.app.Driver()
}

func (a *FakeApp) UniqueID() string {
	return a.app.UniqueID()
}

func (a *FakeApp) SendNotification(n *fyne.Notification) {
	a.app.SendNotification(n)
}

func (a *FakeApp) Settings() fyne.Settings {
	return a.app.Settings()
}

func (a *FakeApp) Preferences() fyne.Preferences {
	return a.app.Preferences()
}

func (a *FakeApp) Storage() fyne.Storage {
	return a.app.Storage()
}

func (a *FakeApp) Lifecycle() fyne.Lifecycle {
	return a.app.Lifecycle()
}

func (a *FakeApp) Metadata() fyne.AppMetadata {
	return a.app.Metadata()
}

func (a *FakeApp) CloudProvider() fyne.CloudProvider {
	return a.app.CloudProvider()
}

func (a *FakeApp) SetCloudProvider(o fyne.CloudProvider) {
	a.app.SetCloudProvider(o)
}

func (a *FakeApp) SetSystemTrayMenu(_ *fyne.Menu) {
	// noop
}

func (a *FakeApp) SetSystemTrayIcon(_ fyne.Resource) {
	// noop
}

func (a *FakeApp) SetSystemTrayWindow(fyne.Window) {
	// noop
}

var _ fyne.App = (*FakeApp)(nil)
var _ desktop.App = (*FakeApp)(nil)

type EveImageServiceFake struct {
	Alliance    fyne.Resource
	Character   fyne.Resource
	Corporation fyne.Resource
	Err         error
	Faction     fyne.Resource
	Type        fyne.Resource
}

func (s *EveImageServiceFake) AllianceLogo(id int32, size int) (fyne.Resource, error) {
	return s.Alliance, s.Err
}
func (s *EveImageServiceFake) CharacterPortrait(id int32, size int) (fyne.Resource, error) {
	return s.Character, s.Err
}

func (s *EveImageServiceFake) CorporationLogo(id int32, size int) (fyne.Resource, error) {
	return s.Corporation, s.Err
}

func (s *EveImageServiceFake) FactionLogo(id int32, size int) (fyne.Resource, error) {
	return s.Faction, s.Err
}

func (s *EveImageServiceFake) InventoryTypeRender(id int32, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceFake) InventoryTypeIcon(id int32, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceFake) InventoryTypeBPO(id int32, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceFake) InventoryTypeBPC(id int32, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

func (s *EveImageServiceFake) InventoryTypeSKIN(id int32, size int) (fyne.Resource, error) {
	return s.Type, s.Err
}

type CharacterServiceFake struct {
	Token          *app.CharacterToken
	CorporationIDs set.Set[int32]
	Error          error
}

func (s *CharacterServiceFake) CharacterTokenForCorporation(ctx context.Context, corporationID int32, roles set.Set[app.Role], scopes set.Set[string], checkToken bool) (*app.CharacterToken, error) {
	return s.Token, s.Error
}

func MakeFakeBaseUI(st *storage.Storage, fyneApp fyne.App, isDesktop bool) *baseUI {
	esiClient := goesi.NewAPIClient(nil, "dummy")
	cache := memcache.New()
	scs := statuscacheservice.New(cache, st)
	if err := scs.InitCache(context.Background()); err != nil {
		panic(err)
	}
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		ESIClient:          esiClient,
		StatusCacheService: scs,
		Storage:            st,
	})
	cs := characterservice.New(characterservice.Params{
		Cache:              testutil.NewCacheFake2(),
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
	})
	rs := corporationservice.New(corporationservice.Params{
		Cache: testutil.NewCacheFake2(),
		CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}},
		EsiClient:          esiClient,
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
	})
	bu := NewBaseUI(BaseUIParams{
		App:                fyneApp,
		CharacterService:   cs,
		CorporationService: rs,
		ESIStatusService:   esistatusservice.New(esiClient),
		EveImageService: &EveImageServiceFake{
			Character:   icons.Characterplaceholder64Jpeg,
			Alliance:    icons.Corporationplaceholder64Png,
			Corporation: icons.Corporationplaceholder64Png,
			Err:         nil,
			Faction:     icons.Factionplaceholder64Png,
			Type:        icons.Typeplaceholder64Png,
		},
		EveUniverseService: eus,
		MemCache:           cache,
		StatusCacheService: scs,
		IsOffline:          true,
		IsMobile:           !isDesktop,
	})
	return bu
}

func TestMakeOrFindWindow(t *testing.T) {
	t.Run("should create new window when it does not yet exist", func(t *testing.T) {
		ui := NewBaseUI(BaseUIParams{App: test.NewTempApp(t)})
		w, ok := ui.getOrCreateWindow("abc", "title")
		assert.True(t, ok)
		assert.Contains(t, w.Title(), "title")
	})
	t.Run("should return existing window", func(t *testing.T) {
		ui := NewBaseUI(BaseUIParams{App: test.NewTempApp(t)})
		ui.getOrCreateWindow("abc", "title-old")
		w, ok := ui.getOrCreateWindow("abc", "title-new")
		assert.False(t, ok)
		assert.Contains(t, w.Title(), "title-old")
	})
	t.Run("should create new window when previous one was closed", func(t *testing.T) {
		ui := NewBaseUI(BaseUIParams{App: test.NewTempApp(t)})
		w, _ := ui.getOrCreateWindow("abc", "title-old")
		w.Close()
		w, ok := ui.getOrCreateWindow("abc", "title-new")
		assert.True(t, ok)
		assert.Contains(t, w.Title(), "title-new")
	})
	t.Run("should create new window when previous one was reshown and then closed", func(t *testing.T) {
		ui := NewBaseUI(BaseUIParams{App: test.NewTempApp(t)})
		ui.getOrCreateWindow("abc", "title-old")
		w, ok := ui.getOrCreateWindow("abc", "title-new")
		assert.False(t, ok)
		assert.Contains(t, w.Title(), "title-old")
		w.Close()
		w, ok = ui.getOrCreateWindow("abc", "title-new")
		assert.True(t, ok)
		assert.Contains(t, w.Title(), "title-new")
	})
	t.Run("should allow setting onClose calback by caller", func(t *testing.T) {
		ui := NewBaseUI(BaseUIParams{App: test.NewTempApp(t)})
		w, _, onClosed := ui.getOrCreateWindowWithOnClosed("abc", "title-old")
		var called bool
		w.SetOnClosed(func() {
			onClosed()
			called = true
		})
		w.Close()
		w, ok := ui.getOrCreateWindow("abc", "title-new")
		assert.True(t, ok)
		assert.True(t, called)
		assert.Contains(t, w.Title(), "title-new")
	})
}
