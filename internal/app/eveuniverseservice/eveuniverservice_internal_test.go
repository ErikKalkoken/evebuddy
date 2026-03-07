package eveuniverseservice

import (
	"net/http"

	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

func NewTestService(st *storage.Storage) *EVEUniverseService {
	client := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
		UserAgent: "EveBuddy/1.0 (test@kalkoken.net)",
	})

	s := New(Params{
		ESIClient:          client,
		Signals:            app.NewSignals(),
		StatusCacheService: statuscacheservice.New(st),
		Storage:            st,
	})
	return s
}
