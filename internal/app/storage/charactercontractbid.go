package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type CreateCharacterContractBidParams struct {
	ContractID int64
	Amount     float32
	BidID      int32
	BidderID   int32
	DateBid    time.Time
}

func (st *Storage) CreateCharacterContractBid(ctx context.Context, arg CreateCharacterContractBidParams) error {
	if arg.ContractID == 0 || arg.BidID == 0 {
		return fmt.Errorf("create contract bid: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateCharacterContractBidParams{
		ContractID: arg.ContractID,
		Amount:     float64(arg.Amount),
		BidID:      int64(arg.BidID),
		BidderID:   int64(arg.BidderID),
		DateBid:    arg.DateBid,
	}
	if err := st.qRW.CreateCharacterContractBid(ctx, arg2); err != nil {
		return fmt.Errorf("create contract bid: %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetCharacterContractBid(ctx context.Context, contractID int64, bidID int32) (*app.CharacterContractBid, error) {
	arg := queries.GetCharacterContractBidParams{
		ContractID: contractID,
		BidID:      int64(bidID),
	}
	r, err := st.qRO.GetCharacterContractBid(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get contract bid %+v: %w", arg, err)
	}
	return characterContractBidFromDBModel(r.CharacterContractBid, r.EveEntity), err
}

func (st *Storage) ListCharacterContractBids(ctx context.Context, contractID int64) ([]*app.CharacterContractBid, error) {
	rows, err := st.qRO.ListCharacterContractBids(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("list bids for contract %d: %w", contractID, err)
	}
	oo := make([]*app.CharacterContractBid, len(rows))
	for i, r := range rows {
		oo[i] = characterContractBidFromDBModel(r.CharacterContractBid, r.EveEntity)
	}
	return oo, nil
}

func (st *Storage) ListCharacterContractBidIDs(ctx context.Context, contractID int64) (set.Set[int32], error) {
	ids, err := st.qRO.ListCharacterContractBidIDs(ctx, contractID)
	if err != nil {
		return set.Set[int32]{}, fmt.Errorf("list bid IDs for contract %d: %w", contractID, err)
	}
	return set.Of(convertNumericSlice[int32](ids)...), err
}

func characterContractBidFromDBModel(o queries.CharacterContractBid, e queries.EveEntity) *app.CharacterContractBid {
	o2 := &app.CharacterContractBid{
		ContractID: o.ContractID,
		Amount:     float32(o.Amount),
		BidID:      int32(o.BidID),
		Bidder:     eveEntityFromDBModel(e),
		DateBid:    o.DateBid,
	}
	return o2
}
