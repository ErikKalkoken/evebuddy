package evenotification

import (
	"context"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
)

func TestMakeStructureBaseText(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	xesi.ActivateRateLimiterMock()
	defer xesi.DeactivateRateLimiterMock()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		Storage: st,
	})
	ctx := context.Background()
	t.Run("can create base text from complete input data", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		o := factory.CreateEveLocationStructure()
		// when
		x, err := makeStructureBaseText(ctx, o.Type.ID, o.SolarSystem.ID, o.ID, o.Name, eus)
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
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		es := factory.CreateEveSolarSystem()
		// when
		x, err := makeStructureBaseText(ctx, 0, es.ID, 1_000_000_000_000, "", eus)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, "???", x.name)
			assert.Equal(t, es.Name, x.solarSystem.Name)
			assert.Empty(t, x.eveType)
			assert.Empty(t, x.owner)
			assert.NotEmpty(t, x.intro)
		}
	})
}

func TestEveEntityFromHTMLLink(t *testing.T) {
	cases := []struct {
		url      string
		category app.EveEntityCategory
		id       int32
		name     string
		isValid  bool
	}{
		{`<a href="showinfo:2//2011">Bad Corp</a>`, app.EveEntityCorporation, 2011, "Bad Corp", true},
		{`<a href="showinfo:16159//3011">Bad Alliance</a>`, app.EveEntityAlliance, 3011, "Bad Alliance", true},
		{`<a href="showinfo:1376//1001">Charlie</a>`, app.EveEntityCharacter, 1001, "Charlie", true},
		{`<a href="showinfo:5//42">Alpha</a>`, app.EveEntitySolarSystem, 42, "Alpha", true},
		{`<a href="http://www.google.com">Alpha</a>`, app.EveEntityUndefined, 0, "", false},
		{``, app.EveEntityUndefined, 0, "", false},
		{`<br>`, app.EveEntityUndefined, 0, "", false},
		{`<a href="showinfo:666//42">Invalid</a>`, app.EveEntityUndefined, 0, "", false},
	}
	for _, tc := range cases {
		o, err := eveEntityFromHTMLLink(tc.url)
		if tc.isValid {
			if !assert.NoError(t, err) {
				t.Fatal()
			}
			assert.Equal(t, tc.category, o.Category)
			assert.EqualValues(t, tc.id, o.ID)
			assert.EqualValues(t, tc.name, o.Name)
		} else {
			assert.Error(t, err)
		}
	}
}
