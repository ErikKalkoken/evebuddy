// Package character contains the character service.
package character

import (
	"errors"
	"net/http"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/sso"
	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"
)

var (
	ErrAborted  = errors.New("process aborted prematurely")
	ErrNotFound = errors.New("object not found")
)

// CharacterService provides access to all managed Eve Online characters both online and from local storage.
type CharacterService struct {
	EveUniverseService *eveuniverse.EveUniverseService
	StatusCacheService app.StatusCacheService
	SSOService         *sso.SSOService

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
