package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/kx/set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCharacterWalletJournalEntry(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		date := time.Now()
		arg := storage.CreateCharacterWalletJournalEntryParams{
			Amount:        123.45,
			Balance:       234.56,
			ContextID:     42,
			ContextIDType: "character",
			Date:          date,
			Description:   "bla bla",
			RefID:         4,
			CharacterID:   c.ID,
			Reason:        "my reason",
			RefType:       "player_donation",
			Tax:           0.12,
		}
		// when
		err := st.CreateCharacterWalletJournalEntry(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := st.GetCharacterWalletJournalEntry(ctx, storage.GetCharacterWalletJournalEntryParams{
				CharacterID: c.ID,
				RefID:       4,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, 123.45, i.Amount)
				assert.Equal(t, 234.56, i.Balance)
				assert.Equal(t, int64(42), i.ContextID)
				assert.Equal(t, "character", i.ContextIDType)
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
		c := factory.CreateCharacterFull()
		e1 := factory.CreateEveEntity()
		e2 := factory.CreateEveEntity()
		e3 := factory.CreateEveEntity()
		date := time.Now()
		arg := storage.CreateCharacterWalletJournalEntryParams{
			Amount:        123.45,
			Balance:       234.56,
			ContextID:     42,
			ContextIDType: "character",
			FirstPartyID:  e1.ID,
			Date:          date,
			Description:   "bla bla",
			RefID:         4,
			CharacterID:   c.ID,
			Reason:        "my reason",
			RefType:       "player_donation",
			SecondPartyID: e2.ID,
			Tax:           0.12,
			TaxReceiverID: e3.ID,
		}
		// when
		err := st.CreateCharacterWalletJournalEntry(ctx, arg)
		// then
		if assert.NoError(t, err) {
			i, err := st.GetCharacterWalletJournalEntry(ctx, storage.GetCharacterWalletJournalEntryParams{
				CharacterID: c.ID,
				RefID:       4,
			})
			if assert.NoError(t, err) {
				assert.Equal(t, 123.45, i.Amount)
				assert.Equal(t, 234.56, i.Balance)
				assert.Equal(t, int64(42), i.ContextID)
				assert.Equal(t, "character", i.ContextIDType)
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
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		e1 := factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		e2 := factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		e3 := factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		// when
		got, err := st.ListCharacterWalletJournalEntryIDs(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			want := set.Of(e1.RefID, e2.RefID, e3.RefID)
			xassert.EqualSet(t, want, got)
		}
	})
	t.Run("can list existing entries", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		factory.CreateCharacterWalletJournalEntry(storage.CreateCharacterWalletJournalEntryParams{CharacterID: c.ID})
		// when
		ee, err := st.ListCharacterWalletJournalEntries(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, ee, 3)
		}
	})
}
