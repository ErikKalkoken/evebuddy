package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
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
			Amount:        optional.New(123.45),
			Balance:       optional.New(234.56),
			ContextID:     optional.New[int64](42),
			ContextIDType: optional.New("character"),
			Date:          date,
			Description:   "bla bla",
			RefID:         4,
			CharacterID:   c.ID,
			Reason:        optional.New("my reason"),
			RefType:       "player_donation",
			Tax:           optional.New(0.12),
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
				xassert.Equal(t, 123.45, i.Amount.ValueOrZero())
				xassert.Equal(t, 234.56, i.Balance.ValueOrZero())
				xassert.Equal(t, int64(42), i.ContextID.ValueOrZero())
				xassert.Equal(t, "character", i.ContextIDType.ValueOrZero())
				xassert.Equal2(t, date, i.Date)
				xassert.Equal(t, "bla bla", i.Description)
				xassert.Equal(t, "player_donation", i.RefType)
				xassert.Equal(t, "my reason", i.Reason.ValueOrZero())
				xassert.Equal(t, 0.12, i.Tax.ValueOrZero())
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
			Amount:        optional.New(123.45),
			Balance:       optional.New(234.56),
			ContextID:     optional.New[int64](42),
			ContextIDType: optional.New("character"),
			FirstPartyID:  optional.New(e1.ID),
			Date:          date,
			Description:   "bla bla",
			RefID:         4,
			CharacterID:   c.ID,
			Reason:        optional.New("my reason"),
			RefType:       "player_donation",
			SecondPartyID: optional.New(e2.ID),
			Tax:           optional.New(0.12),
			TaxReceiverID: optional.New(e3.ID),
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
				xassert.Equal(t, 123.45, i.Amount.ValueOrZero())
				xassert.Equal(t, 234.56, i.Balance.ValueOrZero())
				xassert.Equal(t, int64(42), i.ContextID.ValueOrZero())
				xassert.Equal(t, "character", i.ContextIDType.ValueOrZero())
				xassert.Equal(t, e1, i.FirstParty)
				xassert.Equal2(t, date, i.Date)
				xassert.Equal(t, "bla bla", i.Description)
				xassert.Equal(t, "player_donation", i.RefType)
				xassert.Equal(t, "my reason", i.Reason.ValueOrZero())
				xassert.Equal(t, e2, i.SecondParty)
				xassert.Equal(t, e3, i.TaxReceiver)
				xassert.Equal(t, 0.12, i.Tax.ValueOrZero())
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
			xassert.Equal2(t, want, got)
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
