package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

func (st *Storage) GetCharacterAttributes(ctx context.Context, characterID int64) (*app.CharacterAttributes, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterAttributes character ID %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	o, err := st.qRO.GetCharacterAttributes(ctx, characterID)
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	return characterAttributeFromDBModel(o), nil
}

type UpdateOrCreateCharacterAttributesParams struct {
	ID            int64
	BonusRemaps   optional.Optional[int64]
	CharacterID   int64
	Charisma      int64
	Intelligence  int64
	LastRemapDate optional.Optional[time.Time]
	Memory        int64
	Perception    int64
	Willpower     int64
}

func (st *Storage) UpdateOrCreateCharacterAttributes(ctx context.Context, arg UpdateOrCreateCharacterAttributesParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCharacterAttributes %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateCharacterAttributes(ctx, queries.UpdateOrCreateCharacterAttributesParams{
		CharacterID:   arg.CharacterID,
		BonusRemaps:   arg.BonusRemaps.ValueOrZero(),
		Charisma:      arg.Charisma,
		Intelligence:  arg.Intelligence,
		Memory:        arg.Memory,
		Perception:    arg.Perception,
		Willpower:     arg.Willpower,
		LastRemapDate: optional.ToNullTime(arg.LastRemapDate),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func characterAttributeFromDBModel(o queries.CharacterAttribute) *app.CharacterAttributes {
	o2 := &app.CharacterAttributes{
		BonusRemaps:   optional.FromZeroValue(o.BonusRemaps),
		CharacterID:   o.CharacterID,
		Charisma:      o.Charisma,
		ID:            o.ID,
		Intelligence:  o.Intelligence,
		LastRemapDate: optional.FromNullTime(o.LastRemapDate),
		Memory:        o.Memory,
		Perception:    o.Perception,
		Willpower:     o.Willpower,
	}
	return o2
}
