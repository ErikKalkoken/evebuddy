package infowindow

import (
	"net/http"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
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

func (u *UIServiceFake) MakeWindowTitle(parts ...string) string {
	return strings.Join(parts, " - ")
}

func (u *UIServiceFake) ShowInformationDialog(title, message string, parent fyne.Window) {}

type SettingsFake struct{}

func (s *SettingsFake) PreferMarketTab() bool {
	return false
}

func TestInfoWindow_CanRenderLocationInfo(t *testing.T) {
	test.ApplyTheme(t, test.Theme())
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	esiClient := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
		UserAgent: "EveBuddy/1.0 (test@kalkoken.net)",
	})
	scs := statuscacheservice.New(st)
	if err := scs.InitCache(t.Context()); err != nil {
		panic(err)
	}
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		ESIClient:          esiClient,
		StatusCacheService: scs,
		Storage:            st,
	})
	cs := characterservice.New(characterservice.Params{
		Cache:              testutil.NewCacheFake2(),
		ESIClient:          esiClient,
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
	})
	app := test.NewTempApp(t)
	makeInfoWindow := func() *InfoWindow {
		return New(Params{
			CharacterService: cs,
			EveImageService: &testutil.EveImageServiceFake{
				Character:   icons.Characterplaceholder64Jpeg,
				Alliance:    icons.Corporationplaceholder64Png,
				Corporation: icons.Corporationplaceholder64Png,
				Err:         nil,
				Faction:     icons.Factionplaceholder64Png,
				Type:        icons.Typeplaceholder64Png,
			},
			EveUniverseService: eus,
			IsMobile:           false,
			JaniceService:      new(janiceservice.JaniceService),
			StatusCacheService: scs,
			Settings:           new(SettingsFake),
			UIService:          new(UIServiceFake),
			Window:             app.NewWindow("Dummy"),
		})
	}
	t.Run("can render full location", func(t *testing.T) {
		l := factory.CreateEveLocationStation()
		iw := makeInfoWindow()
		a := newLocationInfo(iw, l.ID)
		a.update(t.Context())
		test.RenderObjectToMarkup(a)
	})
	t.Run("can render minimal location", func(t *testing.T) {
		l := factory.CreateEveLocationEmptyStructure()
		iw := makeInfoWindow()
		a := newLocationInfo(iw, l.ID)
		a.update(t.Context())
		test.RenderObjectToMarkup(a)
	})
}
