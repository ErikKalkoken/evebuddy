package clones

import (
	"net/http"

	"fyne.io/fyne/v2"
	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infoviewer"
)

type UIFake struct {
	a        fyne.App
	cs       *characterservice.CharacterService
	eis      ui.EVEImageService
	eus      *eveuniverseservice.EVEUniverseService
	iw       *infoviewer.InfoViewer
	scs      *statuscache.StatusCache
	sig      *app.Signals
	isMobile bool
}

func NewUIFake(st *storage.Storage, a fyne.App) *UIFake {
	scs := new(statuscache.StatusCache)
	client := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
		UserAgent: "MyApp/1.0 (contact@example.com)",
	})
	signals := app.NewSignals()
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		ESIClient:          client,
		Signals:            signals,
		StatusCacheService: scs,
		Storage:            st,
	})
	u := &UIFake{
		a:   a,
		cs:  testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st, EveUniverseService: eus, StatusCacheService: scs, Signals: signals}),
		eis: testutil.NewEveImageServiceStub(),
		eus: eus,
		scs: scs,
		sig: signals,
	}
	return u
}

func (u *UIFake) Character() *characterservice.CharacterService {
	return u.cs
}

func (u *UIFake) ErrorDisplay(err error) string {
	return err.Error()
}

func (u *UIFake) EVEImage() ui.EVEImageService {
	return u.eis
}

func (u *UIFake) EVEUniverse() *eveuniverseservice.EVEUniverseService {
	return u.eus
}

func (u *UIFake) GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool) {
	return u.a.NewWindow("Dummy"), true
}
func (u *UIFake) InfoViewer() ui.InfoViewer {
	return u.iw
}

func (u *UIFake) IsDeveloperMode() bool {
	return false
}

func (u *UIFake) IsMobile() bool {
	return u.isMobile
}

func (u *UIFake) IsOffline() bool {
	return true
}

func (u *UIFake) MainWindow() fyne.Window {
	return u.a.NewWindow("Dummy")
}
func (u *UIFake) ShowSnackbar(text string) {

}

func (u *UIFake) Signals() *app.Signals {
	return u.sig
}

func (u *UIFake) StatusCache() *statuscache.StatusCache {
	return u.scs
}
