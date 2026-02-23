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
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

type CharacterService interface {
	TokenSourceForCorporation(ctx context.Context, corporationID int64, roles set.Set[app.Role], scopes set.Set[string]) (oauth2.TokenSource, int64, error)
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
	eus              *eveuniverseservice.EveUniverseService
	httpClient       *http.Client
	scs              *statuscacheservice.StatusCacheService
	sfg              singleflight.Group
	st               *storage.Storage
}

type Params struct {
	Cache              Cache
	CharacterService   CharacterService
	ConcurrencyLimit   int // max number of concurrent Goroutines (per group)
	ESIClient          *esi.APIClient
	EveUniverseService *eveuniverseservice.EveUniverseService
	StatusCacheService *statuscacheservice.StatusCacheService
	Storage            *storage.Storage
	// optional
	HTTPClient *http.Client
}

// New creates a new corporation service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(arg Params) *CorporationService {
	if arg.ESIClient == nil {
		panic("must provide esi client")
	}
	s := &CorporationService{
		cache:            arg.Cache,
		concurrencyLimit: -1, // Default is no limit
		cs:               arg.CharacterService,
		esiClient:        arg.ESIClient,
		eus:              arg.EveUniverseService,
		scs:              arg.StatusCacheService,
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
