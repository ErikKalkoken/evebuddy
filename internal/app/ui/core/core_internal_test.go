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
	"fyne.io/fyne/v2/theme"
	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatusservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
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
// It does not change fyne.CurrentApp(), which still returns the metadata of the inner test app.
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
	// closes the DB after the cleanup of MakeFakeBaseUI
	t.Cleanup(func() {
		db.Close()
	})
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

func TestBaseUI_ErrorDisplay(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	t.Cleanup(func() {
		db.Close()
	})
	err := fmt.Errorf("details: %w", app.ErrTokenError)
	t.Run("shows user friendly message", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		// when
		got := u.ErrorDisplay(err)
		// then
		xassert.Equal(t, "token error", got)
	})
	t.Run("shows full error in developer mode", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.SetDeveloperMode(true)
		// when
		got := u.ErrorDisplay(err)
		// then
		xassert.Equal(t, err.Error(), got)
	})
}

func TestBaseUI_MakeWindowTitle(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	t.Cleanup(func() {
		db.Close()
	})
	t.Run("joins parts with app name on desktop", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		// when
		got := u.MakeWindowTitle("Alpha", "Bravo")
		// then
		xassert.Equal(t, "Alpha - Bravo - "+ui.Name(), got)
	})
	t.Run("shows first part only on mobile", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.isMobile = true
		// when
		got := u.MakeWindowTitle("Alpha", "Bravo")
		// then
		xassert.Equal(t, "Alpha", got)
	})
	t.Run("shows placeholder when no parts given", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		// when
		got := u.MakeWindowTitle()
		// then
		xassert.Equal(t, "PLACEHOLDER - "+ui.Name(), got)
	})
}

func TestBaseUI_DestroyWindow(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	t.Cleanup(func() {
		db.Close()
	})
	t.Run("destroys existing window", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.GetOrCreateWindow("abc", "title")
		// when
		ok := u.DestroyWindow("abc")
		// then
		assert.True(t, ok)
		_, created := u.GetOrCreateWindow("abc", "title")
		assert.True(t, created)
	})
	t.Run("reports false for unknown window", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		// when
		ok := u.DestroyWindow("abc")
		// then
		assert.False(t, ok)
	})
	t.Run("reports false for empty id", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		// when
		ok := u.DestroyWindow("")
		// then
		assert.False(t, ok)
	})
}

func TestBaseUI_ClearAllCaches(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	t.Cleanup(func() {
		db.Close()
	})
	u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
	var called bool
	u.clearCache = func() {
		called = true
	}
	// when
	u.ClearAllCaches()
	// then
	assert.True(t, called)
}

func TestBaseUI_ListCorporationsForSelection(t *testing.T) {
	// On disk, because settings are persisted in a goroutine, which would wipe an in-memory DB.
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("lists all corporations of characters", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		c1 := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c1.ID, app.CorporationSectionWalletJournal(app.Division1))
		c2 := factory.CreateCorporation()
		factory.CreateCorporationToken(c2.ID, set.Of[app.Role](), set.Of[string]())
		// when
		xx, err := u.ListCorporationsForSelection(t.Context())
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(xx, func(x *app.EntityShort) int64 {
			return x.ID
		}))
		xassert.Equal(t, set.Of(c1.ID, c2.ID), got)
	})
	t.Run("lists privileged corporations only when limited ones are hidden", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.settings.SetHideLimitedCorporations(true)
		c1 := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c1.ID, app.CorporationSectionWalletJournal(app.Division1))
		c2 := factory.CreateCorporation()
		factory.CreateCorporationToken(c2.ID, set.Of[app.Role](), set.Of[string]())
		// when
		xx, err := u.ListCorporationsForSelection(t.Context())
		// then
		require.NoError(t, err)
		got := set.Collect(xiter.MapSlice(xx, func(x *app.EntityShort) int64 {
			return x.ID
		}))
		xassert.Equal(t, set.Of(c1.ID), got)
	})
}

func TestBaseUI_InitCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("uses last character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		// several, so that falling back to any character would likely pick another one
		for range 4 {
			factory.CreateCharacterFull()
		}
		c := factory.CreateCharacterFull()
		u.settings.SetLastCharacterID(c.ID)
		// when
		u.initCharacter(t.Context())
		// then
		xassert.Equal(t, c.ID, u.CurrentCharacter().IDOrZero())
	})
	t.Run("uses any character when last one is gone", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCharacterFull()
		u.settings.SetLastCharacterID(c.ID + 1)
		// when
		u.initCharacter(t.Context())
		// then
		xassert.Equal(t, c.ID, u.CurrentCharacter().IDOrZero())
		xassert.Equal(t, c.ID, u.settings.LastCharacterID())
	})
	t.Run("resets when there are no characters", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		u.settings.SetLastCharacterID(42)
		var emitted bool
		u.signals.CurrentCharacterExchanged.AddListener(func(_ context.Context, c *app.Character) {
			emitted = c == nil
		})
		// when
		u.initCharacter(t.Context())
		// then
		assert.Nil(t, u.CurrentCharacter())
		xassert.Equal(t, 0, u.settings.LastCharacterID())
		assert.True(t, emitted)
	})
}

func TestBaseUI_SetCharacter(t *testing.T) {
	_, st, factory := testutil.NewDBOnDisk(t)
	u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
	u.signals = testdouble.NewSignalsSync()
	c := factory.CreateCharacterFull()
	var emitted, called *app.Character
	u.signals.CurrentCharacterExchanged.AddListener(func(_ context.Context, c *app.Character) {
		emitted = c
	})
	u.onSetCharacter = func(c *app.Character) {
		called = c
	}
	// when
	u.SetCharacter(t.Context(), c)
	// then
	assert.Same(t, c, u.CurrentCharacter())
	assert.Same(t, c, emitted)
	assert.Same(t, c, called)
	xassert.Equal(t, c.ID, u.settings.LastCharacterID())
}

func TestBaseUI_ResetCharacter(t *testing.T) {
	_, st, factory := testutil.NewDBOnDisk(t)
	u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
	u.signals = testdouble.NewSignalsSync()
	c := factory.CreateCharacterFull()
	u.SetCharacter(t.Context(), c)
	var emitted bool
	u.signals.CurrentCharacterExchanged.AddListener(func(_ context.Context, c *app.Character) {
		emitted = c == nil
	})
	// when
	u.ResetCharacter(t.Context())
	// then
	assert.Nil(t, u.CurrentCharacter())
	assert.True(t, emitted)
	xassert.Equal(t, 0, u.settings.LastCharacterID())
}

func TestBaseUI_LoadCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("sets loaded character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCharacterFull()
		// when
		err := u.LoadCharacter(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, c.ID, u.CurrentCharacter().IDOrZero())
	})
	t.Run("returns error and keeps current character when not found", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCharacterFull()
		u.SetCharacter(t.Context(), c)
		// when
		err := u.LoadCharacter(t.Context(), c.ID+1)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
		assert.Same(t, c, u.CurrentCharacter())
	})
}

func TestBaseUI_SetAnyCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("sets a character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCharacterFull()
		// when
		err := u.SetAnyCharacter(t.Context())
		// then
		require.NoError(t, err)
		xassert.Equal(t, c.ID, u.CurrentCharacter().IDOrZero())
	})
	t.Run("resets when there are no characters", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		u.character.Store(&app.Character{ID: 42})
		// when
		err := u.SetAnyCharacter(t.Context())
		// then
		require.NoError(t, err)
		assert.Nil(t, u.CurrentCharacter())
	})
}

func TestBaseUI_ReloadCurrentCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("reloads current character from storage", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		c := factory.CreateCharacterFull()
		stale := &app.Character{ID: c.ID}
		u.character.Store(stale)
		// when
		u.ReloadCurrentCharacter(t.Context())
		// then
		got := u.CurrentCharacter()
		assert.NotSame(t, stale, got)
		xassert.Equal(t, c.EveCharacter.Name, got.EveCharacter.Name)
	})
	t.Run("does nothing when no character is set", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		factory.CreateCharacterFull()
		// when
		u.ReloadCurrentCharacter(t.Context())
		// then
		assert.Nil(t, u.CurrentCharacter())
	})
	t.Run("keeps current character when reload fails", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		stale := &app.Character{ID: 42}
		u.character.Store(stale)
		// when
		u.ReloadCurrentCharacter(t.Context())
		// then
		assert.Same(t, stale, u.CurrentCharacter())
	})
}

func TestBaseUI_ShowCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("shows and loads another character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c1 := factory.CreateCharacterFull()
		c2 := factory.CreateCharacterFull()
		u.SetCharacter(t.Context(), c1)
		var shown bool
		u.onShowCharacter = func() {
			shown = true
		}
		// when
		u.ShowCharacter(t.Context(), c2.ID)
		// then
		assert.True(t, shown)
		xassert.Equal(t, c2.ID, u.CurrentCharacter().IDOrZero())
	})
	t.Run("does not reload current character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCharacterFull()
		u.SetCharacter(t.Context(), c)
		var shown bool
		u.onShowCharacter = func() {
			shown = true
		}
		// when
		u.ShowCharacter(t.Context(), c.ID)
		// then
		assert.True(t, shown)
		assert.Same(t, c, u.CurrentCharacter())
	})
	t.Run("keeps current character when load fails", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCharacterFull()
		u.SetCharacter(t.Context(), c)
		// when
		u.ShowCharacter(t.Context(), c.ID+1)
		// then
		assert.Same(t, c, u.CurrentCharacter())
	})
}

func TestBaseUI_InitCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("uses last corporation", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		// several, so that falling back to any corporation would likely pick another one
		for range 4 {
			factory.CreateCorporation()
		}
		c := factory.CreateCorporation()
		u.settings.SetLastCorporationID(c.ID)
		// when
		u.initCorporation(t.Context())
		// then
		xassert.Equal(t, c.ID, u.CurrentCorporation().IDOrZero())
	})
	t.Run("uses any corporation when last one is gone", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCorporation()
		u.settings.SetLastCorporationID(c.ID + 1)
		// when
		u.initCorporation(t.Context())
		// then
		xassert.Equal(t, c.ID, u.CurrentCorporation().IDOrZero())
		xassert.Equal(t, c.ID, u.settings.LastCorporationID())
	})
	t.Run("resets when there are no corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		u.settings.SetLastCorporationID(42)
		var emitted bool
		u.signals.CurrentCorporationExchanged.AddListener(func(_ context.Context, c *app.Corporation) {
			emitted = c == nil
		})
		// when
		u.initCorporation(t.Context())
		// then
		assert.Nil(t, u.CurrentCorporation())
		xassert.Equal(t, 0, u.settings.LastCorporationID())
		assert.True(t, emitted)
	})
}

func TestBaseUI_SetCorporation(t *testing.T) {
	_, st, factory := testutil.NewDBOnDisk(t)
	u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
	u.signals = testdouble.NewSignalsSync()
	c := factory.CreateCorporation()
	var emitted, called *app.Corporation
	u.signals.CurrentCorporationExchanged.AddListener(func(_ context.Context, c *app.Corporation) {
		emitted = c
	})
	u.onSetCorporation = func(c *app.Corporation) {
		called = c
	}
	// when
	u.SetCorporation(t.Context(), c)
	// then
	assert.Same(t, c, u.CurrentCorporation())
	assert.Same(t, c, emitted)
	assert.Same(t, c, called)
	xassert.Equal(t, c.ID, u.settings.LastCorporationID())
}

func TestBaseUI_ResetCorporation(t *testing.T) {
	_, st, factory := testutil.NewDBOnDisk(t)
	u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
	u.signals = testdouble.NewSignalsSync()
	c := factory.CreateCorporation()
	u.SetCorporation(t.Context(), c)
	var emitted bool
	u.signals.CurrentCorporationExchanged.AddListener(func(_ context.Context, c *app.Corporation) {
		emitted = c == nil
	})
	// when
	u.ResetCorporation(t.Context())
	// then
	assert.Nil(t, u.CurrentCorporation())
	assert.True(t, emitted)
	xassert.Equal(t, 0, u.settings.LastCorporationID())
}

func TestBaseUI_LoadCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("sets loaded corporation", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCorporation()
		// when
		err := u.LoadCorporation(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, c.ID, u.CurrentCorporation().IDOrZero())
	})
	t.Run("returns error and keeps current corporation when not found", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCorporation()
		u.SetCorporation(t.Context(), c)
		// when
		err := u.LoadCorporation(t.Context(), c.ID+1)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
		assert.Same(t, c, u.CurrentCorporation())
	})
}

func TestBaseUI_SetAnyCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("sets a corporation", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCorporation()
		// when
		err := u.SetAnyCorporation(t.Context())
		// then
		require.NoError(t, err)
		xassert.Equal(t, c.ID, u.CurrentCorporation().IDOrZero())
	})
	t.Run("resets when there are no corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		u.corporation.Store(&app.Corporation{ID: 42})
		// when
		err := u.SetAnyCorporation(t.Context())
		// then
		require.NoError(t, err)
		assert.Nil(t, u.CurrentCorporation())
	})
}

func TestBaseUI_ShowCorporation(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("shows and loads another corporation", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c1 := factory.CreateCorporation()
		c2 := factory.CreateCorporation()
		u.SetCorporation(t.Context(), c1)
		var shown bool
		u.onShowCorporation = func() {
			shown = true
		}
		// when
		u.ShowCorporation(t.Context(), c2.ID)
		// then
		assert.True(t, shown)
		xassert.Equal(t, c2.ID, u.CurrentCorporation().IDOrZero())
	})
	t.Run("does not reload current corporation", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCorporation()
		u.SetCorporation(t.Context(), c)
		var shown bool
		u.onShowCorporation = func() {
			shown = true
		}
		// when
		u.ShowCorporation(t.Context(), c.ID)
		// then
		assert.True(t, shown)
		assert.Same(t, c, u.CurrentCorporation())
	})
	t.Run("keeps current corporation when load fails", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCorporation()
		u.SetCorporation(t.Context(), c)
		// when
		u.ShowCorporation(t.Context(), c.ID+1)
		// then
		assert.Same(t, c, u.CurrentCorporation())
	})
}

func TestBaseUI_HandleCharacterSectionChanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("reloads shown character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		c := factory.CreateCharacterFull()
		stale := &app.Character{ID: c.ID}
		u.character.Store(stale)
		// when
		u.handleCharacterSectionChanged(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkills,
		})
		// then
		assert.NotSame(t, stale, u.CurrentCharacter())
	})
	t.Run("does not reload when another character changed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		c := factory.CreateCharacterFull()
		other := factory.CreateCharacterFull()
		stale := &app.Character{ID: c.ID}
		u.character.Store(stale)
		// when
		u.handleCharacterSectionChanged(t.Context(), app.CharacterSectionUpdated{
			CharacterID: other.ID,
			Section:     app.SectionCharacterSkills,
		})
		// then
		assert.Same(t, stale, u.CurrentCharacter())
	})
	t.Run("updates mail indicator when mail headers changed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.settings.SetSysTrayEnabled(true)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID})
		var shown bool
		u.showMailIndicator = func() {
			shown = true
		}
		u.hideMailIndicator = func() {}
		// when
		u.handleCharacterSectionChanged(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterMailHeaders,
		})
		// then
		assert.True(t, shown)
	})
	t.Run("updates status when roles changed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		// NPC corporation, so no corporation update is started
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: 1_000_001})
		c := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
		var updated bool
		u.onUpdateStatus = func(_ context.Context) {
			updated = true
		}
		// when
		u.handleCharacterSectionChanged(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterRoles,
		})
		// then
		assert.True(t, updated)
	})
}

func TestBaseUI_HandleEveUniverseSectionChanged(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("reloads current character and updates status when it changed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		c := factory.CreateCharacterFull()
		stale := &app.Character{ID: c.ID}
		u.character.Store(stale)
		var updated bool
		u.onUpdateStatus = func(_ context.Context) {
			updated = true
		}
		// when
		u.handleEveUniverseSectionChanged(t.Context(), app.EveUniverseSectionUpdated{
			Changed: set.Of(c.ID),
			Section: app.SectionEveCharacters,
		})
		// then
		assert.NotSame(t, stale, u.CurrentCharacter())
		assert.True(t, updated)
	})
	t.Run("updates status when another own character changed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		c := factory.CreateCharacterFull()
		other := factory.CreateCharacterFull()
		stale := &app.Character{ID: c.ID}
		u.character.Store(stale)
		var updated bool
		u.onUpdateStatus = func(_ context.Context) {
			updated = true
		}
		// when
		u.handleEveUniverseSectionChanged(t.Context(), app.EveUniverseSectionUpdated{
			Changed: set.Of(other.ID),
			Section: app.SectionEveCharacters,
		})
		// then
		assert.Same(t, stale, u.CurrentCharacter())
		assert.True(t, updated)
	})
	t.Run("ignores characters that are not owned", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		factory.CreateCharacterFull()
		ec := factory.CreateEveCharacter()
		var updated bool
		u.onUpdateStatus = func(_ context.Context) {
			updated = true
		}
		// when
		u.handleEveUniverseSectionChanged(t.Context(), app.EveUniverseSectionUpdated{
			Changed: set.Of(ec.ID),
			Section: app.SectionEveCharacters,
		})
		// then
		assert.False(t, updated)
	})
	t.Run("updates status when a corporation of a character changed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		c := factory.CreateCharacterFull()
		var updated bool
		u.onUpdateStatus = func(_ context.Context) {
			updated = true
		}
		// when
		u.handleEveUniverseSectionChanged(t.Context(), app.EveUniverseSectionUpdated{
			Changed: set.Of(c.EveCharacter.Corporation.ID),
			Section: app.SectionEveCorporations,
		})
		// then
		assert.True(t, updated)
	})
	t.Run("ignores other corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		factory.CreateCharacterFull()
		other := factory.CreateEveEntityCorporation()
		var updated bool
		u.onUpdateStatus = func(_ context.Context) {
			updated = true
		}
		// when
		u.handleEveUniverseSectionChanged(t.Context(), app.EveUniverseSectionUpdated{
			Changed: set.Of(other.ID),
			Section: app.SectionEveCorporations,
		})
		// then
		assert.False(t, updated)
	})
}

func TestBaseUI_UpdateMailIndicator(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("shows indicator when there are unread mails", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.settings.SetSysTrayEnabled(true)
		factory.CreateCharacterMail()
		var shown, hidden bool
		u.showMailIndicator = func() {
			shown = true
		}
		u.hideMailIndicator = func() {
			hidden = true
		}
		// when
		u.UpdateMailIndicator(t.Context())
		// then
		assert.True(t, shown)
		assert.False(t, hidden)
	})
	t.Run("hides indicator when there are no unread mails", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.settings.SetSysTrayEnabled(true)
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{IsRead: optional.New(true)})
		var shown, hidden bool
		u.showMailIndicator = func() {
			shown = true
		}
		u.hideMailIndicator = func() {
			hidden = true
		}
		// when
		u.UpdateMailIndicator(t.Context())
		// then
		assert.False(t, shown)
		assert.True(t, hidden)
	})
	t.Run("does nothing when system tray is disabled", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.settings.SetSysTrayEnabled(false)
		factory.CreateCharacterMail()
		var called bool
		u.showMailIndicator = func() {
			called = true
		}
		u.hideMailIndicator = func() {
			called = true
		}
		// when
		u.UpdateMailIndicator(t.Context())
		// then
		assert.False(t, called)
	})
	t.Run("does nothing when indicator is not supported", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.settings.SetSysTrayEnabled(true)
		factory.CreateCharacterMail()
		// when
		u.UpdateMailIndicator(t.Context())
		// then no panic
	})
}

func TestBaseUI_SetCharacterSwitchMenu(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("lists characters with current one disabled", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		var cc []*app.Character
		for _, name := range []string{"Alpha", "Bravo", "Charlie"} {
			ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: name})
			cc = append(cc, factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID}))
		}
		u.character.Store(cc[1])
		avatar := theme.AccountIcon()
		u.avatarCache.Store(cc[0].ID, avatar) // avoids loading avatars in the background
		u.avatarCache.Store(cc[2].ID, avatar)
		var items []*fyne.MenuItem
		// when
		u.setCharacterSwitchMenu(t.Context(), func(x []*fyne.MenuItem) {
			items = x
		}, func() {})
		// then
		var labels []string
		var disabled []bool
		for _, it := range items {
			labels = append(labels, it.Label)
			disabled = append(disabled, it.Disabled)
		}
		xassert.Equal(t, []string{"Switch to...", "Alpha", "Bravo", "Charlie"}, labels)
		xassert.Equal(t, []bool{true, false, true, false}, disabled)
		xassert.Equal(t, avatar, items[1].Icon)
		xassert.Equal(t, avatar, items[3].Icon)
	})
	t.Run("loads character when item is tapped", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCharacterFull()
		u.avatarCache.Store(c.ID, theme.AccountIcon()) // avoids loading avatars in the background
		var items []*fyne.MenuItem
		u.setCharacterSwitchMenu(t.Context(), func(x []*fyne.MenuItem) {
			items = x
		}, func() {})
		exchanged := make(chan int64, 1)
		u.signals.CurrentCharacterExchanged.AddListener(func(_ context.Context, c *app.Character) {
			exchanged <- c.IDOrZero()
		})
		// when
		items[1].Action()
		// then
		select {
		case id := <-exchanged:
			xassert.Equal(t, c.ID, id)
		case <-time.After(time.Second):
			t.Fatal("character was not loaded")
		}
	})
	t.Run("sets no items when there are no characters", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		var called bool
		items := []*fyne.MenuItem{fyne.NewMenuItem("old", nil)}
		// when
		u.setCharacterSwitchMenu(t.Context(), func(x []*fyne.MenuItem) {
			called = true
			items = x
		}, func() {})
		// then
		assert.True(t, called)
		assert.Empty(t, items)
	})
}

func TestBaseUI_SetCorporationSwitchMenu(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	t.Run("lists corporations with current one disabled", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		var cc []*app.Corporation
		for _, name := range []string{"Alpha", "Bravo"} {
			e := factory.CreateEveEntityCorporation(app.EveEntity{Name: name})
			c := factory.CreateCorporation(e.ID)
			factory.CreateCorporationToken(c.ID, set.Of[app.Role](), set.Of[string]())
			cc = append(cc, c)
		}
		u.corporation.Store(cc[1])
		avatar := theme.AccountIcon()
		u.avatarCache.Store(cc[0].ID, avatar) // avoids loading avatars in the background
		var items []*fyne.MenuItem
		// when
		u.setCorporationSwitchMenu(t.Context(), func(x []*fyne.MenuItem) {
			items = x
		}, func() {})
		// then
		require.Len(t, items, 3)
		xassert.Equal(t, "Switch to...", items[0].Label)
		assert.True(t, items[0].Disabled)
		got := make(map[string]bool)
		for _, it := range items[1:] {
			got[it.Label] = it.Disabled
		}
		xassert.Equal(t, map[string]bool{"Alpha": false, "Bravo": true}, got)
		for _, it := range items[1:] {
			if it.Label == "Alpha" {
				xassert.Equal(t, avatar, it.Icon)
			}
		}
	})
	t.Run("loads corporation when item is tapped", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		c := factory.CreateCorporation()
		factory.CreateCorporationToken(c.ID, set.Of[app.Role](), set.Of[string]())
		u.avatarCache.Store(c.ID, theme.AccountIcon()) // avoids loading avatars in the background
		var items []*fyne.MenuItem
		u.setCorporationSwitchMenu(t.Context(), func(x []*fyne.MenuItem) {
			items = x
		}, func() {})
		exchanged := make(chan int64, 1)
		u.signals.CurrentCorporationExchanged.AddListener(func(_ context.Context, c *app.Corporation) {
			exchanged <- c.IDOrZero()
		})
		// when
		items[1].Action()
		// then
		select {
		case id := <-exchanged:
			xassert.Equal(t, c.ID, id)
		case <-time.After(time.Second):
			t.Fatal("corporation was not loaded")
		}
	})
	t.Run("sets no items when there are no corporations", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		var called bool
		items := []*fyne.MenuItem{fyne.NewMenuItem("old", nil)}
		// when
		u.setCorporationSwitchMenu(t.Context(), func(x []*fyne.MenuItem) {
			called = true
			items = x
		}, func() {})
		// then
		assert.True(t, called)
		assert.Empty(t, items)
	})
	t.Run("switches current corporation when it is no longer listed", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		u.signals = testdouble.NewSignalsSync()
		u.settings.SetHideLimitedCorporations(true)
		c := factory.CreateCorporation()
		factory.CreateCorporationTokenForSection(c.ID, app.CorporationSectionWalletJournal(app.Division1))
		var hidden *app.Corporation
		for range 4 {
			hidden = factory.CreateCorporation()
			factory.CreateCorporationToken(hidden.ID, set.Of[app.Role](), set.Of[string]())
		}
		u.corporation.Store(hidden)
		var called bool
		// when
		u.setCorporationSwitchMenu(t.Context(), func(_ []*fyne.MenuItem) {
			called = true
		}, func() {})
		// then
		assert.False(t, called)
		xassert.Equal(t, c.ID, u.CurrentCorporation().IDOrZero())
	})
}

func TestBaseUI_SetCharacterAvatarAsync(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	t.Cleanup(func() {
		db.Close()
	})
	t.Run("shows placeholder, then shows and caches loaded avatar", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		icons := make(chan fyne.Resource, 2)
		// when
		u.SetCharacterAvatarAsync(42, func(r fyne.Resource) {
			icons <- r
		})
		// then
		xassert.Equal(t, u.characterAvatarPlaceholder64, <-icons)
		select {
		case r := <-icons:
			cached, ok := u.avatarCache.Load(42)
			require.True(t, ok)
			xassert.Equal(t, cached, r)
		case <-time.After(time.Second):
			t.Fatal("avatar was not loaded")
		}
	})
	t.Run("uses cached avatar", func(t *testing.T) {
		u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
		avatar := theme.AccountIcon()
		u.avatarCache.Store(42, avatar)
		var got []fyne.Resource
		// when
		u.SetCharacterAvatarAsync(42, func(r fyne.Resource) {
			got = append(got, r)
		})
		// then
		xassert.Equal(t, []fyne.Resource{avatar}, got)
	})
}

func TestBaseUI_SetCorporationAvatarAsync(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	t.Cleanup(func() {
		db.Close()
	})
	u := MakeFakeBaseUI(t, st, test.NewTempApp(t))
	icons := make(chan fyne.Resource, 2)
	// when
	u.setCorporationAvatarAsync(42, func(r fyne.Resource) {
		icons <- r
	})
	// then
	xassert.Equal(t, u.corporationAvatarPlaceholder64, <-icons)
	select {
	case r := <-icons:
		cached, ok := u.avatarCache.Load(42)
		require.True(t, ok)
		xassert.Equal(t, cached, r)
	case <-time.After(time.Second):
		t.Fatal("avatar was not loaded")
	}
}
