package characterservice

import (
	"net/http"

	"github.com/ErikKalkoken/eveauth"
	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func NewFake(st *storage.Storage, args ...Params) *CharacterService {
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
	arg := Params{
		Cache:                  testutil.NewCacheFake2(),
		ESIClient:              client,
		EveNotificationService: evenotification.New(eus),
		EveUniverseService:     eus,
		Signals:                signals,
		StatusCacheService:     scs,
		Storage:                st,
	}
	if len(args) > 0 {
		a := args[0]
		if a.AuthClient != nil {
			arg.AuthClient = a.AuthClient
		}
		if a.Settings != nil {
			arg.Settings = a.Settings
		}
		if a.SendDesktopNotification != nil {
			arg.SendDesktopNotification = a.SendDesktopNotification
		}
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
	if arg.Settings == nil {
		arg.Settings = new(testutil.SettingsFake)
	}
	s := New(arg)
	return s
}
