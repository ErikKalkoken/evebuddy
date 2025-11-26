package characterservice

import (
	"context"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	"github.com/antihax/goesi"
)

func NewFake(st *storage.Storage, args ...Params) *CharacterService {
	scs := statuscacheservice.New(memcache.New(), st)
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		ESIClient:          goesi.NewAPIClient(nil, ""),
		StatusCacheService: scs,
		Storage:            st,
	})
	arg := Params{
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
	}
	if len(args) > 0 {
		a := args[0]
		if a.SSOService != nil {
			arg.SSOService = a.SSOService
		}
	}
	s := New(arg)
	return s
}

type SSOFake struct {
	Token *app.Token
	Err   error
}

func (s SSOFake) Authenticate(ctx context.Context, scopes []string) (*app.Token, error) {
	return s.Token, s.Err
}

func (s SSOFake) RefreshToken(ctx context.Context, refreshToken string) (*app.Token, error) {
	return s.Token, s.Err
}
