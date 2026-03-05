package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func (st *Storage) ListCharacterShipsAbilities(ctx context.Context, characterID int64) ([]*app.CharacterShipAbility, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterShipsAbilities: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterShipsAbilities(ctx, characterID)
	if err != nil {
		return nil, wrapErr(err)
	}
	var oo []*app.CharacterShipAbility
	for _, row := range rows {
		oo = append(oo, &app.CharacterShipAbility{
			Group:  app.EntityShort{ID: row.GroupID, Name: row.GroupName},
			Type:   app.EntityShort{ID: row.TypeID, Name: row.TypeName},
			CanFly: row.CanFly,
		})
	}
	return oo, nil
}

func (st *Storage) ListCharacterShipSkills(ctx context.Context, characterID, shipTypeID int64) ([]*app.CharacterShipSkill, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterShipSkills: %d %d: %w", characterID, shipTypeID, err)
	}
	rows, err := st.qRO.ListCharacterShipSkills(ctx, queries.ListCharacterShipSkillsParams{
		CharacterID: characterID,
		ShipTypeID:  shipTypeID,
	})
	if err != nil {
		return nil, wrapErr(err)
	}
	var oo []*app.CharacterShipSkill
	for _, r := range rows {
		oo = append(oo, characterShiSkillFromDBModel(r))
	}
	return oo, nil
}

func characterShiSkillFromDBModel(r queries.ListCharacterShipSkillsRow) *app.CharacterShipSkill {
	css := &app.CharacterShipSkill{
		Rank:        uint(r.Rank),
		ShipTypeID:  r.ShipTypeID,
		SkillTypeID: r.SkillTypeID,
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
