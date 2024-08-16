package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/pkg/optional"
)

func (st *Storage) ListCharacterShipsAbilities(ctx context.Context, characterID int32, search string) ([]*app.CharacterShipAbility, error) {
	arg := queries.ListCharacterShipsAbilitiesParams{
		CharacterID: int64(characterID),
		Name:        search,
	}
	rows, err := st.q.ListCharacterShipsAbilities(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list ship abilities for character %d and search %s: %w", characterID, search, err)
	}
	oo := make([]*app.CharacterShipAbility, len(rows))
	for i, row := range rows {
		o := &app.CharacterShipAbility{
			Group:  app.EntityShort[int32]{ID: int32(row.GroupID), Name: row.GroupName},
			Type:   app.EntityShort[int32]{ID: int32(row.TypeID), Name: row.TypeName},
			CanFly: row.CanFly,
		}
		oo[i] = o
	}
	return oo, nil
}

func (st *Storage) ListCharacterShipSkills(ctx context.Context, characterID, shipTypeID int32) ([]*app.CharacterShipSkill, error) {
	arg := queries.ListCharacterShipSkillsParams{
		CharacterID: int64(characterID),
		ShipTypeID:  int64(shipTypeID),
	}
	rows, err := st.q.ListCharacterShipSkills(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("failed to list character ship skills for character %d and type %d: %w", characterID, shipTypeID, err)
	}
	oo := make([]*app.CharacterShipSkill, len(rows))
	for i, r := range rows {
		oo[i] = characterShiSkillFromDBModel(r)
	}
	return oo, nil
}

func characterShiSkillFromDBModel(r queries.ListCharacterShipSkillsRow) *app.CharacterShipSkill {
	css := &app.CharacterShipSkill{
		Rank:        uint(r.Rank),
		ShipTypeID:  int32(r.ShipTypeID),
		SkillTypeID: int32(r.SkillTypeID),
		SkillName:   r.SkillName,
		SkillLevel:  uint(r.SkillLevel),
	}
	if r.ActiveSkillLevel.Valid {
		css.ActiveSkillLevel = optional.New(int(r.ActiveSkillLevel.Int64))
	}
	if r.TrainedSkillLevel.Valid {
		css.TrainedSkillLevel = optional.New(int(r.TrainedSkillLevel.Int64))
	}
	return css
}
