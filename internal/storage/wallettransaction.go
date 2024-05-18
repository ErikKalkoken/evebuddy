package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type CreateWalletTransactionParams struct {
	ClientID      int32
	Date          time.Time
	EveTypeID     int32
	IsBuy         bool
	IsPersonal    bool
	JournalRefID  int64
	LocationID    int64
	MyCharacterID int32
	Quantity      int32
	TransactionID int64
	UnitPrice     float64
}

func (r *Storage) CreateWalletTransaction(ctx context.Context, arg CreateWalletTransactionParams) error {
	if arg.TransactionID == 0 {
		return fmt.Errorf("WalletTransaction ID can not be zero, Character %d", arg.MyCharacterID)
	}
	arg2 := queries.CreateWalletTransactionParams{
		ClientID:      int64(arg.ClientID),
		Date:          arg.Date,
		EveTypeID:     int64(arg.EveTypeID),
		IsBuy:         arg.IsBuy,
		IsPersonal:    arg.IsPersonal,
		JournalRefID:  arg.JournalRefID,
		LocationID:    arg.LocationID,
		MyCharacterID: int64(arg.MyCharacterID),
		Quantity:      int64(arg.Quantity),
		TransactionID: arg.TransactionID,
		UnitPrice:     arg.UnitPrice,
	}

	err := r.q.CreateWalletTransaction(ctx, arg2)
	return err
}

func (r *Storage) GetWalletTransaction(ctx context.Context, characterID int32, transactionID int64) (*model.WalletTransaction, error) {
	arg := queries.GetWalletTransactionParams{
		MyCharacterID: int64(characterID),
		TransactionID: transactionID,
	}
	row, err := r.q.GetWalletTransaction(ctx, arg)
	if err != nil {
		return nil, err
	}
	return walletTransactionFromDBModel(row.WalletTransaction, row.EveEntity, row.EveTypeName, row.LocationName), err
}

func (r *Storage) ListWalletTransactionIDs(ctx context.Context, characterID int32) ([]int64, error) {
	return r.q.ListWalletTransactionIDs(ctx, int64(characterID))
}

func (r *Storage) ListWalletTransactions(ctx context.Context, characterID int32) ([]*model.WalletTransaction, error) {
	rows, err := r.q.ListWalletTransactions(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	ee := make([]*model.WalletTransaction, len(rows))
	for i, row := range rows {
		ee[i] = walletTransactionFromDBModel(row.WalletTransaction, row.EveEntity, row.EveTypeName, row.LocationName)
	}
	return ee, nil
}

func walletTransactionFromDBModel(
	t queries.WalletTransaction,
	client queries.EveEntity,
	eveTypeName string,
	locationName string,
) *model.WalletTransaction {
	x := &model.WalletTransaction{
		Client:        eveEntityFromDBModel(client),
		Date:          t.Date,
		EveTypeID:     int32(t.EveTypeID),
		EveTypeName:   eveTypeName,
		IsBuy:         t.IsBuy,
		IsPersonal:    t.IsPersonal,
		JournalRefID:  t.JournalRefID,
		LocationID:    t.LocationID,
		LocationName:  locationName,
		MyCharacterID: int32(t.MyCharacterID),
		Quantity:      int32(t.Quantity),
		TransactionID: t.TransactionID,
		UnitPrice:     t.UnitPrice,
	}
	return x
}
