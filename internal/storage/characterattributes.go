package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type UpdateOrCreateCharacterAttributesParams struct {
	ID            int64
	BonusRemaps   int
	CharacterID   int32
	Charisma      int
	Intelligence  int
	LastRemapDate time.Time
	Memory        int
	Perception    int
	Willpower     int
}

func (r *Storage) GetCharacterAttributes(ctx context.Context, characterID int32) (*model.CharacterAttributes, error) {
	o, err := r.q.GetCharacterAttributes(ctx, int64(characterID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, err
	}
	return characterAttributeFromDBModel(o), nil
}

func (r *Storage) UpdateOrCreateCharacterAttributes(ctx context.Context, arg UpdateOrCreateCharacterAttributesParams) error {
	arg2 := queries.UpdateOrCreateCharacterAttributesParams{
		CharacterID:  int64(arg.CharacterID),
		BonusRemaps:  int64(arg.BonusRemaps),
		Charisma:     int64(arg.Charisma),
		Intelligence: int64(arg.Intelligence),
		Memory:       int64(arg.Memory),
		Perception:   int64(arg.Perception),
		Willpower:    int64(arg.Willpower),
	}
	if !arg.LastRemapDate.IsZero() {
		arg2.LastRemapDate.Time = arg.LastRemapDate
		arg2.LastRemapDate.Valid = true
	}
	return r.q.UpdateOrCreateCharacterAttributes(ctx, arg2)
}

func characterAttributeFromDBModel(o queries.CharacterAttribute) *model.CharacterAttributes {
	o2 := &model.CharacterAttributes{
		ID:           o.ID,
		BonusRemaps:  int(o.BonusRemaps),
		CharacterID:  int32(o.CharacterID),
		Charisma:     int(o.Charisma),
		Intelligence: int(o.Intelligence),
		Memory:       int(o.Memory),
		Perception:   int(o.Perception),
		Willpower:    int(o.Willpower),
	}
	if o.LastRemapDate.Valid {
		o2.LastRemapDate = o.LastRemapDate.Time
	}
	return o2
}
