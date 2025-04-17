// Package characterservice contains the EVE character service.
package characterservice

import (
	"net/http"

	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/sso"
	"github.com/antihax/goesi"
)

// CharacterService provides access to all managed Eve Online characters both online and from local storage.
type CharacterService struct {
	ens        *evenotification.EveNotificationService
	esiClient  *goesi.APIClient
	eus        app.EveUniverseService
	httpClient *http.Client
	scs        app.StatusCacheService
	sfg        *singleflight.Group
	sso        *sso.SSOService
	st         *storage.Storage
}

type Params struct {
	EveNotificationService *evenotification.EveNotificationService
	EveUniverseService     app.EveUniverseService
	SSOService             *sso.SSOService
	StatusCacheService     app.StatusCacheService
	Storage                *storage.Storage
	// optional
	HttpClient *http.Client
	EsiClient  *goesi.APIClient
}

// New creates a new Characters service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(args Params) *CharacterService {
	s := &CharacterService{
		ens: args.EveNotificationService,
		eus: args.EveUniverseService,
		scs: args.StatusCacheService,
		sso: args.SSOService,
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
