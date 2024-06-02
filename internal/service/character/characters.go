// Package character contains the character service.
package character

import (
	"errors"
	"net/http"

	"github.com/ErikKalkoken/evebuddy/internal/helper/cache"
	"github.com/ErikKalkoken/evebuddy/internal/service/characterstatus"
	"github.com/ErikKalkoken/evebuddy/internal/service/dictionary"
	"github.com/ErikKalkoken/evebuddy/internal/service/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"
)

var (
	ErrAborted  = errors.New("process aborted prematurely")
	ErrNotFound = errors.New("object not found")
)

// CharacterService provides access to all managed Eve Online characters both online and from local storage.
type CharacterService struct {
	esiClient  *goesi.APIClient
	httpClient *http.Client
	sfg        *singleflight.Group
	// Storage service
	st *storage.Storage
	// EveUniverse service
	eu *eveuniverse.EveUniverseService
	// CharacterStatus service
	cs *characterstatus.CharacterStatusService
	// Dictionary service
	dt *dictionary.DictionaryService
}

// New creates a new Characters service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(
	st *storage.Storage,
	httpClient *http.Client,
	esiClient *goesi.APIClient,
	cs *characterstatus.CharacterStatusService,
	dt *dictionary.DictionaryService,
	eu *eveuniverse.EveUniverseService,
) *CharacterService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if esiClient == nil {
		esiClient = goesi.NewAPIClient(httpClient, "")
	}
	if cs == nil {
		cache := cache.New()
		cs = characterstatus.New(cache)
	}
	if dt == nil {
		dt = dictionary.New(st)
	}
	if eu == nil {
		eu = eveuniverse.New(st, esiClient)
	}
	ct := &CharacterService{
		st:         st,
		esiClient:  esiClient,
		httpClient: httpClient,
		sfg:        new(singleflight.Group),
		eu:         eu,
		cs:         cs,
		dt:         dt,
	}
	return ct
}
