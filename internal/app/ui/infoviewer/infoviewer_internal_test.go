package infoviewer

import (
	"strings"

	"fyne.io/fyne/v2"
)

type UIServiceFake struct {
	app fyne.App
}

func (u *UIServiceFake) GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool) {
	return u.app.NewWindow("Dunmy"), true
}

func (u *UIServiceFake) HumanizeError(err error) string {
	return err.Error()
}

func (u *UIServiceFake) IsDeveloperMode() bool {
	return false
}

func (u *UIServiceFake) IsOffline() bool {
	return false
}

func (u *UIServiceFake) MainWindow() fyne.Window {
	return u.app.NewWindow("Dummy")
}

func (u *UIServiceFake) MakeWindowTitle(parts ...string) string {
	return strings.Join(parts, " - ")
}

type SettingsFake struct{}

func (s *SettingsFake) PreferMarketTab() bool {
	return false
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
