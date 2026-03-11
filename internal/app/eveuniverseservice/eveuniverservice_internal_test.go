package eveuniverseservice

import (
	"net/http"

	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

func NewFake(st *storage.Storage) *EVEUniverseService {
	client := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
		UserAgent: "EveBuddy/1.0 (test@kalkoken.net)",
	})

	s := New(Params{
		ESIClient:          client,
		Signals:            app.NewSignals(),
		StatusCacheService: new(statuscache.StatusCache),
		Storage:            st,
	})
	return s
}
