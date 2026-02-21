package storage

import (
	"context"
	"fmt"
	"slices"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) DeleteCharacterSkills(ctx context.Context, characterID int64, typeIDs set.Set[int64]) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("DeleteCharacterSkills: %d %v: %w", characterID, typeIDs, err)
	}
	if characterID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.DeleteCharacterSkills(ctx, queries.DeleteCharacterSkillsParams{
		CharacterID: characterID,
		EveTypeIds:  slices.Collect(typeIDs.All()),
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func (st *Storage) GetCharacterSkill(ctx context.Context, characterID int64, typeID int64) (*app.CharacterSkill, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterSkill: %d %d: %w", characterID, typeID, err)
	}
	if characterID == 0 || typeID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	r, err := st.qRO.GetCharacterSkill(ctx, queries.GetCharacterSkillParams{
		CharacterID: characterID,
		EveTypeID:   typeID,
	})
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	t2 := characterSkillFromDBModel(r.CharacterSkill, r.EveType, r.EveGroup, r.EveCategory)
	return t2, nil
}

func (st *Storage) ListCharacterSkills(ctx context.Context, characterID int64) ([]*app.CharacterSkill, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterSkills: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterSkills(ctx, characterID)
	if err != nil {
		return nil, wrapErr(err)
	}
	var oo []*app.CharacterSkill
	for _, r := range rows {
		oo = append(oo, characterSkillFromDBModel(r.CharacterSkill, r.EveType, r.EveGroup, r.EveCategory))
	}
	return oo, nil
}
func (st *Storage) ListAllCharactersActiveSkillLevels(ctx context.Context, typeID int64) ([]app.CharacterActiveSkillLevel, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListAllCharactersActiveSkillLevels: %d: %w", typeID, err)
	}
	if typeID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharactersActiveSkillLevels(ctx, typeID)
	if err != nil {
		return nil, wrapErr(err)
	}
	var oo []app.CharacterActiveSkillLevel
	for _, r := range rows {
		oo = append(oo, app.CharacterActiveSkillLevel{
			CharacterID: r.CharacterID,
			Level:       int(r.Level),
			TypeID:      typeID,
		})
	}
	return oo, nil

}

func (st *Storage) ListCharacterSkillIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterSkillIDs: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return set.Set[int64]{}, wrapErr(app.ErrInvalid)
	}
	ids, err := st.qRO.ListCharacterSkillIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, wrapErr(err)
	}
	return set.Collect(slices.Values(ids)), nil
}

type UpdateOrCreateCharacterSkillParams struct {
	ActiveSkillLevel   int64
	TypeID             int64
	SkillPointsInSkill int64
	CharacterID        int64
	TrainedSkillLevel  int64
}

func (st *Storage) UpdateOrCreateCharacterSkill(ctx context.Context, arg UpdateOrCreateCharacterSkillParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("UpdateOrCreateCharacterSkill: %v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.TypeID == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := st.qRW.UpdateOrCreateCharacterSkill(ctx, queries.UpdateOrCreateCharacterSkillParams{
		ActiveSkillLevel:   arg.ActiveSkillLevel,
		EveTypeID:          arg.TypeID,
		SkillPointsInSkill: arg.SkillPointsInSkill,
		CharacterID:        arg.CharacterID,
		TrainedSkillLevel:  arg.TrainedSkillLevel,
	})
	if err != nil {
		return wrapErr(err)
	}
	return nil
}

func characterSkillFromDBModel(o queries.CharacterSkill, t queries.EveType, g queries.EveGroup, c queries.EveCategory) *app.CharacterSkill {
	return &app.CharacterSkill{
		ActiveSkillLevel:   o.ActiveSkillLevel,
		CharacterID:        o.CharacterID,
		Type:               eveTypeFromDBModel(t, g, c),
		SkillPointsInSkill: o.SkillPointsInSkill,
		TrainedSkillLevel:  o.TrainedSkillLevel,
	}
}
