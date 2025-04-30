package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func (st *Storage) DeleteCharacterSkills(ctx context.Context, characterID int32, eveTypeIDs []int32) error {
	arg := queries.DeleteCharacterSkillsParams{
		CharacterID: int64(characterID),
		EveTypeIds:  convertNumericSlice[int64](eveTypeIDs),
	}
	err := st.qRW.DeleteCharacterSkills(ctx, arg)
	if err != nil {
		return fmt.Errorf("delete skills for character %d: %w", characterID, err)
	}
	return nil
}

func (st *Storage) GetCharacterSkill(ctx context.Context, characterID int32, typeID int32) (*app.CharacterSkill, error) {
	arg := queries.GetCharacterSkillParams{
		CharacterID: int64(characterID),
		EveTypeID:   int64(typeID),
	}
	r, err := st.qRO.GetCharacterSkill(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = app.ErrNotFound
		}
		return nil, fmt.Errorf("get skill %d for character %d: %w", typeID, characterID, err)
	}
	t2 := characterSkillFromDBModel(r.CharacterSkill, r.EveType, r.EveGroup, r.EveCategory)
	return t2, nil
}

func (st *Storage) ListCharacterSkillIDs(ctx context.Context, characterID int32) (set.Set[int32], error) {
	ids1, err := st.qRO.ListCharacterSkillIDs(ctx, int64(characterID))
	if err != nil {
		return set.Set[int32]{}, fmt.Errorf("list skill ids for character %d: %w", characterID, err)
	}
	ids2 := set.Of(convertNumericSlice[int32](ids1)...)
	return ids2, nil
}

func (st *Storage) ListCharacterSkillProgress(ctx context.Context, characterID, eveGroupID int32) ([]app.ListSkillProgress, error) {
	arg := queries.ListCharacterSkillProgressParams{
		CharacterID: int64(characterID),
		EveGroupID:  int64(eveGroupID),
	}
	rows, err := st.qRO.ListCharacterSkillProgress(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list skill progress for character %d: %w", characterID, err)
	}
	oo := make([]app.ListSkillProgress, len(rows))
	for i, r := range rows {
		oo[i] = app.ListSkillProgress{
			ActiveSkillLevel:  int(r.ActiveSkillLevel.Int64),
			TypeDescription:   r.Description,
			TypeID:            int32(r.ID),
			TypeName:          r.Name,
			TrainedSkillLevel: int(r.TrainedSkillLevel.Int64),
		}
	}
	return oo, nil
}

func (st *Storage) ListCharacterSkillGroupsProgress(ctx context.Context, characterID int32) ([]app.ListCharacterSkillGroupProgress, error) {
	arg := queries.ListCharacterSkillGroupsProgressParams{
		CharacterID:   int64(characterID),
		EveCategoryID: app.EveCategorySkill,
	}
	rows, err := st.qRO.ListCharacterSkillGroupsProgress(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("list skill groups progress for character %d: %w", characterID, err)
	}
	oo := make([]app.ListCharacterSkillGroupProgress, len(rows))
	for i, r := range rows {
		o := app.ListCharacterSkillGroupProgress{
			GroupID:   int32(r.EveGroupID),
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
	ActiveSkillLevel   int
	EveTypeID          int32
	SkillPointsInSkill int
	CharacterID        int32
	TrainedSkillLevel  int
}

func (st *Storage) UpdateOrCreateCharacterSkill(ctx context.Context, arg UpdateOrCreateCharacterSkillParams) error {
	arg2 := queries.UpdateOrCreateCharacterSkillParams{
		ActiveSkillLevel:   int64(arg.ActiveSkillLevel),
		EveTypeID:          int64(arg.EveTypeID),
		SkillPointsInSkill: int64(arg.SkillPointsInSkill),
		CharacterID:        int64(arg.CharacterID),
		TrainedSkillLevel:  int64(arg.TrainedSkillLevel),
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
		ActiveSkillLevel:   int(o.ActiveSkillLevel),
		CharacterID:        int32(o.CharacterID),
		EveType:            eveTypeFromDBModel(t, g, c),
		ID:                 o.ID,
		SkillPointsInSkill: int(o.SkillPointsInSkill),
		TrainedSkillLevel:  int(o.TrainedSkillLevel),
	}
}
