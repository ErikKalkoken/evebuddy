package corporationservice

import (
	"context"
	"net/http"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"
)

// CorporationService provides access to all managed Eve Online corporations both online and from local storage.
type CorporationService struct {
	esiClient  *goesi.APIClient
	eus        *eveuniverseservice.EveUniverseService
	httpClient *http.Client
	scs        *statuscacheservice.StatusCacheService
	sfg        *singleflight.Group
	st         *storage.Storage
}

type Params struct {
	EveUniverseService *eveuniverseservice.EveUniverseService
	StatusCacheService *statuscacheservice.StatusCacheService
	Storage            *storage.Storage
	// optional
	HttpClient *http.Client
	EsiClient  *goesi.APIClient
}

// New creates a new corporation service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(args Params) *CorporationService {
	s := &CorporationService{
		eus: args.EveUniverseService,
		scs: args.StatusCacheService,
		st:  args.Storage,
		sfg: new(singleflight.Group),
	}
	if args.HttpClient == nil {
		s.httpClient = http.DefaultClient
	} else {
		s.httpClient = args.HttpClient
	}
	if args.EsiClient == nil {
		s.esiClient = goesi.NewAPIClient(s.httpClient, "")
	} else {
		s.esiClient = args.EsiClient
	}
	return s
}

func (s *CorporationService) GetOrCreateCorporation(ctx context.Context, corporationID int32) (*app.Corporation, error) {
	return s.st.GetOrCreateCorporation(ctx, corporationID)
}
