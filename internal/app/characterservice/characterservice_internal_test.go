package characterservice

import (
	"context"
	"net/http"
	"time"

	"github.com/ErikKalkoken/eveauth"
	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type StatusCacheStub struct{}

func (c *StatusCacheStub) SetCharacterSection(o *app.CharacterSectionStatus) {
}

func (c *StatusCacheStub) UpdateCharacters(ctx context.Context, st statuscache.Storage) error {
	return nil
}

func (c *StatusCacheStub) UpdateCorporations(ctx context.Context, st statuscache.Storage) error {
	return nil
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

func NewFake(args ...Params) *CharacterService {
	var arg Params
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
		arg.EveUniverseService = eveuniverseservice.New(eveuniverseservice.Params{
			ESIClient:          arg.ESIClient,
			Signals:            arg.Signals,
			StatusCacheService: new(statuscache.StatusCache),
			Storage:            arg.Storage,
		})
	}
	if arg.Settings == nil {
		arg.Settings = new(testutil.SettingsStub)
	}
	s := New(arg)
	return s
}
