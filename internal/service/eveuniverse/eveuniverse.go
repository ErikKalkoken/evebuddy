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
	esiClient   *goesi.APIClient
	singleGroup *singleflight.Group
	s           *storage.Storage
}

func New(s *storage.Storage, esiClient *goesi.APIClient) *EveUniverse {
	eu := &EveUniverse{
		esiClient:   esiClient,
		s:           s,
		singleGroup: new(singleflight.Group),
	}
	return eu
}
