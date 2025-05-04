package corporationservice

import (
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
)

func NewFake(st *storage.Storage) *CorporationService {
	scs := statuscacheservice.New(memcache.New(), st)
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		StatusCacheService: scs,
		Storage:            st,
	})
	arg := Params{
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
	}
	s := New(arg)
	return s
}
