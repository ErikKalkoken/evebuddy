// Package eveuniverse contains the Eve universe service.
package eveuniverse

import (
	"errors"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"
)

var ErrNotFound = errors.New("object not found")

// EveUniverseService provides access to Eve Online models with on-demand loading from ESI and local caching.
type EveUniverseService struct {
	StatusCacheService app.StatusCacheService
	// Now returns the current time in UTC. Can be overwritten for tests.
	Now func() time.Time

	esiClient *goesi.APIClient
	sfg       *singleflight.Group
	st        *storage.Storage
}

// New returns a new instance of an Eve universe service.
func New(st *storage.Storage, esiClient *goesi.APIClient) *EveUniverseService {
	eu := &EveUniverseService{
		esiClient: esiClient,
		st:        st,
		sfg:       new(singleflight.Group),
		Now: func() time.Time {
			return time.Now().UTC()
		},
	}
	return eu
}
