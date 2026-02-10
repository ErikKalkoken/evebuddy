package characterservice

import (
	"context"
	"net/http"
	"slices"

	"github.com/ErikKalkoken/eveauth"
	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func NewFake(st *storage.Storage, args ...Params) *CharacterService {
	scs := statuscacheservice.New(st)
	client := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
		UserAgent: "MyApp/1.0 (contact@example.com)",
	})
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		ESIClient:          client,
		StatusCacheService: scs,
		Storage:            st,
	})
	arg := Params{
		Cache:              testutil.NewCacheFake2(),
		ESIClient:          client,
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
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

type AuthClientFake struct {
	Token *eveauth.Token
	Err   error
}

func (s AuthClientFake) Authorize(ctx context.Context, scopes []string) (*eveauth.Token, error) {
	return s.Token, s.Err
}

func (s AuthClientFake) RefreshToken(ctx context.Context, token *eveauth.Token) error {
	token.AccessToken = s.Token.AccessToken
	token.RefreshToken = s.Token.RefreshToken
	token.ExpiresAt = s.Token.ExpiresAt
	return nil
}

func AuthTokenFromAppToken(x *app.Token) *eveauth.Token {
	return &eveauth.Token{
		AccessToken:   x.AccessToken,
		CharacterID:   int32(x.CharacterID),
		CharacterName: x.CharacterName,
		ExpiresAt:     x.ExpiresAt,
		RefreshToken:  x.RefreshToken,
		Scopes:        slices.Clone(x.Scopes),
		TokenType:     x.TokenType,
	}
}
