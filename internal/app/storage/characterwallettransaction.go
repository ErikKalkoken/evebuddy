package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
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

	err := st.q.CreateCharacterWalletTransaction(ctx, arg2)
	return err
}

func (st *Storage) GetCharacterWalletTransaction(ctx context.Context, characterID int32, transactionID int64) (*app.CharacterWalletTransaction, error) {
	arg := queries.GetCharacterWalletTransactionParams{
		CharacterID:   int64(characterID),
		TransactionID: transactionID,
	}
	row, err := st.q.GetCharacterWalletTransaction(ctx, arg)
	if err != nil {
		return nil, err
	}
	return characterWalletTransactionFromDBModel(row.CharacterWalletTransaction, row.EveEntity, row.EveTypeName, row.LocationName), err
}

func (st *Storage) ListCharacterWalletTransactionIDs(ctx context.Context, characterID int32) ([]int64, error) {
	return st.q.ListCharacterWalletTransactionIDs(ctx, int64(characterID))
}

func (st *Storage) ListCharacterWalletTransactions(ctx context.Context, characterID int32) ([]*app.CharacterWalletTransaction, error) {
	rows, err := st.q.ListCharacterWalletTransactions(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	ee := make([]*app.CharacterWalletTransaction, len(rows))
	for i, row := range rows {
		ee[i] = characterWalletTransactionFromDBModel(row.CharacterWalletTransaction, row.EveEntity, row.EveTypeName, row.LocationName)
	}
	return ee, nil
}

func characterWalletTransactionFromDBModel(
	o queries.CharacterWalletTransaction,
	client queries.EveEntity,
	eveTypeName string,
	locationName string,
) *app.CharacterWalletTransaction {
	o2 := &app.CharacterWalletTransaction{
		Client:        eveEntityFromDBModel(client),
		Date:          o.Date,
		EveType:       &app.EntityShort[int32]{ID: int32(o.EveTypeID), Name: eveTypeName},
		ID:            o.ID,
		IsBuy:         o.IsBuy,
		IsPersonal:    o.IsPersonal,
		JournalRefID:  o.JournalRefID,
		Location:      &app.EntityShort[int64]{ID: o.LocationID, Name: locationName},
		CharacterID:   int32(o.CharacterID),
		Quantity:      int32(o.Quantity),
		TransactionID: o.TransactionID,
		UnitPrice:     o.UnitPrice,
	}
	return o2
}
