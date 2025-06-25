package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) GetCharacterAttributes(ctx context.Context, characterID int32) (*app.CharacterAttributes, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreGetCharacterAttributesateCharacterAttributes character ID %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	o, err := st.qRO.GetCharacterAttributes(ctx, int64(characterID))
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	return characterAttributeFromDBModel(o), nil
}

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

func (st *Storage) UpdateOrCreateCharacterAttributes(ctx context.Context, arg UpdateOrCreateCharacterAttributesParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCharacterAttributes %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
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
	err := st.qRW.UpdateOrCreateCharacterAttributes(ctx, arg2)
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func characterAttributeFromDBModel(o queries.CharacterAttribute) *app.CharacterAttributes {
	o2 := &app.CharacterAttributes{
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
