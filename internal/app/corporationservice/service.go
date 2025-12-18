package corporationservice

import (
	"context"
	"net/http"

	"github.com/ErikKalkoken/kx/set"
	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

type CharacterService interface {
	CharacterTokenForCorporation(ctx context.Context, corporationID int32, roles set.Set[app.Role], scopes set.Set[string], checkToken bool) (*app.CharacterToken, error)
}

// CorporationService provides access to all managed Eve Online corporations both online and from local storage.
type CorporationService struct {
	concurrencyLimit int
	cs               CharacterService
	esiClient        *goesi.APIClient
	eus              *eveuniverseservice.EveUniverseService
	httpClient       *http.Client
	scs              *statuscacheservice.StatusCacheService
	sfg              *singleflight.Group
	st               *storage.Storage
}

type Params struct {
	CharacterService   CharacterService
	ConcurrencyLimit   int // max number of concurrent Goroutines (per group)
	EveUniverseService *eveuniverseservice.EveUniverseService
	StatusCacheService *statuscacheservice.StatusCacheService
	Storage            *storage.Storage
	// optional
	HTTPClient *http.Client
	EsiClient  *goesi.APIClient
}

// New creates a new corporation service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(arg Params) *CorporationService {
	s := &CorporationService{
		concurrencyLimit: -1, // Default is no limit
		cs:               arg.CharacterService,
		eus:              arg.EveUniverseService,
		scs:              arg.StatusCacheService,
		st:               arg.Storage,
		sfg:              new(singleflight.Group),
	}
	if arg.HTTPClient == nil {
		s.httpClient = http.DefaultClient
	} else {
		s.httpClient = arg.HTTPClient
	}
	if arg.EsiClient == nil {
		s.esiClient = goesi.NewAPIClient(s.httpClient, "")
	} else {
		s.esiClient = arg.EsiClient
	}
	if arg.ConcurrencyLimit > 0 {
		s.concurrencyLimit = arg.ConcurrencyLimit
	}
	return s
}
