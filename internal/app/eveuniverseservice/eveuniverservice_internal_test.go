package eveuniverseservice

import (
	"context"
	"net/http"

	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
)

type StatusCacheStub struct{}

func (c *StatusCacheStub) SetEveUniverseSection(o *app.EveUniverseSectionStatus) {}

func (c *StatusCacheStub) UpdateCharacters(ctx context.Context, st statuscache.Storage) error {
	return nil
}

func (c *StatusCacheStub) UpdateCorporations(ctx context.Context, st statuscache.Storage) error {
	return nil
}
func NewFake(st *storage.Storage) *EVEUniverseService {
	client := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
		UserAgent: "EveBuddy/1.0 (test@kalkoken.net)",
	})

	s := New(Params{
		ESIClient:          client,
		Signals:            app.NewSignals(),
		StatusCacheService: new(StatusCacheStub),
		Storage:            st,
	})
	return s
}
