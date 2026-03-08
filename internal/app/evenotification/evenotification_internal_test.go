package evenotification

import (
	"net/http"
	"testing"
	"time"

	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestLDAPTimeConversion(t *testing.T) {
	t.Run("should convert LDAP time", func(t *testing.T) {
		x := fromLDAPTime(131924601300000000)
		xassert.Equal(t, time.Date(2019, 1, 20, 12, 15, 30, 0, time.UTC), x)
	})
	t.Run("should convert LDAP duration", func(t *testing.T) {
		x := fromLDAPDuration(9000000000)
		xassert.Equal(t, time.Duration(15*time.Minute), x)
	})
}

func NewEveUniverseService(st *storage.Storage) *eveuniverseservice.EVEUniverseService {
	client := goesi.NewESIClientWithOptions(http.DefaultClient, goesi.ClientOptions{
		UserAgent: "EveBuddy/1.0 (test@kalkoken.net)",
	})
	s := eveuniverseservice.New(eveuniverseservice.Params{
		ESIClient:          client,
		Signals:            app.NewSignals(),
		StatusCacheService: statuscacheservice.New(st),
		Storage:            st,
	})
	return s
}
