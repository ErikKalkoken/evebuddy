package evenotification_test

import (
	"testing"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestBilling_RenderESI(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	eus := evenotification.NewEveUniverseService(st)
	en := evenotification.New(eus)

	t.Run("CorpAllBillMsg full data", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		creditor := f.CreateEveEntityCorporation()
		debtor := f.CreateEveEntityCorporation()
		office := f.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 27})
		station := f.CreateEveEntity(app.EveEntity{Name: "Alpha", Category: app.EveEntityStation})
		text, err := yaml.Marshal(map[string]any{
			"amount":      10000,
			"billTypeID":  2,
			"creditorID":  creditor.ID,
			"currentDate": 133678830021821155,
			"debtorID":    debtor.ID,
			"dueDate":     133704743590000000,
			"externalID":  office.ID,
			"externalID2": station.ID,
		})
		require.NoError(t, err)

		// when
		title, body, err := en.RenderESI(t.Context(), app.CorpAllBillMsg, optional.New(string(text)), time.Now())

		// then
		require.NoError(t, err)
		xassert.Equal(t, "Bill issued for extending lease at Alpha", title)
		assert.Contains(t, body, creditor.Name)
		assert.Contains(t, body, debtor.Name)
		assert.Contains(t, body, office.Name)
		assert.Contains(t, body, station.Name)
	})

	t.Run("CorpAllBillMsg partial data", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		creditor := f.CreateEveEntityCorporation(app.EveEntity{ID: 1000023})
		debtor := f.CreateEveEntityCorporation(app.EveEntity{ID: 98267621})
		text := `
amount: 10000
billTypeID: 2
creditorID: 1000023
currentDate: 133678830021821155
debtorID: 98267621
dueDate: 133704743590000000
externalID: 0
externalID2: 0`

		// when
		title, body, err := en.RenderESI(t.Context(), app.CorpAllBillMsg, optional.New(text), time.Now())

		// then
		require.NoError(t, err)
		xassert.Equal(t, "Bill issued for extending lease at ?", title)
		assert.Contains(t, body, creditor.Name)
		assert.Contains(t, body, debtor.Name)
		assert.Contains(t, body, "?")
	})

	t.Run("CorpAllBillMsg full data with structure ID", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		creditor := f.CreateEveEntityCorporation()
		debtor := f.CreateEveEntityCorporation()
		office := f.CreateEveEntityWithCategory(app.EveEntityInventoryType, app.EveEntity{ID: 27})
		structure := f.CreateEveLocationStructure(storage.UpdateOrCreateLocationParams{Name: "Bravo"})
		text, err := yaml.Marshal(map[string]any{
			"amount":      10000,
			"billTypeID":  2,
			"creditorID":  creditor.ID,
			"currentDate": 133678830021821155,
			"debtorID":    debtor.ID,
			"dueDate":     133704743590000000,
			"externalID":  office.ID,
			"externalID2": structure.ID,
		})
		require.NoError(t, err)

		// when
		title, body, err := en.RenderESI(t.Context(), app.CorpAllBillMsg, optional.New(string(text)), time.Now())

		// then
		require.NoError(t, err)
		xassert.Equal(t, "Bill issued for extending lease at Bravo", title)
		assert.Contains(t, body, creditor.Name)
		assert.Contains(t, body, debtor.Name)
		assert.Contains(t, body, office.Name)
		assert.Contains(t, body, structure.Name)
	})
}

func TestBilling_EntityIDs(t *testing.T) {
	db, st, _ := testutil.NewDBInMemory()
	defer db.Close()
	eus := evenotification.NewEveUniverseService(st)
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

		// when
		ids, err := en.EntityIDs(app.CorpAllBillMsg, optional.New(text))

		// then
		require.NoError(t, err)
		want := set.Of[int64](27, 1000023, 60003760, 98267621)
		xassert.Equal(t, want, ids)
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

		// when
		ids, err := en.EntityIDs(app.CorpAllBillMsg, optional.New(text))
		require.NoError(t, err)

		// then
		want := set.Of[int64](27, 1000023, 98267621)
		xassert.Equal(t, want, ids)
	})
}
