package characterservice

import (
	"context"
	"sync"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	"github.com/antihax/goesi"
)

type CacheFake struct {
	m sync.Map
}

func (c *CacheFake) Clear() {
	c.m.Clear()
}

func (c *CacheFake) Get(k string) ([]byte, bool) {
	x, ok := c.m.Load(k)
	if !ok {
		return nil, false
	}
	v := x.([]byte)
	return v, ok
}

func (c *CacheFake) Set(k string, v []byte, _ time.Duration) {
	c.m.Store(k, v)
}

func NewFake(st *storage.Storage, args ...Params) *CharacterService {
	scs := statuscacheservice.New(memcache.New(), st)
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		ESIClient:          goesi.NewAPIClient(nil, ""),
		StatusCacheService: scs,
		Storage:            st,
	})
	arg := Params{
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
		TickerSource:       &testutil.FakeTicker{},
		Cache:              new(CacheFake),
	}
	if len(args) > 0 {
		a := args[0]
		if a.SSOService != nil {
			arg.SSOService = a.SSOService
		}
	}
	s := New(arg)
	return s
}

type SSOFake struct {
	Token *app.Token
	Err   error
}

func (s SSOFake) Authenticate(ctx context.Context, scopes []string) (*app.Token, error) {
	return s.Token, s.Err
}

func (s SSOFake) RefreshToken(ctx context.Context, refreshToken string) (*app.Token, error) {
	return s.Token, s.Err
}
