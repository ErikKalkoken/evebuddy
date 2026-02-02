package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func (st *Storage) ListCharacterShipsAbilities(ctx context.Context, characterID int32) ([]*app.CharacterShipAbility, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterShipsAbilities: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterShipsAbilities(ctx, int64(characterID))
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]*app.CharacterShipAbility, 0)
	for _, row := range rows {
		oo = append(oo, &app.CharacterShipAbility{
			Group:  app.EntityShort[int32]{ID: int32(row.GroupID), Name: row.GroupName},
			Type:   app.EntityShort[int32]{ID: int32(row.TypeID), Name: row.TypeName},
			CanFly: row.CanFly,
		})
	}
	return oo, nil
}

func (st *Storage) ListCharacterShipSkills(ctx context.Context, characterID, shipTypeID int32) ([]*app.CharacterShipSkill, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterShipSkills: %d %d: %w", characterID, shipTypeID, err)
	}
	rows, err := st.qRO.ListCharacterShipSkills(ctx, queries.ListCharacterShipSkillsParams{
		CharacterID: int64(characterID),
		ShipTypeID:  int64(shipTypeID),
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]*app.CharacterShipSkill, 0)
	for _, r := range rows {
		oo = append(oo, characterShiSkillFromDBModel(r))
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
