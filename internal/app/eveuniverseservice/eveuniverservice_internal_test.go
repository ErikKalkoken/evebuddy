package eveuniverseservice

import (
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/antihax/goesi"
)

func NewTestService(st *storage.Storage) *EveUniverseService {
	s := New(Params{
		Storage:   st,
		ESIClient: goesi.NewAPIClient(nil, "test@kalkoken.net"),
	})
	return s
}
