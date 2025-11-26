package evenotification_test

import (
	"context"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xesi"
)

func TestTowerNotification(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	xesi.ActivateRateLimiterMock()
	defer xesi.DeactivateRateLimiterMock()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		Storage: st,
	})
	en := evenotification.New(eus)
	ctx := context.Background()
	t.Run("TowerAlertMsg full data", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		aggressorAlliance := factory.CreateEveEntityAlliance(app.EveEntity{ID: 3011})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 2011})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 1011})
		type_ := factory.CreateEveType(storage.CreateEveTypeParams{ID: 16213})
		moon := factory.CreateEveMoon(storage.CreateEveMoonParams{ID: 40161465})
		text := `
aggressorAllianceID: 3011
aggressorCorpID: 2011
aggressorID: 1011
armorValue: 0.6950949076033535
hullValue: 1.0
moonID: 40161465
shieldValue: 0.3950949076033535
solarSystemID: 30002537
typeID: 16213`
		title, body, err := en.RenderESI(ctx, app.TowerAlertMsg, text, time.Now())
		if assert.NoError(t, err) {

			assert.Contains(t, title, "is under attack")
			assert.Contains(t, body, aggressorAlliance.Name)
			assert.Contains(t, body, moon.Name)
			assert.Contains(t, body, type_.Name)
		}
	})
	t.Run("TowerAlertMsg partial data 1", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		aggressorAlliance := factory.CreateEveEntityAlliance(app.EveEntity{ID: 3011})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 2011})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 1011})
		type_ := factory.CreateEveType(storage.CreateEveTypeParams{ID: 16213})
		moon := factory.CreateEveMoon(storage.CreateEveMoonParams{ID: 40161465})
		text := `
aggressorAllianceID: 3011
aggressorCorpID: 2011
aggressorID: 0
armorValue: 0.6950949076033535
hullValue: 1.0
moonID: 40161465
shieldValue: 0.3950949076033535
solarSystemID: 30002537
typeID: 16213`
		title, body, err := en.RenderESI(ctx, app.TowerAlertMsg, text, time.Now())
		if assert.NoError(t, err) {

			assert.Contains(t, title, "is under attack")
			assert.Contains(t, body, aggressorAlliance.Name)
			assert.Contains(t, body, moon.Name)
			assert.Contains(t, body, type_.Name)
		}
	})
	t.Run("TowerAlertMsg partial data 1", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		aggressorAlliance := factory.CreateEveEntityAlliance(app.EveEntity{ID: 3011})
		factory.CreateEveEntityCorporation(app.EveEntity{ID: 2011})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: 1011})
		type_ := factory.CreateEveType(storage.CreateEveTypeParams{ID: 16213})
		moon := factory.CreateEveMoon(storage.CreateEveMoonParams{ID: 40161465})
		text := `
aggressorAllianceID: 3011
aggressorCorpID: 2011
aggressorID: 0
armorValue: 0.6950949076033535
hullValue: 1.0
moonID: 40161465
shieldValue: 0.3950949076033535
solarSystemID: 30002537
typeID: 16213`
		title, body, err := en.RenderESI(ctx, app.TowerAlertMsg, text, time.Now())
		if assert.NoError(t, err) {

			assert.Contains(t, title, "is under attack")
			assert.Contains(t, body, aggressorAlliance.Name)
			assert.Contains(t, body, moon.Name)
			assert.Contains(t, body, type_.Name)
		}
	})
}
