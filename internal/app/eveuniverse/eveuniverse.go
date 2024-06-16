// Package eveuniverse contains the Eve universe service.
package eveuniverse

import (
	"errors"

	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"
)

var ErrNotFound = errors.New("object not found")

// EveUniverseService provides access to Eve Online models with on-demand loading from ESI and local caching.
type EveUniverseService struct {
	StatusCacheService *statuscache.StatusCacheService

	esiClient *goesi.APIClient
	sfg       *singleflight.Group
	st        *sqlite.Storage
}

// New returns a new instance of an Eve universe service.
func New(st *sqlite.Storage, esiClient *goesi.APIClient) *EveUniverseService {
	eu := &EveUniverseService{
		esiClient: esiClient,
		st:        st,
		sfg:       new(singleflight.Group),
	}
	return eu
}
