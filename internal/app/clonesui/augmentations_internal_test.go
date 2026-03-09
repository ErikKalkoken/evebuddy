package clonesui

import (
	"net/http"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/infowindow"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
)

func TestAugmentations_CanRenderWithData(t *testing.T) {
	if testing.Short() {
		t.Skip("UI tests are flakey")
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
		Name: "Bruce Wayne",
		ID:   42,
	})
	character := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
	et := factory.CreateEveType(storage.CreateEveTypeParams{
		Name: "Dummy Implant",
	})
	da := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{ID: app.EveDogmaAttributeImplantSlot})
	factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
		DogmaAttributeID: da.ID,
		EveTypeID:        et.ID,
		Value:            3,
	})
	factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{
		CharacterID: character.ID,
		TypeID:      et.ID,
	})
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character.ID,
		Section:     app.SectionCharacterImplants,
	})
	test.ApplyTheme(t, test.Theme())

	a := NewAugmentations(newUIFake(st, test.NewTempApp(t)))
	w := test.NewWindow(a)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 300))

	a.Update(t.Context())
	time.Sleep(50 * time.Millisecond)

	a.tree.OpenAllBranches()
	test.AssertImageMatches(t, "augmentations/master.png", w.Canvas().Capture())
}

type uiFake struct {
	a   fyne.App
	cs  *characterservice.CharacterService
	eis *eveimageservice.EVEImageService
	eus *eveuniverseservice.EVEUniverseService
	iw  *infowindow.InfoWindow
	scs *statuscache.StatusCache
	sig *app.Signals
}

func newUIFake(st *storage.Storage, a fyne.App) *uiFake {
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
	cs := characterservice.New(characterservice.Params{
		AuthClient:             testutil.AuthClientFake{},
		Cache:                  testutil.NewCacheFake2(),
		ESIClient:              client,
		EveNotificationService: evenotification.New(eus),
		EveUniverseService:     eus,
		Settings:               &testutil.SettingsFake{},
		Signals:                signals,
		StatusCacheService:     scs,
		Storage:                st,
	})
	u := &uiFake{
		a:   a,
		cs:  cs,
		eis: eveimageservice.New(testutil.NewCacheFake(), http.DefaultClient, true),
		eus: eus,
		scs: scs,
		sig: signals,
	}
	return u
}

func (u *uiFake) Character() *characterservice.CharacterService {
	return u.cs
}

func (u *uiFake) ErrorDisplay(err error) string {
	return err.Error()
}

func (u *uiFake) EVEImage() *eveimageservice.EVEImageService {
	return u.eis
}

func (u *uiFake) EVEUniverse() *eveuniverseservice.EVEUniverseService {
	return u.eus
}

func (u *uiFake) GetOrCreateWindow(id string, titles ...string) (window fyne.Window, created bool) {
	return u.a.NewWindow("Dummy"), true
}
func (u *uiFake) InfoWindow() *infowindow.InfoWindow {
	return u.iw
}

func (u *uiFake) IsDeveloperMode() bool {
	return false
}

func (u *uiFake) IsMobile() bool {
	return false
}

func (u *uiFake) IsOffline() bool {
	return true
}

func (u *uiFake) MainWindow() fyne.Window {
	return u.a.NewWindow("Dummy")
}
func (u *uiFake) ShowSnackbar(text string) {

}

func (u *uiFake) Signals() *app.Signals {
	return u.sig
}

func (u *uiFake) StatusCache() *statuscache.StatusCache {
	return u.scs
}
