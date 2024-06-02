// Package eveuniverse provides access to Eve Online models with on-demand loading from ESI
// and storage of fetches data to local storage for caching.
package eveuniverse

import (
	"errors"

	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"
)

var ErrNotFound = errors.New("object not found")

// EveUniverse services provides access to Eve Online models with on-demand loading from ESI.
type EveUniverseService struct {
	esiClient *goesi.APIClient
	sfg       *singleflight.Group
	st        *storage.Storage
}

// New returns a new instance of an EveUniverse service.
func New(st *storage.Storage, esiClient *goesi.APIClient) *EveUniverseService {
	eu := &EveUniverseService{
		esiClient: esiClient,
		st:        st,
		sfg:       new(singleflight.Group),
	}
	return eu
}
