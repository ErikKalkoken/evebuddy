package evenotification_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverse"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestBilling(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eu := eveuniverse.New(st, nil)
	en := evenotification.New()
	en.EveUniverseService = eu
	ctx := context.Background()
	t.Run("CorpAllBillMsg full data", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		creditor := factory.CreateEveEntityCorporation(app.EveEntity{ID: 1000023})
		debtor := factory.CreateEveEntityCorporation(app.EveEntity{ID: 98267621})
		office := factory.CreateEveEntityInventoryType(app.EveEntity{ID: 27})
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
		title, body, err := en.RenderESI(ctx, "CorpAllBillMsg", text, time.Now())
		if assert.NoError(t, err) {
			assert.Equal(t, "Bill issued for lease", title.ValueOrZero())
			assert.Contains(t, body.ValueOrZero(), creditor.Name)
			assert.Contains(t, body.ValueOrZero(), debtor.Name)
			assert.Contains(t, body.ValueOrZero(), office.Name)
			assert.Contains(t, body.ValueOrZero(), station.Name)
		}
	})
	t.Run("CorpAllBillMsg partial data", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
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
		title, body, err := en.RenderESI(ctx, "CorpAllBillMsg", text, time.Now())
		if assert.NoError(t, err) {
			assert.Equal(t, "Bill issued for lease", title.ValueOrZero())
			assert.Contains(t, body.ValueOrZero(), creditor.Name)
			assert.Contains(t, body.ValueOrZero(), debtor.Name)
			assert.Contains(t, body.ValueOrZero(), "?")
		}
	})
}