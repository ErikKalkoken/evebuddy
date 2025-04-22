package evenotification

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestMakeStructureBaseText(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		Storage: st,
	})
	s := New(eus)
	ctx := context.Background()
	t.Run("can create base text from complete input data", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		o := factory.CreateEveLocationStructure()
		// when
		x, err := s.makeStructureBaseText(ctx, o.Type.ID, o.SolarSystem.ID, o.ID, o.Name)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, o.Name, x.name)
			assert.Equal(t, o.SolarSystem.Name, x.solarSystem.Name)
			assert.Equal(t, o.Type.Name, x.eveType.Name)
			assert.Equal(t, o.Owner.Name, x.owner.Name)
			assert.NotEmpty(t, x.intro)
		}
	})
	t.Run("can create base text from minimal input data", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		es := factory.CreateEveSolarSystem()
		// when
		x, err := s.makeStructureBaseText(ctx, 0, es.ID, 1_000_000_000_000, "")
		// then
		if assert.NoError(t, err) {
			assert.Empty(t, x.name)
			assert.Equal(t, es.Name, x.solarSystem.Name)
			assert.Empty(t, x.eveType)
			assert.Empty(t, x.owner)
			assert.NotEmpty(t, x.intro)
		}
	})
}

func TestLDAPTimeConversion(t *testing.T) {
	t.Run("should convert LDAP time", func(t *testing.T) {
		x := fromLDAPTime(131924601300000000)
		assert.Equal(t, time.Date(2019, 1, 20, 12, 15, 30, 0, time.UTC), x)
	})
	t.Run("should convert LDAP duration", func(t *testing.T) {
		x := fromLDAPDuration(9000000000)
		assert.Equal(t, time.Duration(15*time.Minute), x)
	})
}
