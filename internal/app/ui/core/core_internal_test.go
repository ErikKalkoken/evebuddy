package core

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatusservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

// AppFake is an extension of the Fyne test app which also conforms to the desktop app interface.
type AppFake struct {
	app      fyne.App
	metadata fyne.AppMetadata
}

func NewFakeApp(t testing.TB) *AppFake {
	a := &AppFake{app: test.NewTempApp(t)}
	return a
}

func (a *AppFake) NewWindow(title string) fyne.Window {
	return a.app.NewWindow(title)
}

func (a *AppFake) Clipboard() fyne.Clipboard {
	return a.app.Clipboard()
}

func (a *AppFake) OpenURL(url *url.URL) error {
	return a.app.OpenURL(url)
}

func (a *AppFake) Icon() fyne.Resource {
	return a.app.Icon()
}

func (a *AppFake) SetIcon(r fyne.Resource) {
	a.app.SetIcon(r)
}

func (a *AppFake) Run() {
	a.app.Run()
}

func (a *AppFake) Quit() {
	a.app.Quit()
}

func (a *AppFake) Driver() fyne.Driver {
	return a.app.Driver()
}

func (a *AppFake) UniqueID() string {
	return a.app.UniqueID()
}

func (a *AppFake) SendNotification(n *fyne.Notification) {
	a.app.SendNotification(n)
}

func (a *AppFake) Settings() fyne.Settings {
	return a.app.Settings()
}

func (a *AppFake) Preferences() fyne.Preferences {
	return a.app.Preferences()
}

func (a *AppFake) Storage() fyne.Storage {
	return a.app.Storage()
}

func (a *AppFake) Lifecycle() fyne.Lifecycle {
	return a.app.Lifecycle()
}

func (a *AppFake) Metadata() fyne.AppMetadata {
	return a.metadata
}

// SetMetadata sets the metadata returned by Metadata.
func (a *AppFake) SetMetadata(m fyne.AppMetadata) {
	a.metadata = m
}

func (a *AppFake) CloudProvider() fyne.CloudProvider {
	return a.app.CloudProvider()
}

func (a *AppFake) SetCloudProvider(o fyne.CloudProvider) {
	a.app.SetCloudProvider(o)
}

func (a *AppFake) SetSystemTrayMenu(_ *fyne.Menu) {
	// noop
}

func (a *AppFake) SetSystemTrayIcon(_ fyne.Resource) {
	// noop
}

func (a *AppFake) SetSystemTrayWindow(fyne.Window) {
	// noop
}

func (a *AppFake) Cache() fyne.Cache {
	return a.app.Cache()
}

func (a *AppFake) ScheduleNotification(n *fyne.Notification, deliverAt time.Time) (*fyne.ScheduledNotification, error) {
	panic("not implemented")
}

func (a *AppFake) CancelScheduledNotification(id string) error {
	return nil
}

var _ fyne.App = (*AppFake)(nil)
var _ desktop.App = (*AppFake)(nil)

// MakeFakeBaseUI returns a baseUI with test doubles of its services.
// Signals call their listeners synchronously. Background tasks and settings are stopped at the end of the test.
func MakeFakeBaseUI(t testing.TB, st *storage.Storage, fyneApp fyne.App) *baseUI {
	esiClient := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
		UserAgent: "EveBuddy/1.0 (test@kalkoken.net)",
	})
	scs := new(statuscache.StatusCache)
	if err := scs.Init(context.Background(), st); err != nil {
		t.Fatal(err)
	}
	signals := testdouble.NewSignalsSync()
	settings, err := settings.New(st)
	if err != nil {
		t.Fatal(err)
	}
	eus := testdouble.NewEVEUniverseServiceFake(eveuniverseservice.Params{
		ESIClient:          esiClient,
		Signals:            signals,
		StatusCacheService: scs,
		Storage:            st,
	})
	cs := testdouble.NewCharacterServiceFake(characterservice.Params{
		ESIClient:          esiClient,
		EveUniverseService: eus,
		Settings:           settings,
		Signals:            signals,
		StatusCacheService: scs,
		Storage:            st,
	})
	rs := testdouble.NewCorporationServiceFake(corporationservice.Params{
		CharacterService:   cs,
		ESIClient:          esiClient,
		EveUniverseService: eus,
		Settings:           settings,
		Signals:            signals,
		StatusCacheService: scs,
		Storage:            st,
	})
	u := newBaseUI(UIParams{
		App:         fyneApp,
		Character:   cs,
		Corporation: rs,
		ESIStatus:   esistatusservice.New(esiClient),
		EVEImage:    testutil.NewEveImageServiceStub(),
		EVEUniverse: eus,
		Janice:      janiceservice.New(http.DefaultClient, ""),
		Settings:    settings,
		Signals:     signals,
		StatusCache: scs,
	})
	t.Cleanup(func() {
		u.tasks.Stop()
		settings.Close()
	})
	return u
}

// TestNewBaseUI_InfoViewerInitializedBeforeWidgets is a regression test for the bug where
// u.iw was initialized after the widget constructors. NewCommunications and NewMails both
// call u.InfoViewer().Show to bind a method value at construction time. When u.iw was nil
// at that point, the bound method had a nil receiver and panicked on the first tap of any
// sender, recipient, or icon in the mail or notification detail views.
func TestNewBaseUI_InfoViewerInitializedBeforeWidgets(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
	assert.NotNil(t, u.iw)
}

func TestMakeOrFindWindow(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	t.Run("should create new window when it does not yet exist", func(t *testing.T) {
		ui := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		w, ok := ui.GetOrCreateWindow("abc", "title")
		assert.True(t, ok)
		assert.Contains(t, w.Title(), "title")
	})
	t.Run("should return existing window", func(t *testing.T) {
		ui := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		ui.GetOrCreateWindow("abc", "title-old")
		w, ok := ui.GetOrCreateWindow("abc", "title-new")
		assert.False(t, ok)
		assert.Contains(t, w.Title(), "title-old")
	})
	t.Run("should create new window when previous one was closed", func(t *testing.T) {
		ui := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		w, _ := ui.GetOrCreateWindow("abc", "title-old")
		w.Close()
		w, ok := ui.GetOrCreateWindow("abc", "title-new")
		assert.True(t, ok)
		assert.Contains(t, w.Title(), "title-new")
	})
	t.Run("should create new window when previous one was reshown and then closed", func(t *testing.T) {
		ui := MakeFakeBaseUI(t, st, test.NewTempApp(t))
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
		ui := MakeFakeBaseUI(t, st, test.NewTempApp(t))
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

func TestBaseUI_Go(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	t.Run("runs the job", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		done := make(chan struct{})
		ok := u.Go(func(ctx context.Context) {
			close(done)
		})
		assert.True(t, ok)
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("job did not run")
		}
	})
	t.Run("stopping cancels the job and waits for it", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		entered := make(chan struct{})
		var wasCanceled bool
		u.Go(func(ctx context.Context) {
			close(entered)
			<-ctx.Done()
			wasCanceled = true
		})
		<-entered
		u.tasks.Stop()
		assert.True(t, wasCanceled)
	})
	t.Run("skips the job after stop", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.tasks.Stop()
		ok := u.Go(func(ctx context.Context) {
			t.Error("job should not run")
		})
		assert.False(t, ok)
	})
}

func TestBaseUI_InitKeepsLastSelectionWhenCanceled(t *testing.T) {
	// On disk, because a canceled query drops its connection, which would wipe an in-memory DB.
	_, st, factory := testutil.NewDBOnDisk(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	t.Run("character", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		c := factory.CreateCharacterFull()
		u.settings.SetLastCharacterID(c.ID)
		// when
		u.initCharacter(ctx)
		// then
		assert.Equal(t, c.ID, u.settings.LastCharacterID())
	})
	t.Run("corporation", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		c := factory.CreateCorporation()
		u.settings.SetLastCorporationID(c.ID)
		// when
		u.initCorporation(ctx)
		// then
		assert.Equal(t, c.ID, u.settings.LastCorporationID())
	})
}

func TestFormatCount(t *testing.T) {
	format := func(v int) string {
		return fmt.Sprintf("%d ready", v)
	}
	t.Run("shows ? when unknown", func(t *testing.T) {
		assert.Equal(t, "?", formatCount(optional.Optional[int]{}, format))
	})
	t.Run("shows nothing for 0", func(t *testing.T) {
		assert.Equal(t, "", formatCount(optional.New(0), format))
	})
	t.Run("shows formatted count", func(t *testing.T) {
		assert.Equal(t, "3 ready", formatCount(optional.New(3), format))
	})
}

func TestFormatMailUnread(t *testing.T) {
	t.Run("shows ? unread when unknown", func(t *testing.T) {
		assert.Equal(t, "? unread", formatMailUnread(optional.Optional[int]{}))
	})
	t.Run("shows nothing for 0", func(t *testing.T) {
		assert.Equal(t, "", formatMailUnread(optional.New(0)))
	})
	t.Run("shows unread count", func(t *testing.T) {
		assert.Equal(t, "1,234 unread", formatMailUnread(optional.New(1234)))
	})
}

func TestShakedown(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()

	t.Run("can render status bar", func(t *testing.T) {
		du := &DesktopUI{baseUI: MakeFakeBaseUI(t, st, test.NewTempApp(t))}
		du.tasks.Stop() // don't start the tickers
		sb := newStatusBar(du)
		sb.start(ctx)
		w := test.NewWindow(sb)
		defer w.Close()
		w.Resize(fyne.NewSize(800, 50))
	})
	t.Run("can render about page", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.tasks.Stop() // don't check for updates
		u.SetDeveloperMode(true)
		w := test.NewWindow(newAboutPage(u))
		defer w.Close()
		w.Resize(fyne.NewSize(600, 600))
	})
	t.Run("can render nav selector", func(t *testing.T) {
		test.NewTempApp(t)
		ns := newNavSelector()
		ns.SetTitle("Bruce Wayne")
		ns.SetMenu([]*fyne.MenuItem{fyne.NewMenuItem("Alpha", nil)})
		w := test.NewWindow(ns)
		defer w.Close()
		w.Resize(fyne.NewSize(300, 50))
	})
}
