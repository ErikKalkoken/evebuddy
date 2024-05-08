package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestWalletJournalEntry(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateMyCharacter()
		date := time.Now()
		arg := storage.CreateWalletJournalEntryParams{
			Amount:        123.45,
			Balance:       234.56,
			ContextID:     42,
			ContextIDType: "character",
			Date:          date,
			Description:   "bla bla",
			ID:            4,
			MyCharacterID: c.ID,
			Reason:        "my reason",
			RefType:       "player_donation",
			Tax:           0.12,
		}
		// when
		err := r.CreateWalletJournalEntry(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := r.GetWalletJournalEntry(ctx, c.ID, 4)
			if assert.NoError(t, err) {
				assert.Equal(t, 123.45, i.Amount)
				assert.Equal(t, 234.56, i.Balance)
				assert.Equal(t, int64(42), i.ContextID)
				assert.Equal(t, "character", i.ContextIDType)
				assert.Equal(t, date.Unix(), i.Date.Unix())
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
		c := factory.CreateMyCharacter()
		e1 := factory.CreateEveEntity()
		e2 := factory.CreateEveEntity()
		e3 := factory.CreateEveEntity()
		date := time.Now()
		arg := storage.CreateWalletJournalEntryParams{
			Amount:        123.45,
			Balance:       234.56,
			ContextID:     42,
			ContextIDType: "character",
			FirstPartyID:  e1.ID,
			Date:          date,
			Description:   "bla bla",
			ID:            4,
			MyCharacterID: c.ID,
			Reason:        "my reason",
			RefType:       "player_donation",
			SecondPartyID: e2.ID,
			Tax:           0.12,
			TaxReceiverID: e3.ID,
		}
		// when
		err := r.CreateWalletJournalEntry(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := r.GetWalletJournalEntry(ctx, c.ID, 4)
			if assert.NoError(t, err) {
				assert.Equal(t, 123.45, i.Amount)
				assert.Equal(t, 234.56, i.Balance)
				assert.Equal(t, int64(42), i.ContextID)
				assert.Equal(t, "character", i.ContextIDType)
				assert.Equal(t, e1, i.FirstParty)
				assert.Equal(t, date.Unix(), i.Date.Unix())
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
		c := factory.CreateMyCharacter()
		e1 := factory.CreateWalletJournalEntry(storage.CreateWalletJournalEntryParams{MyCharacterID: c.ID})
		e2 := factory.CreateWalletJournalEntry(storage.CreateWalletJournalEntryParams{MyCharacterID: c.ID})
		e3 := factory.CreateWalletJournalEntry(storage.CreateWalletJournalEntryParams{MyCharacterID: c.ID})
		// when
		ids, err := r.ListWalletJournalEntryIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			got := set.NewFromSlice(ids)
			want := set.NewFromSlice([]int64{e1.ID, e2.ID, e3.ID})
			assert.Equal(t, want, got)
		}
	})
}
