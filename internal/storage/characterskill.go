package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	islices "github.com/ErikKalkoken/evebuddy/internal/helper/slices"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

func (st *Storage) DeleteExcludedCharacterSkills(ctx context.Context, characterID int32, eveTypeIDs []int32) error {
	arg := queries.DeleteExcludedCharacterSkillsParams{
		CharacterID: int64(characterID),
		EveTypeIds:  islices.ConvertNumeric[int32, int64](eveTypeIDs),
	}
	err := st.q.DeleteExcludedCharacterSkills(ctx, arg)
	if err != nil {
		return err
	}
	return nil
}

func (st *Storage) GetCharacterSkill(ctx context.Context, characterID int32, eveTypeID int32) (*model.CharacterSkill, error) {
	arg := queries.GetCharacterSkillParams{
		CharacterID: int64(characterID),
		EveTypeID:   int64(eveTypeID),
	}
	row, err := st.q.GetCharacterSkill(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get CharacterSkill for character %d: %w", characterID, err)
	}
	t2 := characterSkillFromDBModel(row.CharacterSkill, row.EveType, row.EveGroup, row.EveCategory)
	return t2, nil
}

func (st *Storage) ListCharacterSkillProgress(ctx context.Context, characterID, eveGroupID int32) ([]model.ListCharacterSkillProgress, error) {
	arg := queries.ListCharacterSkillProgressParams{
		CharacterID: int64(characterID),
		EveGroupID:  int64(eveGroupID),
	}
	xx, err := st.q.ListCharacterSkillProgress(ctx, arg)
	if err != nil {
		return nil, err
	}
	oo := make([]model.ListCharacterSkillProgress, len(xx))
	for i, x := range xx {
		oo[i] = model.ListCharacterSkillProgress{
			ActiveSkillLevel:  int(x.ActiveSkillLevel.Int64),
			TypeDescription:   x.Description,
			TypeID:            int32(x.ID),
			TypeName:          x.Name,
			TrainedSkillLevel: int(x.TrainedSkillLevel.Int64),
		}
	}
	return oo, nil
}

func (st *Storage) ListCharacterSkillGroupsProgress(ctx context.Context, characterID int32) ([]model.ListCharacterSkillGroupProgress, error) {
	arg := queries.ListCharacterSkillGroupsProgressParams{
		CharacterID:   int64(characterID),
		EveCategoryID: model.EveCategorySkill,
	}
	xx, err := st.q.ListCharacterSkillGroupsProgress(ctx, arg)
	if err != nil {
		return nil, err
	}
	oo := make([]model.ListCharacterSkillGroupProgress, len(xx))
	for i, x := range xx {
		o := model.ListCharacterSkillGroupProgress{
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

func characterSkillFromDBModel(o queries.CharacterSkill, t queries.EveType, g queries.EveGroup, c queries.EveCategory) *model.CharacterSkill {
	if o.CharacterID == 0 {
		panic("missing character ID")
	}
	return &model.CharacterSkill{
		ActiveSkillLevel:   int(o.ActiveSkillLevel),
		CharacterID:        int32(o.CharacterID),
		EveType:            eveTypeFromDBModel(t, g, c),
		ID:                 o.ID,
		SkillPointsInSkill: int(o.SkillPointsInSkill),
		TrainedSkillLevel:  int(o.TrainedSkillLevel),
	}
}
