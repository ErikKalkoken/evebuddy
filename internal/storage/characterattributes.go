package storage

import (
	"context"
	"database/sql"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type UpdateOrCreateCharacterAttributesParams struct {
	ID            int64
	BonusRemaps   sql.NullInt64
	CharacterID   int32
	Charisma      int
	Intelligence  int
	LastRemapDate sql.NullTime
	Memory        int
	Perception    int
	Willpower     int
}

func (r *Storage) GetCharacterAttributes(ctx context.Context, characterID int32) (*model.CharacterAttribute, error) {
	o, err := r.q.GetCharacterAttributes(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	return characterAttributeFromDBModel(o), nil
}

func (r *Storage) UpdateOrCreateCharacterAttributes(ctx context.Context, arg UpdateOrCreateCharacterAttributesParams) error {
	arg2 := queries.UpdateOrCreateCharacterAttributesParams{
		CharacterID:   int64(arg.CharacterID),
		BonusRemaps:   arg.BonusRemaps,
		Charisma:      int64(arg.Charisma),
		Intelligence:  int64(arg.Intelligence),
		LastRemapDate: arg.LastRemapDate,
		Memory:        int64(arg.Memory),
		Perception:    int64(arg.Perception),
		Willpower:     int64(arg.Willpower),
	}
	return r.q.UpdateOrCreateCharacterAttributes(ctx, arg2)
}

func characterAttributeFromDBModel(o queries.CharacterAttribute) *model.CharacterAttribute {
	o2 := &model.CharacterAttribute{
		ID:            o.ID,
		BonusRemaps:   o.BonusRemaps,
		CharacterID:   int32(o.CharacterID),
		Charisma:      int(o.Charisma),
		Intelligence:  int(o.Intelligence),
		LastRemapDate: o.LastRemapDate,
		Memory:        int(o.Memory),
		Perception:    int(o.Perception),
		Willpower:     int(o.Willpower),
	}
	return o2
}
