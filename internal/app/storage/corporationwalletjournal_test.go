package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/stretchr/testify/assert"
)

func TestCorporationWalletJournalEntry(t *testing.T) {
	db, r, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		date := time.Now()
		arg := storage.CreateCorporationWalletJournalEntryParams{
			Amount:        123.45,
			Balance:       234.56,
			ContextID:     42,
			ContextIDType: "corporation",
			Date:          date,
			Description:   "bla bla",
			DivisionID:    1,
			RefID:         4,
			CorporationID: c.ID,
			Reason:        "my reason",
			RefType:       "player_donation",
			Tax:           0.12,
		}
		// when
		err := r.CreateCorporationWalletJournalEntry(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := r.GetCorporationWalletJournalEntry(ctx, storage.GetCorporationWalletJournalEntryParams{
				CorporationID: c.ID,
				DivisionID:    1,
				RefID:         4,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, 123.45, i.Amount)
				assert.Equal(t, 234.56, i.Balance)
				assert.Equal(t, int64(42), i.ContextID)
				assert.Equal(t, "corporation", i.ContextIDType)
				assert.Equal(t, date.UTC(), i.Date.UTC())
				assert.Equal(t, "bla bla", i.Description)
				assert.Equal(t, "player_donation", i.RefType)
				assert.Equal(t, "my reason", i.Reason)
				assert.Equal(t, 0.12, i.Tax)
			}
		}
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		e1 := factory.CreateEveEntity()
		e2 := factory.CreateEveEntity()
		e3 := factory.CreateEveEntity()
		date := time.Now()
		arg := storage.CreateCorporationWalletJournalEntryParams{
			Amount:        123.45,
			Balance:       234.56,
			ContextID:     42,
			ContextIDType: "corporation",
			FirstPartyID:  e1.ID,
			Date:          date,
			Description:   "bla bla",
			DivisionID:    1,
			RefID:         4,
			CorporationID: c.ID,
			Reason:        "my reason",
			RefType:       "player_donation",
			SecondPartyID: e2.ID,
			Tax:           0.12,
			TaxReceiverID: e3.ID,
		}
		// when
		err := r.CreateCorporationWalletJournalEntry(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := r.GetCorporationWalletJournalEntry(ctx, storage.GetCorporationWalletJournalEntryParams{
				CorporationID: c.ID,
				DivisionID:    1,
				RefID:         4,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, 123.45, i.Amount)
				assert.Equal(t, 234.56, i.Balance)
				assert.Equal(t, int64(42), i.ContextID)
				assert.Equal(t, "corporation", i.ContextIDType)
				assert.Equal(t, e1, i.FirstParty)
				assert.Equal(t, date.UTC(), i.Date.UTC())
				assert.Equal(t, "bla bla", i.Description)
				assert.Equal(t, "player_donation", i.RefType)
				assert.Equal(t, "my reason", i.Reason)
				assert.Equal(t, e2, i.SecondParty)
				assert.Equal(t, e3, i.TaxReceiver)
				assert.Equal(t, 0.12, i.Tax)
			}
		}
	})
	t.Run("can list IDs of existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		e1 := factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		e2 := factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    2,
		})
		factory.CreateCorporationWalletJournalEntry()
		// when
		got, err := r.ListCorporationWalletJournalEntryIDs(ctx, storage.CorporationDivision{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		// then
		if assert.NoError(t, err) {
			want := set.Of(e1.RefID, e2.RefID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCorporation()
		e1 := factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		e2 := factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		factory.CreateCorporationWalletJournalEntry(storage.CreateCorporationWalletJournalEntryParams{
			CorporationID: c.ID,
			DivisionID:    2,
		})
		factory.CreateCorporationWalletJournalEntry()
		// when
		oo, err := r.ListCorporationWalletJournalEntries(ctx, storage.CorporationDivision{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		// then
		if assert.NoError(t, err) {
			got := set.Of(xslices.Map(oo, func(x *app.CorporationWalletJournalEntry) int64 {
				return x.RefID
			})...)
			want := set.Of(e1.RefID, e2.RefID)
			assert.True(t, got.Equal(want), "got %q, wanted %q", got, want)
		}
	})
}
