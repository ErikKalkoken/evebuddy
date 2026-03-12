// Package testdouble provides test doubles of services.
//
// Doubles are in this package to circumvent import cycles
// which would occur when they were in the testutil package.
package testdouble

import (
	"context"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"github.com/ErikKalkoken/eveauth"
	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"
	"golang.org/x/oauth2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infoviewer"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func NewEVEUniverseServiceFake(args ...eveuniverseservice.Params) *eveuniverseservice.EVEUniverseService {
	var arg eveuniverseservice.Params
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.Storage == nil {
		panic("must define storage")
	}
	if arg.ESIClient == nil {
		arg.ESIClient = goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
			UserAgent: "MyApp/1.0 (contact@example.com)",
		})
	}
	if arg.Signals == nil {
		arg.Signals = app.NewSignals()
	}
	if arg.StatusCacheService == nil {
		arg.StatusCacheService = new(StatusCacheStub)
	}
	s := eveuniverseservice.New(arg)
	return s
}

type EVENotificationServiceStub struct {
	entityIDs set.Set[int64]
	title     string
	body      string
	err       error
}

func (s *EVENotificationServiceStub) EntityIDs(nt app.EveNotificationType, text optional.Optional[string]) (set.Set[int64], error) {
	return s.entityIDs, s.err
}
func (s *EVENotificationServiceStub) RenderESI(ctx context.Context, nt app.EveNotificationType, text optional.Optional[string], timestamp time.Time) (title string, body string, err error) {
	return s.title, s.body, s.err
}

// NewCharacterServiceFake returns a fake for a CharacterService.
func NewCharacterServiceFake(args ...characterservice.Params) *characterservice.CharacterService {
	var arg characterservice.Params
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.Storage == nil {
		panic("must define storage")
	}
	if arg.AuthClient == nil {
		ac, err := eveauth.NewClient(eveauth.Config{
			ClientID: "DUMMY",
			Port:     8000,
		})
		if err != nil {
			panic(err)
		}
		arg.AuthClient = ac
	}
	if arg.Cache == nil {
		arg.Cache = testutil.NewCacheFake2()
	}
	if arg.ESIClient == nil {
		var c *http.Client
		if arg.HTTPClient != nil {
			c = arg.HTTPClient
		} else {
			c = http.DefaultClient
		}
		arg.ESIClient = goesi.NewESIClientWithOptions(c, goesi.ClientOptions{
			UserAgent: "MyApp/1.0 (contact@example.com)",
		})
	}
	if arg.EveNotificationService == nil {
		arg.EveNotificationService = &EVENotificationServiceStub{}
	}
	if arg.Signals == nil {
		arg.Signals = app.NewSignals()
	}
	if arg.StatusCacheService == nil {
		arg.StatusCacheService = new(StatusCacheStub)
	}
	if arg.EveUniverseService == nil {
		arg.EveUniverseService = NewEVEUniverseServiceFake(eveuniverseservice.Params{
			ESIClient:          arg.ESIClient,
			Signals:            arg.Signals,
			StatusCacheService: new(StatusCacheStub),
			Storage:            arg.Storage,
		})
	}
	if arg.Settings == nil {
		arg.Settings = new(testutil.SettingsStub)
	}
	s := characterservice.New(arg)
	return s
}

type SettingsFake struct {
	MaxTransactions int
}

func (s *SettingsFake) MaxWalletTransactions() int {
	return s.MaxTransactions
}

type CharacterServiceFake struct {
	Token          *app.CharacterToken
	CorporationIDs set.Set[int64]
	Error          error
}

func (s *CharacterServiceFake) TokenSourceForCorporation(_ context.Context, _ int64, _ set.Set[app.Role], scopes set.Set[string]) (oauth2.TokenSource, int64, error) {
	if s.Error != nil {
		return &testutil.TokenSourceStub{CharacterToken: s.Token, Error: s.Error}, 0, nil
	}
	return &testutil.TokenSourceStub{CharacterToken: s.Token, Error: nil}, s.Token.CharacterID, nil
}

// NewCorporationServiceFake returns a fake for a CorporationService.
func NewCorporationServiceFake(args ...corporationservice.Params) *corporationservice.CorporationService {
	var arg corporationservice.Params
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.Storage == nil {
		panic("must define storage")
	}
	if arg.Cache == nil {
		arg.Cache = testutil.NewCacheFake2()
	}
	if arg.Signals == nil {
		arg.Signals = app.NewSignals()
	}
	if arg.StatusCacheService == nil {
		arg.StatusCacheService = new(StatusCacheStub)
	}
	if arg.ESIClient == nil {
		var c *http.Client
		if arg.HTTPClient != nil {
			c = arg.HTTPClient
		} else {
			c = http.DefaultClient
		}
		arg.ESIClient = goesi.NewESIClientWithOptions(c, goesi.ClientOptions{
			UserAgent: "MyApp/1.0 (contact@example.com)",
		})
	}
	if arg.EveUniverseService == nil {
		arg.EveUniverseService = NewEVEUniverseServiceFake(eveuniverseservice.Params{
			ESIClient:          arg.ESIClient,
			Signals:            arg.Signals,
			StatusCacheService: new(StatusCacheStub),
			Storage:            arg.Storage,
		})
	}
	if arg.Settings == nil {
		arg.Settings = new(SettingsFake)
	}
	if arg.CharacterService == nil {
		arg.CharacterService = new(CharacterServiceFake)
	}
	s := corporationservice.New(arg)
	return s
}

type StatusCacheStub struct{}

func (c *StatusCacheStub) SetCharacterSection(o *app.CharacterSectionStatus) {}

func (c *StatusCacheStub) SetCorporationSection(o *app.CorporationSectionStatus) {}

func (c *StatusCacheStub) SetEveUniverseSection(o *app.EveUniverseSectionStatus) {}

func (c *StatusCacheStub) UpdateCharacters(ctx context.Context, st statuscache.Storage) error {
	return nil
}

func (c *StatusCacheStub) UpdateCorporations(ctx context.Context, st statuscache.Storage) error {
	return nil
}

type UIFake struct {
	app               fyne.App
	cs                *characterservice.CharacterService
	rs                *corporationservice.CorporationService
	eis               ui.EVEImageService
	eus               *eveuniverseservice.EVEUniverseService
	isMobile          bool
	iw                *infoviewer.InfoViewer
	signals           *app.Signals
	showCharacterFunc func(ctx context.Context, characterID int64)
	showSnackbarFunc  func(text string)
}

type UIParams struct {
	App               fyne.App
	IsMobile          bool
	Signals           *app.Signals
	Storage           *storage.Storage
	ShowCharacterFunc func(ctx context.Context, characterID int64)
	ShowSnackbarFunc  func(text string)
}

func NewUIFake(args ...UIParams) *UIFake {
	var arg UIParams
	if len(args) > 0 {
		arg = args[0]
	}
	if arg.Storage == nil {
		panic("must define storage")
	}
	if arg.App == nil {
		panic("must define app")
	}
	if arg.Signals == nil {
		arg.Signals = app.NewSignals()
	}
	esiClient := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
		UserAgent: "MyApp/1.0 (contact@example.com)",
	})
	scs := new(StatusCacheStub)
	eus := NewEVEUniverseServiceFake(eveuniverseservice.Params{
		Storage:            arg.Storage,
		Signals:            arg.Signals,
		ESIClient:          esiClient,
		StatusCacheService: scs,
	})
	cs := NewCharacterServiceFake(characterservice.Params{
		Storage:            arg.Storage,
		EveUniverseService: eus,
		Signals:            arg.Signals,
		ESIClient:          esiClient,
		StatusCacheService: scs,
	})
	rs := NewCorporationServiceFake(corporationservice.Params{
		CharacterService:   cs,
		ESIClient:          esiClient,
		EveUniverseService: eus,
		Signals:            arg.Signals,
		StatusCacheService: scs,
		Storage:            arg.Storage,
	})
	u := &UIFake{
		app:               arg.App,
		cs:                cs,
		eis:               testutil.NewEveImageServiceStub(),
		eus:               eus,
		isMobile:          arg.IsMobile,
		rs:                rs,
		showCharacterFunc: arg.ShowCharacterFunc,
		showSnackbarFunc:  arg.ShowSnackbarFunc,
		signals:           arg.Signals,
	}
	return u
}

func (u *UIFake) Character() *characterservice.CharacterService {
	return u.cs
}

func (u *UIFake) Corporation() *corporationservice.CorporationService {
	return u.rs
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
	return u.app.NewWindow("Dummy"), true
}

func (u *UIFake) GetOrCreateWindowWithOnClosed(id string, titles ...string) (window fyne.Window, created bool, onClosed func()) {
	return u.app.NewWindow("Dummy"), true, func() {}
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
	return u.app.NewWindow("Dummy")
}

func (u *UIFake) ShowCharacter(ctx context.Context, characterID int64) {
	if f := u.showCharacterFunc; f != nil {
		f(ctx, characterID)
	}
}

func (u *UIFake) ShowSnackbar(text string) {
	if f := u.showSnackbarFunc; f != nil {
		f(text)
	}
}

func (u *UIFake) Signals() *app.Signals {
	return u.signals
}
