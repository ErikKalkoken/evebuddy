package evenotification_test

import (
	"context"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestBilling_RenderESI(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		Storage: st,
	})
	en := evenotification.New(eus)
	ctx := context.Background()
	t.Run("CorpAllBillMsg full data", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		creditor := factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000023})
		debtor := factory.CreateEveEntityCorporation(app.EveEntity{ID: 98267621})
		office := factory.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 27})
		station := factory.CreateEveEntity(app.EveEntity{ID: 60003760, Category: app.EveEntityStation})
		text := `
amount: 10000
billTypeID: 2
creditorID: 1000023
currentDate: 133678830021821155
debtorID: 98267621
dueDate: 133704743590000000
externalID: 27
externalID2: 60003760`
		title, body, err := en.RenderESI(ctx, app.CorpAllBillMsg, optional.New(text), time.Now())
		require.NoError(t, err)
		xassert.Equal(t, "Bill issued for lease", title)
		assert.Contains(t, body, creditor.Name)
		assert.Contains(t, body, debtor.Name)
		assert.Contains(t, body, office.Name)
		assert.Contains(t, body, station.Name)
	})
	t.Run("CorpAllBillMsg partial data", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		creditor := factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000023})
		debtor := factory.CreateEveEntityCorporation(app.EveEntity{ID: 98267621})
		text := `
amount: 10000
billTypeID: 2
creditorID: 1000023
currentDate: 133678830021821155
debtorID: 98267621
dueDate: 133704743590000000
externalID: 0
externalID2: 0`
		title, body, err := en.RenderESI(ctx, app.CorpAllBillMsg, optional.New(text), time.Now())
		require.NoError(t, err)
		xassert.Equal(t, "Bill issued for lease", title)
		assert.Contains(t, body, creditor.Name)
		assert.Contains(t, body, debtor.Name)
		assert.Contains(t, body, "?")
	})
}

func TestBilling_EntityIDs(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		Storage: st,
	})
	en := evenotification.New(eus)
	t.Run("Can retrieve all entity IDs", func(t *testing.T) {
		// given
		text := `
amount: 10000
billTypeID: 2
creditorID: 1000023
currentDate: 133678830021821155
debtorID: 98267621
dueDate: 133704743590000000
externalID: 27
externalID2: 60003760`
		ids, err := en.EntityIDs(app.CorpAllBillMsg, optional.New(text))
		require.NoError(t, err)
		want := set.Of[int64](27, 1000023, 60003760, 98267621)
		xassert.Equal2(t, want, ids)
	})
	t.Run("should not return invalid entity IDs", func(t *testing.T) {
		// given
		text := `
amount: 10000
billTypeID: 2
creditorID: 1000023
currentDate: 133678830021821155
debtorID: 98267621
dueDate: 133704743590000000
externalID: 27
externalID2: 1047607396377`
		ids, err := en.EntityIDs(app.CorpAllBillMsg, optional.New(text))
		require.NoError(t, err)
		want := set.Of[int64](27, 1000023, 98267621)
		xassert.Equal2(t, want, ids)
	})
}
