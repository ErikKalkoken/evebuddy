// Package characterservice contains the EVE character service.
package characterservice

import (
	"context"
	"net/http"

	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

type SSOService interface {
	Authenticate(ctx context.Context, scopes []string) (*app.Token, error)
	RefreshToken(ctx context.Context, refreshToken string) (*app.Token, error)
}

// CharacterService provides access to all managed Eve Online characters both online and from local storage.
type CharacterService struct {
	ens              *evenotification.EveNotificationService
	esiClient        *goesi.APIClient
	eus              *eveuniverseservice.EveUniverseService
	httpClient       *http.Client
	concurrencyLimit int
	scs              *statuscacheservice.StatusCacheService
	sfg              *singleflight.Group
	sso              SSOService
	st               *storage.Storage
}

type Params struct {
	ConcurrencyLimit       int // max number of concurrent Goroutines (per group)
	EveNotificationService *evenotification.EveNotificationService
	EveUniverseService     *eveuniverseservice.EveUniverseService
	SSOService             SSOService
	StatusCacheService     *statuscacheservice.StatusCacheService
	Storage                *storage.Storage
	// optional
	HTTPClient *http.Client
	ESIClient  *goesi.APIClient
}

// New creates a new character service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(arg Params) *CharacterService {
	s := &CharacterService{
		concurrencyLimit: -1, // Default is no limit
		ens:              arg.EveNotificationService,
		eus:              arg.EveUniverseService,
		scs:              arg.StatusCacheService,
		sso:              arg.SSOService,
		st:               arg.Storage,
		sfg:              new(singleflight.Group),
	}
	if arg.HTTPClient == nil {
		s.httpClient = http.DefaultClient
	} else {
		s.httpClient = arg.HTTPClient
	}
	if arg.ESIClient == nil {
		s.esiClient = goesi.NewAPIClient(s.httpClient, "")
	} else {
		s.esiClient = arg.ESIClient
	}
	if arg.ConcurrencyLimit > 0 {
		s.concurrencyLimit = arg.ConcurrencyLimit
	}
	return s
}

// DumpData returns the current content of the given SQL tables as JSON string.
// When no tables are given all tables will be included.
// This is a low-level method meant mainly for debugging.
func (s *CharacterService) DumpData(tables ...string) string {
	return s.st.DumpData(tables...)
}
