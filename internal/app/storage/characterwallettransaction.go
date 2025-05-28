package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type CreateCharacterWalletTransactionParams struct {
	ClientID      int32
	Date          time.Time
	EveTypeID     int32
	IsBuy         bool
	IsPersonal    bool
	JournalRefID  int64
	LocationID    int64
	CharacterID   int32
	Quantity      int32
	TransactionID int64
	UnitPrice     float64
}

func (st *Storage) CreateCharacterWalletTransaction(ctx context.Context, arg CreateCharacterWalletTransactionParams) error {
	if arg.CharacterID == 0 || arg.EveTypeID == 0 || arg.ClientID == 0 || arg.JournalRefID == 0 || arg.LocationID == 0 || arg.TransactionID == 0 {
		return fmt.Errorf("CreateCharacterWalletTransaction: %+v: %w", arg, app.ErrInvalid)
	}
	arg2 := queries.CreateCharacterWalletTransactionParams{
		ClientID:      int64(arg.ClientID),
		Date:          arg.Date,
		EveTypeID:     int64(arg.EveTypeID),
		IsBuy:         arg.IsBuy,
		IsPersonal:    arg.IsPersonal,
		JournalRefID:  arg.JournalRefID,
		LocationID:    arg.LocationID,
		CharacterID:   int64(arg.CharacterID),
		Quantity:      int64(arg.Quantity),
		TransactionID: arg.TransactionID,
		UnitPrice:     arg.UnitPrice,
	}

	if err := st.qRW.CreateCharacterWalletTransaction(ctx, arg2); err != nil {
		return fmt.Errorf("create wallet transaction: %+v: %w", arg2, err)
	}
	return nil
}

func (st *Storage) GetCharacterWalletTransaction(ctx context.Context, characterID int32, transactionID int64) (*app.CharacterWalletTransaction, error) {
	arg := queries.GetCharacterWalletTransactionParams{
		CharacterID:   int64(characterID),
		TransactionID: transactionID,
	}
	r, err := st.qRO.GetCharacterWalletTransaction(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get wallet transaction for character %d: %w", characterID, convertGetError(err))
	}
	o := characterWalletTransactionFromDBModel(
		r.CharacterWalletTransaction,
		r.EveEntity,
		r.EveType,
		r.EveGroup,
		r.EveCategory,
		r.LocationName,
		r.SystemSecurityStatus,
		r.RegionID,
		r.RegionName,
	)
	return o, err
}

func (st *Storage) ListCharacterWalletTransactionIDs(ctx context.Context, characterID int32) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterWalletTransactionIDs(ctx, int64(characterID))
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list wallet transaction ids for character %d: %w", characterID, err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListCharacterWalletTransactions(ctx context.Context, characterID int32) ([]*app.CharacterWalletTransaction, error) {
	rows, err := st.qRO.ListCharacterWalletTransactions(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list wallet transactions for character %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterWalletTransaction, len(rows))
	for i, r := range rows {
		oo[i] = characterWalletTransactionFromDBModel(
			r.CharacterWalletTransaction,
			r.EveEntity,
			r.EveType,
			r.EveGroup,
			r.EveCategory,
			r.LocationName,
			r.SystemSecurityStatus,
			r.RegionID,
			r.RegionName,
		)
	}
	return oo, nil
}

func characterWalletTransactionFromDBModel(
	o queries.CharacterWalletTransaction,
	client queries.EveEntity,
	et queries.EveType,
	eg queries.EveGroup,
	ec queries.EveCategory,
	locationName string,
	systemSecurityStatus sql.NullFloat64,
	regionID sql.NullInt64,
	regionName sql.NullString,
) *app.CharacterWalletTransaction {
	o2 := &app.CharacterWalletTransaction{
		Client:       eveEntityFromDBModel(client),
		Date:         o.Date,
		Type:         eveTypeFromDBModel(et, eg, ec),
		ID:           o.ID,
		IsBuy:        o.IsBuy,
		IsPersonal:   o.IsPersonal,
		JournalRefID: o.JournalRefID,
		Location: &app.EveLocationShort{
			ID:             o.LocationID,
			Name:           optional.From(locationName),
			SecurityStatus: optional.FromNullFloat64ToFloat32(systemSecurityStatus)},
		CharacterID:   int32(o.CharacterID),
		Quantity:      int32(o.Quantity),
		TransactionID: o.TransactionID,
		UnitPrice:     o.UnitPrice,
	}
	if regionID.Valid && regionName.Valid {
		o2.Region = &app.EntityShort[int32]{
			ID:   int32(regionID.Int64),
			Name: regionName.String,
		}
	}
	return o2
}
