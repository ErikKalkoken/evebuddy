package testdouble

import (
	"context"
	"net/http"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"
	"golang.org/x/oauth2"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

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

// NewCorporationService returns a fake for a CorporationService.
func NewCorporationService(args ...corporationservice.Params) *corporationservice.CorporationService {
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
		arg.StatusCacheService = new(statuscache.StatusCache)
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
		arg.EveUniverseService = eveuniverseservice.New(eveuniverseservice.Params{
			ESIClient:          arg.ESIClient,
			Signals:            arg.Signals,
			StatusCacheService: new(statuscache.StatusCache),
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
