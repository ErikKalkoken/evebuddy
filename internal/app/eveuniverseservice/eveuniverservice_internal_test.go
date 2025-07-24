package eveuniverseservice

import (
	"github.com/antihax/goesi"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

func NewTestService(st *storage.Storage) *EveUniverseService {
	s := New(Params{
		Storage:   st,
		ESIClient: goesi.NewAPIClient(nil, "test@kalkoken.net"),
	})
	return s
}
