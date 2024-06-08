// Package eveuniverse contains the Eve universe service.
package eveuniverse

import (
	"errors"

	"github.com/ErikKalkoken/evebuddy/internal/service/dictionary"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"
)

var ErrNotFound = errors.New("object not found")

// EveUniverseService provides access to Eve Online models with on-demand loading from ESI and local caching.
type EveUniverseService struct {
	esiClient *goesi.APIClient
	sfg       *singleflight.Group
	st        *storage.Storage

	// Dictionary service
	dt *dictionary.DictionaryService
}

// New returns a new instance of an Eve universe service.
func New(st *storage.Storage, esiClient *goesi.APIClient, dt *dictionary.DictionaryService) *EveUniverseService {
	if dt == nil {
		dt = dictionary.New(st)
	}
	eu := &EveUniverseService{
		esiClient: esiClient,
		dt:        dt,
		st:        st,
		sfg:       new(singleflight.Group),
	}
	return eu
}
