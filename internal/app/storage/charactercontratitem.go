package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

type CreateCharacterContractItemParams struct {
	ContractID  int64
	IsIncluded  bool
	IsSingleton bool
	Quantity    int32
	RawQuantity int32
	RecordID    int64
	TypeID      int32
}

func (st *Storage) CreateCharacterContractItem(ctx context.Context, arg CreateCharacterContractItemParams) error {
	if arg.ContractID == 0 || arg.TypeID == 0 {
		return fmt.Errorf("create contract item. Mandatory fields not set: %v", arg)
	}
	arg2 := queries.CreateCharacterContractItemParams{
		ContractID:  arg.ContractID,
		IsIncluded:  arg.IsIncluded,
		IsSingleton: arg.IsSingleton,
		Quantity:    int64(arg.Quantity),
		RawQuantity: int64(arg.RawQuantity),
		RecordID:    arg.RecordID,
		TypeID:      int64(arg.TypeID),
	}
	if err := st.q.CreateCharacterContractItem(ctx, arg2); err != nil {
		return fmt.Errorf("create contract item: %+v: %w", arg, err)
	}
	return nil
}

func (st *Storage) GetCharacterContractItem(ctx context.Context, contractID, recordID int64) (*app.CharacterContractItem, error) {
	arg := queries.GetCharacterContractItemParams{
		ContractID: contractID,
		RecordID:   recordID,
	}
	r, err := st.q.GetCharacterContractItem(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get contract item %+v: %w", arg, err)
	}
	return characterContractItemFromDBModel(r.CharacterContractItem, r.EveType, r.EveGroup, r.EveCategory), err
}

func (st *Storage) ListCharacterContractItems(ctx context.Context, contractID int64) ([]*app.CharacterContractItem, error) {
	rows, err := st.q.ListCharacterContractItems(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("list items for contract %d: %w", contractID, err)
	}
	oo := make([]*app.CharacterContractItem, len(rows))
	for i, r := range rows {
		oo[i] = characterContractItemFromDBModel(r.CharacterContractItem, r.EveType, r.EveGroup, r.EveCategory)
	}
	return oo, nil
}

func characterContractItemFromDBModel(o queries.CharacterContractItem, t queries.EveType, g queries.EveGroup, c queries.EveCategory) *app.CharacterContractItem {
	o2 := &app.CharacterContractItem{
		ContractID:  o.ContractID,
		IsIncluded:  o.IsIncluded,
		IsSingleton: o.IsSingleton,
		Quantity:    int(o.Quantity),
		RawQuantity: int(o.RawQuantity),
		RecordID:    o.RecordID,
		Type:        eveTypeFromDBModel(t, g, c),
	}
	return o2
}
