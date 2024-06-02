// Package eveuniverse access to Eve Online models with on-demand loading from ESI
// and storage of fetches data to local storage for caching.
package eveuniverse

import (
	"errors"

	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/antihax/goesi"
	"golang.org/x/sync/singleflight"
)

var ErrNotFound = errors.New("object not found")

type EveUniverse struct {
	esiClient *goesi.APIClient
	sfg       *singleflight.Group
	st        *storage.Storage
}

func New(st *storage.Storage, esiClient *goesi.APIClient) *EveUniverse {
	eu := &EveUniverse{
		esiClient: esiClient,
		st:        st,
		sfg:       new(singleflight.Group),
	}
	return eu
}
