// Package characterservice contains the EVE character service.
package characterservice

import (
	"context"
	"net/http"
	"time"

	"github.com/ErikKalkoken/eveauth"
	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/singleinstance"
)

type AuthClient interface {
	Authorize(ctx context.Context, scopes []string) (*eveauth.Token, error)
	RefreshToken(ctx context.Context, token *eveauth.Token) error
}

// CacheService defines a cache service
type CacheService interface {
	GetInt64(string) (int64, bool)
	SetInt64(string, int64, time.Duration)
}

// CharacterService provides access to all managed Eve Online characters both online and from local storage.
type CharacterService struct {
	authClient       AuthClient
	cache            CacheService
	concurrencyLimit int
	ens              *evenotification.EveNotificationService
	esiClient        *goesi.APIClient
	eus              *eveuniverseservice.EveUniverseService
	httpClient       *http.Client
	scs              *statuscacheservice.StatusCacheService
	sfg              *singleflight.Group
	sig              *singleinstance.Group
	st               *storage.Storage
}

type Params struct {
	ConcurrencyLimit       int // max number of concurrent Goroutines (per group)
	Cache                  CacheService
	EveNotificationService *evenotification.EveNotificationService
	EveUniverseService     *eveuniverseservice.EveUniverseService
	AuthClient             AuthClient
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
		authClient:       arg.AuthClient,
		cache:            arg.Cache,
		concurrencyLimit: -1, // Default is no limit
		ens:              arg.EveNotificationService,
		eus:              arg.EveUniverseService,
		scs:              arg.StatusCacheService,
		sfg:              new(singleflight.Group),
		sig:              singleinstance.NewGroup(),
		st:               arg.Storage,
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
