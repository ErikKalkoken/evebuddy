package characterservice_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestGetWalletJournalEntry(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("can return a wallet journal entry", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o := factory.CreateCharacterWalletJournalEntry()
		// when
		got, err := s.GetWalletJournalEntry(ctx, o.CharacterID, o.RefID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, o.RefID, got.RefID)
		}
	})
	t.Run("should return error when not found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := s.GetWalletJournalEntry(ctx, 1, 1)
		// then
		assert.Error(t, err)
	})
}

func TestGetWalletTransactions(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("can return a wallet transaction", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		o := factory.CreateCharacterWalletTransaction()
		// when
		got, err := s.GetWalletTransactions(ctx, o.CharacterID, o.TransactionID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, o.TransactionID, got.TransactionID)
		}
	})
	t.Run("should return error when not found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := s.GetWalletTransactions(ctx, 1, 1)
		// then
		assert.Error(t, err)
	})
}
