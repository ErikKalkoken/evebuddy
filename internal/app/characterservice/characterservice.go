// Package characterservice provides access to EVE Online characters.
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
	EveNotificationService *evenotification.EveNotificationService
	EveUniverseService     app.EveUniverseService
	StatusCacheService     app.StatusCacheService
	SSOService             *sso.SSOService

	esiClient  *goesi.APIClient
	httpClient *http.Client
	sfg        *singleflight.Group
	st         *storage.Storage
}

// New creates a new Characters service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(
	st *storage.Storage,
	httpClient *http.Client,
	esiClient *goesi.APIClient,

) *CharacterService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if esiClient == nil {
		esiClient = goesi.NewAPIClient(httpClient, "")
	}
	ct := &CharacterService{
		st:         st,
		esiClient:  esiClient,
		httpClient: httpClient,
		sfg:        new(singleflight.Group),
	}
	return ct
}
