package characterservice

import (
	"context"
	"net/http"
	"slices"
	"time"

	"github.com/ErikKalkoken/eveauth"
	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

type SettingsFake struct {
	MaxWalletTransactionsDefault    int
	MaxMailsDefault                 int
	MarketOrderRetentionDaysDefault int
}

func (s *SettingsFake) MaxMails() int {
	return s.MaxMailsDefault
}

func (s *SettingsFake) MaxWalletTransactions() int {
	return s.MaxWalletTransactionsDefault
}

func (s *SettingsFake) MarketOrderRetentionDays() int {
	return s.MarketOrderRetentionDaysDefault
}

func (s *SettingsFake) NotificationTypesEnabled() set.Set[string] { return set.Set[string]{} }
func (s *SettingsFake) NotifyCommunicationsEarliest() time.Time   { return time.Now() }
func (s *SettingsFake) NotifyCommunicationsEnabled() bool         { return true }
func (s *SettingsFake) NotifyContractsEarliest() time.Time        { return time.Now() }
func (s *SettingsFake) NotifyContractsEnabled() bool              { return true }
func (s *SettingsFake) NotifyMailsEarliest() time.Time            { return time.Now() }
func (s *SettingsFake) NotifyMailsEnabled() bool                  { return true }
func (s *SettingsFake) NotifyPIEnabled() bool                     { return true }
func (s *SettingsFake) NotifyTrainingEnabled() bool               { return true }
func (s *SettingsFake) NotifyPIEarliest() time.Time               { return time.Now() }

func NewFake(st *storage.Storage, args ...Params) *CharacterService {
	scs := statuscacheservice.New(st)
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
		arg.Settings = new(SettingsFake)
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
