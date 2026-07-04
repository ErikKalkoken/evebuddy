package infoviewer

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
)

// UIServiceFake is a minimal coreUI implementation for unit tests.
type UIServiceFake struct {
	app        fyne.App
	mainWindow fyne.Window
}

func newUIServiceFake(a fyne.App) *UIServiceFake {
	return &UIServiceFake{app: a, mainWindow: a.NewWindow("Main")}
}

func (u *UIServiceFake) Character() *characterservice.CharacterService       { return nil }
func (u *UIServiceFake) EVEImage() ui.EVEImageService                        { return nil }
func (u *UIServiceFake) EVEUniverse() *eveuniverseservice.EVEUniverseService { return nil }
func (u *UIServiceFake) GetOrCreateWindow(id string, titles ...string) (fyne.Window, bool) {
	return u.app.NewWindow(strings.Join(titles, " - ")), true
}
func (u *UIServiceFake) GetOrCreateWindowWithOnClosed(id string, titles ...string) (fyne.Window, bool, func()) {
	return u.app.NewWindow(strings.Join(titles, " - ")), true, func() {}
}
func (u *UIServiceFake) ErrorDisplay(err error) string        { return err.Error() }
func (u *UIServiceFake) IsDeveloperMode() bool                { return false }
func (u *UIServiceFake) IsMobile() bool                       { return false }
func (u *UIServiceFake) IsOffline() bool                      { return false }
func (u *UIServiceFake) Janice() *janiceservice.JaniceService { return nil }
func (u *UIServiceFake) MainWindow() fyne.Window              { return u.mainWindow }
func (u *UIServiceFake) Settings() *settings.Settings         { return nil }

var _ coreUI = (*UIServiceFake)(nil)

// TestInfoViewer_show2_UsesMainWindowAsFallbackWhenInfoWindowClosed is a regression test
// for the bug where show2 used iw.w directly as the dialog parent. After an info window is
// opened and then closed, its OnClosed handler sets iw.w = nil. Any subsequent show2 call
// (e.g. tapping an unsupported entity type) would then pass nil to a Fyne dialog, causing
// a nil-pointer panic. The fix adds a fallback to iw.u.MainWindow() when iw.w is nil.
func TestInfoViewer_show2_UsesMainWindowAsFallbackWhenInfoWindowClosed(t *testing.T) {
	a := test.NewTempApp(t)
	// iw.w is nil here, simulating the state after an info window was opened and closed.
	iw := &InfoViewer{
		u: newUIServiceFake(a),
		w: nil,
	}
	// entityID=0 takes the early-return error path which passes parentW to a dialog.
	// Before the fix, parentW was iw.w (nil) here, causing a panic.
	// After the fix, parentW falls back to iw.u.MainWindow(), which is always valid.
	assert.NotPanics(t, func() {
		iw.show2(showParams{variant: infoNotSupported, entityID: 0})
	})
}

// FIXME

// func TestInfoWindow_CanRenderLocationInfo(t *testing.T) {
// 	test.ApplyTheme(t, test.Theme())
// 	db, st, factory := testutil.NewDBOnDisk(t)
// 	defer db.Close()

// 	esiClient := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
// 		UserAgent: "EveBuddy/1.0 (test@kalkoken.net)",
// 	})
// 	scs := new(statuscacheservice.StatusCacheService)
// 	if err := scs.InitCache(t.Context()); err != nil {
// 		panic(err)
// 	}
// 	signals := app.NewSignals()
// 	eus := eveuniverseservice.New(eveuniverseservice.Params{
// 		ESIClient:          esiClient,
// 		Signals:            signals,
// 		StatusCacheService: scs,
// 		Storage:            st,
// 	})
// 	ac, err := eveauth.NewClient(eveauth.Config{
// 		ClientID: "DUMMY",
// 		Port:     8000,
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
// 	app := test.NewTempApp(t)
// 	cs := characterservice.New(characterservice.Params{
// 		AuthClient:             ac,
// 		Cache:                  testutil.NewCacheFake2(),
// 		ESIClient:              esiClient,
// 		EveNotificationService: evenotification.New(eus),
// 		EveUniverseService:     eus,
// 		Settings:               settings.New(app.Preferences()),
// 		Signals:                signals,
// 		StatusCacheService:     scs,
// 		Storage:                st,
// 	})
// 	makeInfoWindow := func() *InfoViewer {
// 		iw := New(&UIServiceFake{app: app})
// 		return iw
// 	}
// 	t.Run("can render full location", func(t *testing.T) {
// 		l := factory.CreateEveLocationStation()
// 		iw := makeInfoWindow()
// 		a := newLocationInfo(iw, l.ID)
// 		a.update(t.Context())
// 		test.RenderObjectToMarkup(a)
// 	})
// 	t.Run("can render minimal location", func(t *testing.T) {
// 		l := factory.CreateEveLocationEmptyStructure()
// 		iw := makeInfoWindow()
// 		a := newLocationInfo(iw, l.ID)
// 		a.update(t.Context())
// 		test.RenderObjectToMarkup(a)
// 	})
// }
