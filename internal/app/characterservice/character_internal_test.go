package characterservice

import (
	"context"
	"slices"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/eveauth"
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
		TickerSource:       &testutil.FakeTicker{},
	}
	if len(args) > 0 {
		a := args[0]
		if a.AuthClient != nil {
			arg.AuthClient = a.AuthClient
		}
	}
	s := New(arg)
	return s
}

type SSOServiceFake struct {
	Token *app.Token
	Err   error
}

func (s SSOServiceFake) Authenticate(ctx context.Context, scopes []string) (*eveauth.Token, error) {
	return ssoTokenFromApp(s.Token), s.Err
}

func (s SSOServiceFake) RefreshToken(ctx context.Context, token *eveauth.Token) error {
	t2 := ssoTokenFromApp(s.Token)
	token.AccessToken = t2.AccessToken
	token.RefreshToken = t2.RefreshToken
	token.ExpiresAt = t2.ExpiresAt
	return nil
}

func ssoTokenFromApp(x *app.Token) *eveauth.Token {
	return &eveauth.Token{
		AccessToken:   x.AccessToken,
		CharacterID:   x.CharacterID,
		CharacterName: x.CharacterName,
		ExpiresAt:     x.ExpiresAt,
		RefreshToken:  x.RefreshToken,
		Scopes:        slices.Clone(x.Scopes),
		TokenType:     x.TokenType,
	}
}
