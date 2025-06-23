package ui

import (
	"context"
	"net/url"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatusservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	"github.com/antihax/goesi"
	"github.com/stretchr/testify/assert"
)

type FakeCache map[string][]byte

func NewFakeCache() FakeCache {
	return make(FakeCache)
}

func (c FakeCache) Get(k string) ([]byte, bool) {
	v, ok := c[k]
	return v, ok
}

func (c FakeCache) Set(k string, v []byte, d time.Duration) {
	c[k] = v
}

func (c FakeCache) Clear() {
	for k := range c {
		delete(c, k)
	}
}

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

func (a *FakeApp) Clipboard() fyne.Clipboard {
	return a.app.Clipboard()
}

var _ fyne.App = (*FakeApp)(nil)
var _ desktop.App = (*FakeApp)(nil)

type NewFakeBaseUIParams struct {
}

func NewFakeBaseUI(st *storage.Storage, app fyne.App, isDesktop bool) *baseUI {
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
	eis := eveimageservice.New(NewFakeCache(), nil, true)
	cs := characterservice.New(characterservice.Params{
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
	})
	rs := corporationservice.New(corporationservice.Params{
		CharacterService:   cs,
		EsiClient:          esiClient,
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
	})
	bu := NewBaseUI(BaseUIParams{
		App:                app,
		CharacterService:   cs,
		CorporationService: rs,
		ESIStatusService:   esistatusservice.New(esiClient),
		EveImageService:    eis,
		EveUniverseService: eus,
		MemCache:           cache,
		StatusCacheService: scs,
		IsOffline:          true,
		IsDesktop:          isDesktop,
	})
	return bu
}

func TestIsTimeWithinRange(t *testing.T) {
	start := "11:00"
	duration := 15 * time.Minute
	cases := []struct {
		name string
		t    time.Time
		want bool
	}{

		{"in range", time.Date(2025, 1, 1, 11, 3, 0, 0, time.UTC), true},
		{"equal start", time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC), true},
		{"equal end", time.Date(2025, 1, 1, 11, 15, 0, 0, time.UTC), true},
		{"after 1", time.Date(2025, 1, 1, 11, 16, 0, 0, time.UTC), false},
		{"after 2", time.Date(2025, 1, 1, 12, 10, 0, 0, time.UTC), false},
		{"before", time.Date(2025, 1, 1, 10, 16, 0, 0, time.UTC), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isTimeWithinRange(start, duration, tc.t)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMakeOrFindWindow(t *testing.T) {
	t.Run("should create new window when it does not yet exist", func(t *testing.T) {
		ui := NewBaseUI(BaseUIParams{App: test.NewTempApp(t)})
		w, ok := ui.getOrCreateWindow("abc", "title")
		assert.True(t, ok)
		assert.Equal(t, "title", w.Title())
	})
	t.Run("should return existing window", func(t *testing.T) {
		ui := NewBaseUI(BaseUIParams{App: test.NewTempApp(t)})
		ui.getOrCreateWindow("abc", "title-old")
		w, ok := ui.getOrCreateWindow("abc", "title-new")
		assert.False(t, ok)
		assert.Equal(t, "title-old", w.Title())
	})
	t.Run("should create new window when previous one was closed", func(t *testing.T) {
		ui := NewBaseUI(BaseUIParams{App: test.NewTempApp(t)})
		w, _ := ui.getOrCreateWindow("abc", "title-old")
		w.Close()
		w, ok := ui.getOrCreateWindow("abc", "title-new")
		assert.True(t, ok)
		assert.Equal(t, "title-new", w.Title())
	})
	t.Run("should create new window when previous one was reshown and then closed", func(t *testing.T) {
		ui := NewBaseUI(BaseUIParams{App: test.NewTempApp(t)})
		ui.getOrCreateWindow("abc", "title-old")
		w, ok := ui.getOrCreateWindow("abc", "title-new")
		assert.False(t, ok)
		assert.Equal(t, "title-old", w.Title())
		w.Close()
		w, ok = ui.getOrCreateWindow("abc", "title-new")
		assert.True(t, ok)
		assert.Equal(t, "title-new", w.Title())
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
		assert.Equal(t, "title-new", w.Title())
	})
}
