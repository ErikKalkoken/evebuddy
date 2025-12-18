package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestCorporationWalletJournalEntry(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
		err := st.CreateCorporationWalletJournalEntry(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := st.GetCorporationWalletJournalEntry(ctx, storage.GetCorporationWalletJournalEntryParams{
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
		testutil.MustTruncateTables(db)
		c := factory.CreateCorporation()
		firstParty := factory.CreateEveEntity()
		secondParty := factory.CreateEveEntity()
		taxReceiver := factory.CreateEveEntity()
		date := time.Now()
		arg := storage.CreateCorporationWalletJournalEntryParams{
			Amount:        123.45,
			Balance:       234.56,
			ContextID:     42,
			ContextIDType: "corporation",
			FirstPartyID:  firstParty.ID,
			Date:          date,
			Description:   "bla bla",
			DivisionID:    1,
			RefID:         4,
			CorporationID: c.ID,
			Reason:        "my reason",
			RefType:       "player_donation",
			SecondPartyID: secondParty.ID,
			Tax:           0.12,
			TaxReceiverID: taxReceiver.ID,
		}
		// when
		err := st.CreateCorporationWalletJournalEntry(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := st.GetCorporationWalletJournalEntry(ctx, storage.GetCorporationWalletJournalEntryParams{
				CorporationID: c.ID,
				DivisionID:    1,
				RefID:         4,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, 123.45, i.Amount)
				assert.Equal(t, 234.56, i.Balance)
				assert.Equal(t, int64(42), i.ContextID)
				assert.Equal(t, "corporation", i.ContextIDType)
				assert.Equal(t, firstParty, i.FirstParty)
				assert.Equal(t, date.UTC(), i.Date.UTC())
				assert.Equal(t, "bla bla", i.Description)
				assert.Equal(t, "player_donation", i.RefType)
				assert.Equal(t, "my reason", i.Reason)
				assert.Equal(t, secondParty, i.SecondParty)
				assert.Equal(t, taxReceiver, i.TaxReceiver)
				assert.Equal(t, 0.12, i.Tax)
			}
		}
	})
	t.Run("can list IDs of existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
		got, err := st.ListCorporationWalletJournalEntryIDs(ctx, storage.CorporationDivision{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		// then
		if assert.NoError(t, err) {
			want := set.Of(e1.RefID, e2.RefID)
			xassert.EqualSet(t, want, got)
		}
	})
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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
		oo, err := st.ListCorporationWalletJournalEntries(ctx, storage.CorporationDivision{
			CorporationID: c.ID,
			DivisionID:    1,
		})
		// then
		if assert.NoError(t, err) {
			got := set.Of(xslices.Map(oo, func(x *app.CorporationWalletJournalEntry) int64 {
				return x.RefID
			})...)
			want := set.Of(e1.RefID, e2.RefID)
			xassert.EqualSet(t, want, got)
		}
	})
	t.Run("can store multiple", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
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

		err := st.CreateCorporationWalletJournalEntry(ctx, arg)
		if assert.NoError(t, err) {
			arg.RefID = 5
			err := st.CreateCorporationWalletJournalEntry(ctx, arg)
			if assert.NoError(t, err) {
				got, err := st.ListCorporationWalletJournalEntryIDs(ctx, storage.CorporationDivision{
					CorporationID: c.ID,
					DivisionID:    1,
				})
				if assert.NoError(t, err) {
					want := set.Of[int64](4, 5)
					xassert.EqualSet(t, want, got)
				}
			}
		}
	})
	t.Run("can delete journal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		e1 := factory.CreateCorporationWalletJournalEntry()
		e2 := factory.CreateCorporationWalletJournalEntry()
		// when
		err := st.DeleteCorporationWalletJournal(ctx, e1.CorporationID, app.Division(e1.DivisionID))
		// then
		if assert.NoError(t, err) {
			x1, err := st.ListCorporationWalletJournalEntryIDs(ctx, storage.CorporationDivision{
				CorporationID: e1.CorporationID,
				DivisionID:    e1.DivisionID,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, 0, x1.Size())
			}
			x2, err := st.ListCorporationWalletJournalEntryIDs(ctx, storage.CorporationDivision{
				CorporationID: e2.CorporationID,
				DivisionID:    e2.DivisionID,
			})
			if assert.NoError(t, err) {
				assert.Greater(t, x2.Size(), 0)
			}
		}
	})
}
