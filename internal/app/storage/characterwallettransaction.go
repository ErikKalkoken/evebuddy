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
	if arg.TransactionID == 0 {
		return fmt.Errorf("WalletTransaction ID can not be zero, Character %d", arg.CharacterID)
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

	if err := st.q.CreateCharacterWalletTransaction(ctx, arg2); err != nil {
		return fmt.Errorf("create wallet transaction: %+v: %w", arg2, err)
	}
	return nil
}

func (st *Storage) GetCharacterWalletTransaction(ctx context.Context, characterID int32, transactionID int64) (*app.CharacterWalletTransaction, error) {
	arg := queries.GetCharacterWalletTransactionParams{
		CharacterID:   int64(characterID),
		TransactionID: transactionID,
	}
	r, err := st.q.GetCharacterWalletTransaction(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get wallet transaction for character %d: %w", characterID, err)
	}
	o := characterWalletTransactionFromDBModel(
		r.CharacterWalletTransaction,
		r.EveEntity,
		r.EveTypeName,
		r.LocationName,
		r.SystemSecurityStatus,
	)
	return o, err
}

func (st *Storage) ListCharacterWalletTransactionIDs(ctx context.Context, characterID int32) (set.Set[int64], error) {
	ids, err := st.q.ListCharacterWalletTransactionIDs(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list wallet transaction ids for character %d: %w", characterID, err)
	}
	return set.NewFromSlice(ids), nil
}

func (st *Storage) ListCharacterWalletTransactions(ctx context.Context, characterID int32) ([]*app.CharacterWalletTransaction, error) {
	rows, err := st.q.ListCharacterWalletTransactions(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list wallet transactions for character %d: %w", characterID, err)
	}
	oo := make([]*app.CharacterWalletTransaction, len(rows))
	for i, r := range rows {
		oo[i] = characterWalletTransactionFromDBModel(
			r.CharacterWalletTransaction,
			r.EveEntity,
			r.EveTypeName,
			r.LocationName,
			r.SystemSecurityStatus,
		)
	}
	return oo, nil
}

func characterWalletTransactionFromDBModel(
	o queries.CharacterWalletTransaction,
	client queries.EveEntity,
	eveTypeName string,
	locationName string,
	systemSecurityStatus sql.NullFloat64,
) *app.CharacterWalletTransaction {
	var sss optional.Optional[float32]
	if systemSecurityStatus.Valid {
		sss.Set(float32(systemSecurityStatus.Float64))
	}
	o2 := &app.CharacterWalletTransaction{
		Client:               eveEntityFromDBModel(client),
		Date:                 o.Date,
		EveType:              &app.EntityShort[int32]{ID: int32(o.EveTypeID), Name: eveTypeName},
		ID:                   o.ID,
		IsBuy:                o.IsBuy,
		IsPersonal:           o.IsPersonal,
		JournalRefID:         o.JournalRefID,
		Location:             &app.EntityShort[int64]{ID: o.LocationID, Name: locationName},
		CharacterID:          int32(o.CharacterID),
		Quantity:             int32(o.Quantity),
		TransactionID:        o.TransactionID,
		UnitPrice:            o.UnitPrice,
		SystemSecurityStatus: sss,
	}
	return o2
}
