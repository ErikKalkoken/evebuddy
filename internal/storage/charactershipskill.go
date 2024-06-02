package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

func (st *Storage) ListCharacterShipsAbilities(ctx context.Context, characterID int32, search string) ([]*model.CharacterShipAbility, error) {
	arg := queries.ListCharacterShipsAbilitiesParams{
		CharacterID: int64(characterID),
		Name:        search,
	}
	rows, err := st.q.ListCharacterShipsAbilities(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list ship abilities for character %d and search %s: %w", characterID, search, err)
	}
	oo := make([]*model.CharacterShipAbility, len(rows))
	for i, row := range rows {
		o := &model.CharacterShipAbility{
			Group:  model.EntityShort[int32]{ID: int32(row.GroupID), Name: row.GroupName},
			Type:   model.EntityShort[int32]{ID: int32(row.TypeID), Name: row.TypeName},
			CanFly: row.CanFly,
		}
		oo[i] = o
	}
	return oo, nil
}
