// Package character contains the character service.
package character

import (
	"context"
	"errors"
	"net/http"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"
)

var (
	ErrAborted  = errors.New("process aborted prematurely")
	ErrNotFound = errors.New("object not found")
)

type DictionaryService interface {
	IntWithFallback(string, int) (int, error)
	Float64(key string) (float64, bool, error)
	SetFloat64(key string, value float64) error
}

type StatusCacheService interface {
	UpdateCharacters(ctx context.Context, r app.StatusCacheStorage) error
	CharacterSectionSet(*app.CharacterSectionStatus)
}

// CharacterService provides access to all managed Eve Online characters both online and from local storage.
type CharacterService struct {
	DictionaryService  DictionaryService
	EveUniverseService *eveuniverse.EveUniverseService
	StatusCacheService StatusCacheService

	esiClient  *goesi.APIClient
	httpClient *http.Client
	sfg        *singleflight.Group
	st         *sqlite.Storage
}

// New creates a new Characters service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(
	st *sqlite.Storage,
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
