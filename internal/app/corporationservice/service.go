package corporationservice

import (
	"context"
	"net/http"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi/esi"
	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

type CharacterService interface {
	TokenSourceForCorporation(ctx context.Context, corporationID int64, roles set.Set[app.Role], scopes set.Set[string]) (oauth2.TokenSource, int64, error)
}

type Settings interface {
	MaxWalletTransactions() int
}

// Cache defines a cache.
type Cache interface {
	GetInt64(string) (int64, bool)
	SetInt64(string, int64, time.Duration)
}

// CorporationService provides access to all managed Eve Online corporations both online and from local storage.
type CorporationService struct {
	cache            Cache
	concurrencyLimit int
	cs               CharacterService
	esiClient        *esi.APIClient
	eus              *eveuniverseservice.EVEUniverseService
	httpClient       *http.Client
	scs              *statuscache.StatusCache
	settings         Settings
	sfg              singleflight.Group
	signals          *app.Signals
	st               *storage.Storage
}

type Params struct {
	Cache              Cache
	CharacterService   CharacterService
	ConcurrencyLimit   int // max number of concurrent Goroutines (per group)
	ESIClient          *esi.APIClient
	EveUniverseService *eveuniverseservice.EVEUniverseService
	Settings           Settings
	Signals            *app.Signals
	StatusCacheService *statuscache.StatusCache
	Storage            *storage.Storage
	// optional
	HTTPClient *http.Client
}

// New creates a new corporation service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(arg Params) *CorporationService {
	if arg.Cache == nil {
		panic("Cache missing")
	}
	if arg.CharacterService == nil {
		panic("CharacterService missing")
	}
	if arg.ESIClient == nil {
		panic("ESIClient missing")
	}
	if arg.EveUniverseService == nil {
		panic("EveUniverseService missing")
	}
	if arg.Settings == nil {
		panic("Settings missing")
	}
	if arg.Signals == nil {
		panic("Signals missing")
	}
	if arg.StatusCacheService == nil {
		panic("StatusCacheService missing")
	}
	if arg.Storage == nil {
		panic("Storage missing")
	}
	s := &CorporationService{
		cache:            arg.Cache,
		concurrencyLimit: -1, // Default is no limit
		cs:               arg.CharacterService,
		esiClient:        arg.ESIClient,
		eus:              arg.EveUniverseService,
		scs:              arg.StatusCacheService,
		settings:         arg.Settings,
		signals:          arg.Signals,
		st:               arg.Storage,
	}
	if arg.HTTPClient == nil {
		s.httpClient = http.DefaultClient
	} else {
		s.httpClient = arg.HTTPClient
	}
	if arg.ConcurrencyLimit > 0 {
		s.concurrencyLimit = arg.ConcurrencyLimit
	}
	return s
}
