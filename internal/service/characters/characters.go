// Package characters contains the characters service.
package characters

import (
	"net/http"

	"github.com/ErikKalkoken/evebuddy/internal/helper/cache"
	"github.com/ErikKalkoken/evebuddy/internal/service/characterstatus"
	"github.com/ErikKalkoken/evebuddy/internal/service/dictionary"
	"github.com/ErikKalkoken/evebuddy/internal/service/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"
)

// The Characters service implements the characters related business logic.
type Characters struct {
	esiClient  *goesi.APIClient
	httpClient *http.Client
	sfg        *singleflight.Group
	// Storage service
	st *storage.Storage
	// EveUniverse service
	eu *eveuniverse.EveUniverse
	// CharacterStatus service
	cs *characterstatus.CharacterStatusCache
	// Dictionary service
	dt *dictionary.Dictionary
}

// New creates a new Characters service and returns it.
// When nil is passed for any parameter a new default instance will be created for it (except for storage).
func New(
	st *storage.Storage,
	httpClient *http.Client,
	esiClient *goesi.APIClient,
	cs *characterstatus.CharacterStatusCache,
	dt *dictionary.Dictionary,
	eu *eveuniverse.EveUniverse,
) *Characters {
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
	ct := &Characters{
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
