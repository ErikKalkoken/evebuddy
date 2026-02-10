package storage

import (
	"context"
	"fmt"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) DeleteCharacterSkills(ctx context.Context, characterID int64, eveTypeIDs set.Set[int64]) error {
	arg := queries.DeleteCharacterSkillsParams{
		CharacterID: characterID,
		EveTypeIds:  convertNumericSet[int64](eveTypeIDs),
	}
	err := st.qRW.DeleteCharacterSkills(ctx, arg)
	if err != nil {
		return fmt.Errorf("delete skills for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) GetCharacterSkill(ctx context.Context, characterID int64, typeID int64) (*app.CharacterSkill, error) {
	arg := queries.GetCharacterSkillParams{
		CharacterID: characterID,
		EveTypeID:   typeID,
	}
	r, err := st.qRO.GetCharacterSkill(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get skill %d for character %d: %w", typeID, characterID, convertGetError(err))
	}
	t2 := characterSkillFromDBModel(r.CharacterSkill, r.EveType, r.EveGroup, r.EveCategory)
	return t2, nil
}

func (st *Storage) ListAllCharactersActiveSkillLevels(ctx context.Context, typeID int64) ([]app.CharacterActiveSkillLevel, error) {
	rows, err := st.qRO.ListCharactersActiveSkillLevels(ctx, typeID)
	if err != nil {
		return nil, fmt.Errorf("ListCharactersActiveSkillLevels for type ID: %d: %w", typeID, err)
	}
	oo := make([]app.CharacterActiveSkillLevel, len(rows))
	for i, r := range rows {
		oo[i] = app.CharacterActiveSkillLevel{
			CharacterID: r.CharacterID,
			Level:       int(r.Level),
			TypeID:      typeID,
		}
	}
	return oo, nil

}

func (st *Storage) ListCharacterSkillIDs(ctx context.Context, characterID int64) (set.Set[int64], error) {
	ids, err := st.qRO.ListCharacterSkillIDs(ctx, characterID)
	if err != nil {
		return set.Set[int64]{}, fmt.Errorf("list skill ids for character %d: %w", characterID, err)
	}
	return set.Of(ids...), nil
}

func (st *Storage) ListCharacterSkillProgress(ctx context.Context, characterID, eveGroupID int64) ([]app.ListSkillProgress, error) {
	arg := queries.ListCharacterSkillProgressParams{
		CharacterID: characterID,
		EveGroupID:  eveGroupID,
	}
	rows, err := st.qRO.ListCharacterSkillProgress(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list skill progress for character %d: %w", characterID, err)
	}
	oo := make([]app.ListSkillProgress, len(rows))
	for i, r := range rows {
		oo[i] = app.ListSkillProgress{
			ActiveSkillLevel:  r.ActiveSkillLevel.Int64,
			TypeDescription:   r.Description,
			TypeID:            r.ID,
			TypeName:          r.Name,
			TrainedSkillLevel: r.TrainedSkillLevel.Int64,
		}
	}
	return oo, nil
}

func (st *Storage) ListCharacterSkillGroupsProgress(ctx context.Context, characterID int64) ([]app.ListCharacterSkillGroupProgress, error) {
	arg := queries.ListCharacterSkillGroupsProgressParams{
		CharacterID:   characterID,
		EveCategoryID: app.EveCategorySkill,
	}
	rows, err := st.qRO.ListCharacterSkillGroupsProgress(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list skill groups progress for character %d: %w", characterID, err)
	}
	oo := make([]app.ListCharacterSkillGroupProgress, len(rows))
	for i, r := range rows {
		o := app.ListCharacterSkillGroupProgress{
			GroupID:   r.EveGroupID,
			GroupName: r.EveGroupName,
			Total:     float64(r.Total),
		}
		if r.Trained.Valid {
			o.Trained = r.Trained.Float64
		}
		oo[i] = o
	}
	return oo, nil
}

type UpdateOrCreateCharacterSkillParams struct {
	ActiveSkillLevel   int64
	EveTypeID          int64
	SkillPointsInSkill int64
	CharacterID        int64
	TrainedSkillLevel  int64
}

func (st *Storage) UpdateOrCreateCharacterSkill(ctx context.Context, arg UpdateOrCreateCharacterSkillParams) error {
	arg2 := queries.UpdateOrCreateCharacterSkillParams{
		ActiveSkillLevel:   arg.ActiveSkillLevel,
		EveTypeID:          arg.EveTypeID,
		SkillPointsInSkill: arg.SkillPointsInSkill,
		CharacterID:        arg.CharacterID,
		TrainedSkillLevel:  arg.TrainedSkillLevel,
	}
	if err := st.qRW.UpdateOrCreateCharacterSkill(ctx, arg2); err != nil {
		return fmt.Errorf("update or create character skill for character %d: %w", arg.CharacterID, err)
	}
	return nil
}

func characterSkillFromDBModel(o queries.CharacterSkill, t queries.EveType, g queries.EveGroup, c queries.EveCategory) *app.CharacterSkill {
	if o.CharacterID == 0 {
		panic("missing character ID")
	}
	return &app.CharacterSkill{
		ActiveSkillLevel:   o.ActiveSkillLevel,
		CharacterID:        o.CharacterID,
		EveType:            eveTypeFromDBModel(t, g, c),
		ID:                 o.ID,
		SkillPointsInSkill: o.SkillPointsInSkill,
		TrainedSkillLevel:  o.TrainedSkillLevel,
	}
}
