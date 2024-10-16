package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
)

func (st *Storage) DeleteCharacterSkills(ctx context.Context, characterID int32, eveTypeIDs []int32) error {
	arg := queries.DeleteCharacterSkillsParams{
		CharacterID: int64(characterID),
		EveTypeIds:  convertNumericSlice[int32, int64](eveTypeIDs),
	}
	err := st.q.DeleteCharacterSkills(ctx, arg)
	if err != nil {
		return err
	}
	return nil
}

func (st *Storage) GetCharacterSkill(ctx context.Context, characterID int32, typeID int32) (*app.CharacterSkill, error) {
	arg := queries.GetCharacterSkillParams{
		CharacterID: int64(characterID),
		EveTypeID:   int64(typeID),
	}
	row, err := st.q.GetCharacterSkill(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get skill %d for character %d: %w", typeID, characterID, err)
	}
	t2 := characterSkillFromDBModel(row.CharacterSkill, row.EveType, row.EveGroup, row.EveCategory)
	return t2, nil
}

func (st *Storage) ListCharacterSkillIDs(ctx context.Context, characterID int32) ([]int32, error) {
	ids, err := st.q.ListCharacterSkillIDs(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("failed to list skill IDs for character %d: %w", characterID, err)
	}
	return convertNumericSlice[int64, int32](ids), nil
}

func (st *Storage) ListCharacterSkillProgress(ctx context.Context, characterID, eveGroupID int32) ([]app.ListCharacterSkillProgress, error) {
	arg := queries.ListCharacterSkillProgressParams{
		CharacterID: int64(characterID),
		EveGroupID:  int64(eveGroupID),
	}
	xx, err := st.q.ListCharacterSkillProgress(ctx, arg)
	if err != nil {
		return nil, err
	}
	oo := make([]app.ListCharacterSkillProgress, len(xx))
	for i, x := range xx {
		oo[i] = app.ListCharacterSkillProgress{
			ActiveSkillLevel:  int(x.ActiveSkillLevel.Int64),
			TypeDescription:   x.Description,
			TypeID:            int32(x.ID),
			TypeName:          x.Name,
			TrainedSkillLevel: int(x.TrainedSkillLevel.Int64),
		}
	}
	return oo, nil
}

func (st *Storage) ListCharacterSkillGroupsProgress(ctx context.Context, characterID int32) ([]app.ListCharacterSkillGroupProgress, error) {
	arg := queries.ListCharacterSkillGroupsProgressParams{
		CharacterID:   int64(characterID),
		EveCategoryID: app.EveCategorySkill,
	}
	xx, err := st.q.ListCharacterSkillGroupsProgress(ctx, arg)
	if err != nil {
		return nil, err
	}
	oo := make([]app.ListCharacterSkillGroupProgress, len(xx))
	for i, x := range xx {
		o := app.ListCharacterSkillGroupProgress{
			GroupID:   int32(x.EveGroupID),
			GroupName: x.EveGroupName,
			Total:     float64(x.Total),
		}
		if x.Trained.Valid {
			o.Trained = x.Trained.Float64
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
	if err := st.q.UpdateOrCreateCharacterSkill(ctx, arg2); err != nil {
		return fmt.Errorf("failed to update or create character skill for character %d: %w", arg.CharacterID, err)
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
