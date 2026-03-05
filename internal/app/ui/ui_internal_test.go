package ui

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/eveauth"
	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatusservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
)

const (
	SkipUIReason = "This test is skipped for CI as it is flaky"
)

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

type CharacterServiceFake struct {
	Token          *app.CharacterToken
	CorporationIDs set.Set[int64]
	Error          error
}

type tokenSourceFake struct {
	token *app.CharacterToken
	err   error
}

func (ts tokenSourceFake) Token() (*oauth2.Token, error) {
	if ts.err != nil {
		return nil, ts.err
	}
	return ts.token.OauthToken(), nil
}

func (s *CharacterServiceFake) TokenSourceForCorporation(ctx context.Context, corporationID int64, roles set.Set[app.Role], scopes set.Set[string]) (oauth2.TokenSource, int64, error) {
	if s.Error != nil {
		return &tokenSourceFake{token: s.Token, err: s.Error}, 0, nil
	}
	return &tokenSourceFake{token: s.Token, err: nil}, s.Token.CharacterID, nil
}

func MakeFakeBaseUI(st *storage.Storage, fyneApp fyne.App, isDesktop bool) *baseUI {
	esiClient := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
		UserAgent: "EveBuddy/1.0 (test@kalkoken.net)",
	})
	scs := statuscacheservice.New(st)
	if err := scs.InitCache(context.Background()); err != nil {
		panic(err)
	}
	signals := app.NewSignals()
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		ESIClient:          esiClient,
		Signals:            signals,
		StatusCacheService: scs,
		Storage:            st,
	})
	ac, err := eveauth.NewClient(eveauth.Config{
		ClientID: "DUMMY",
		Port:     8000,
	})
	if err != nil {
		panic(err)
	}
	cs := characterservice.New(characterservice.Params{
		AuthClient:             ac,
		Cache:                  testutil.NewCacheFake2(),
		ESIClient:              esiClient,
		EveNotificationService: evenotification.New(eus),
		EveUniverseService:     eus,
		StatusCacheService:     scs,
		Storage:                st,
	})
	rs := corporationservice.New(corporationservice.Params{
		Cache: testutil.NewCacheFake2(),
		CharacterService: &CharacterServiceFake{Token: &app.CharacterToken{
			AccessToken: "accessToken",
		}},
		ESIClient:          esiClient,
		EveUniverseService: eus,
		Signals:            signals,
		StatusCacheService: scs,
		Storage:            st,
	})
	eisFake := &testutil.EveImageServiceFake{
		Character:   icons.Characterplaceholder64Jpeg,
		Alliance:    icons.Corporationplaceholder64Png,
		Corporation: icons.Corporationplaceholder64Png,
		Err:         nil,
		Faction:     icons.Factionplaceholder64Png,
		Type:        icons.Typeplaceholder64Png,
	}
	bu := NewBaseUI(BaseUIParams{
		App:                fyneApp,
		CharacterService:   cs,
		CorporationService: rs,
		ESIStatusService:   esistatusservice.New(esiClient),
		EveImageService:    eisFake,
		EveUniverseService: eus,
		IsMobile:           !isDesktop,
		IsOffline:          true,
		JaniceService:      janiceservice.New(http.DefaultClient, ""),
		Signals:            signals,
		StatusCacheService: scs,
	})
	return bu
}

func TestMakeOrFindWindow(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	t.Run("should create new window when it does not yet exist", func(t *testing.T) {
		ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
		w, ok := ui.GetOrCreateWindow("abc", "title")
		assert.True(t, ok)
		assert.Contains(t, w.Title(), "title")
	})
	t.Run("should return existing window", func(t *testing.T) {
		ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
		ui.GetOrCreateWindow("abc", "title-old")
		w, ok := ui.GetOrCreateWindow("abc", "title-new")
		assert.False(t, ok)
		assert.Contains(t, w.Title(), "title-old")
	})
	t.Run("should create new window when previous one was closed", func(t *testing.T) {
		ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
		w, _ := ui.GetOrCreateWindow("abc", "title-old")
		w.Close()
		w, ok := ui.GetOrCreateWindow("abc", "title-new")
		assert.True(t, ok)
		assert.Contains(t, w.Title(), "title-new")
	})
	t.Run("should create new window when previous one was reshown and then closed", func(t *testing.T) {
		ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
		ui.GetOrCreateWindow("abc", "title-old")
		w, ok := ui.GetOrCreateWindow("abc", "title-new")
		assert.False(t, ok)
		assert.Contains(t, w.Title(), "title-old")
		w.Close()
		w, ok = ui.GetOrCreateWindow("abc", "title-new")
		assert.True(t, ok)
		assert.Contains(t, w.Title(), "title-new")
	})
	t.Run("should allow setting onClose calback by caller", func(t *testing.T) {
		ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
		w, _, onClosed := ui.GetOrCreateWindowWithOnClosed("abc", "title-old")
		var called bool
		w.SetOnClosed(func() {
			onClosed()
			called = true
		})
		w.Close()
		w, ok := ui.GetOrCreateWindow("abc", "title-new")
		assert.True(t, ok)
		assert.True(t, called)
		assert.Contains(t, w.Title(), "title-new")
	})
}
