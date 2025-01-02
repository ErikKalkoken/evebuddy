package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCharacterContractBid(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterContract()
		bidder := factory.CreateEveEntityCharacter()
		const (
			amount = 123.45
			bidID  = 12345
		)
		dateBid := time.Now().UTC()
		arg := storage.CreateCharacterContractBidParams{
			ContractID: c.ID,
			Amount:     amount,
			BidID:      bidID,
			BidderID:   bidder.ID,
			DateBid:    dateBid,
		}
		// when
		err := r.CreateCharacterContractBid(ctx, arg)
		// then
		if assert.NoError(t, err) {
			o, err := r.GetCharacterContractBid(ctx, c.ID, bidID)
			if assert.NoError(t, err) {
				assert.InDelta(t, amount, o.Amount, 0.1)
				assert.Equal(t, bidder, o.Bidder)
				assert.Equal(t, dateBid, o.DateBid)
			}
		}
	})
	t.Run("can list existing bids", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterContract()
		factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{ContractID: c.ID})
		factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{ContractID: c.ID})
		factory.CreateCharacterContractBid(storage.CreateCharacterContractBidParams{ContractID: c.ID})
		// when
		oo, err := r.ListCharacterContractBids(ctx, c.ID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, oo, 3)
		}
	})
}
