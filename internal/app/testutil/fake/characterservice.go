// Package fake provides fake implementations for services to use in tests.
package fake

import (
	"context"
	"net/http"
	"time"

	"github.com/ErikKalkoken/eveauth"
	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type EVENotificationServiceFake struct {
	entityIDs set.Set[int64]
	title     string
	body      string
	err       error
}

func (s *EVENotificationServiceFake) EntityIDs(nt app.EveNotificationType, text optional.Optional[string]) (set.Set[int64], error) {
	return s.entityIDs, s.err
}
func (s *EVENotificationServiceFake) RenderESI(ctx context.Context, nt app.EveNotificationType, text optional.Optional[string], timestamp time.Time) (title string, body string, err error) {
	return s.title, s.body, s.err
}

func NewCharacterService(args ...characterservice.Params) *characterservice.CharacterService {
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
	if arg.HTTPClient == nil {
		arg.HTTPClient = http.DefaultClient
	}
	if arg.ESIClient == nil {
		arg.ESIClient = goesi.NewESIClientWithOptions(arg.HTTPClient, goesi.ClientOptions{
			UserAgent: "MyApp/1.0 (contact@example.com)",
		})
	}
	if arg.EveNotificationService == nil {
		arg.EveNotificationService = &EVENotificationServiceFake{}
	}
	if arg.Signals == nil {
		arg.Signals = app.NewSignals()
	}
	if arg.StatusCacheService == nil {
		arg.StatusCacheService = new(statuscache.StatusCache)
	}
	if arg.EveUniverseService == nil {
		arg.EveUniverseService = eveuniverseservice.New(eveuniverseservice.Params{
			ESIClient:          arg.ESIClient,
			Signals:            arg.Signals,
			StatusCacheService: new(statuscache.StatusCache),
			Storage:            arg.Storage,
		})
	}
	if arg.SendDesktopNotification == nil {
		arg.SendDesktopNotification = func(title, content string) {
			panic("SendDesktopNotification not configured")
		}
	}
	if arg.Settings == nil {
		arg.Settings = new(testutil.SettingsFake)
	}
	s := characterservice.New(arg)
	return s
}
