package eveuniverseservice

import (
	"net/http"

	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

func NewTestService(st *storage.Storage) *EveUniverseService {
	client := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
		UserAgent: "EveBuddy/1.0 (test@kalkoken.net)",
	})
	s := New(Params{
		Storage:   st,
		ESIClient: client,
	})
	return s
}
