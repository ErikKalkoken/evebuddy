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

func (r *Storage) DeleteExcludedCharacterSkills(ctx context.Context, characterID int32, eveTypeIDs []int32) error {
	arg := queries.DeleteExcludedCharacterSkillsParams{
		MyCharacterID: int64(characterID),
		EveTypeIds:    islices.ConvertNumeric[int32, int64](eveTypeIDs),
	}
	err := r.q.DeleteExcludedCharacterSkills(ctx, arg)
	if err != nil {
		return err
	}
	return nil
}

func (r *Storage) GetCharacterSkill(ctx context.Context, characterID int32, eveTypeID int32) (*model.CharacterSkill, error) {
	arg := queries.GetCharacterSkillParams{
		MyCharacterID: int64(characterID),
		EveTypeID:     int64(eveTypeID),
	}
	row, err := r.q.GetCharacterSkill(ctx, arg)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("failed to get CharacterSkill for character %d: %w", characterID, err)
	}
	t2 := characterSkillFromDBModel(row.CharacterSkill, row.EveType, row.EveGroup, row.EveCategory)
	return t2, nil
}

func (r *Storage) ListCharacterSkillProgress(ctx context.Context, characterID, eveGroupID int32) ([]model.ListCharacterSkillProgress, error) {
	arg := queries.ListCharacterSkillProgressParams{
		MyCharacterID: int64(characterID),
		EveGroupID:    int64(eveGroupID),
	}
	xx, err := r.q.ListCharacterSkillProgress(ctx, arg)
	if err != nil {
		return nil, err
	}
	oo := make([]model.ListCharacterSkillProgress, len(xx))
	for i, x := range xx {
		oo[i] = model.ListCharacterSkillProgress{
			ActiveSkillLevel:  int(x.ActiveSkillLevel.Int64),
			Description:       x.Description,
			ID:                int32(x.ID),
			Name:              x.Name,
			TrainedSkillLevel: int(x.TrainedSkillLevel.Int64),
		}
	}
	return oo, nil
}

func (r *Storage) ListCharacterSkillGroupsProgress(ctx context.Context, characterID int32) ([]model.ListCharacterSkillGroupProgress, error) {
	arg := queries.ListCharacterSkillGroupsProgressParams{
		MyCharacterID: int64(characterID),
		EveCategoryID: model.EveCategoryIDSkill,
	}
	xx, err := r.q.ListCharacterSkillGroupsProgress(ctx, arg)
	if err != nil {
		return nil, err
	}
	oo := make([]model.ListCharacterSkillGroupProgress, len(xx))
	for i, x := range xx {
		o := model.ListCharacterSkillGroupProgress{
			ID:    int32(x.EveGroupID),
			Name:  x.EveGroupName,
			Total: float64(x.Total),
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
	MyCharacterID      int32
	TrainedSkillLevel  int
}

func (r *Storage) UpdateOrCreateCharacterSkill(ctx context.Context, arg UpdateOrCreateCharacterSkillParams) error {
	arg2 := queries.UpdateOrCreateCharacterSkillParams{
		ActiveSkillLevel:   int64(arg.ActiveSkillLevel),
		EveTypeID:          int64(arg.EveTypeID),
		SkillPointsInSkill: int64(arg.SkillPointsInSkill),
		MyCharacterID:      int64(arg.MyCharacterID),
		TrainedSkillLevel:  int64(arg.TrainedSkillLevel),
	}
	if err := r.q.UpdateOrCreateCharacterSkill(ctx, arg2); err != nil {
		return fmt.Errorf("failed to update or create character skill for character %d: %w", arg.MyCharacterID, err)
	}
	return nil
}

func characterSkillFromDBModel(o queries.CharacterSkill, t queries.EveType, g queries.EveGroup, c queries.EveCategory) *model.CharacterSkill {
	if o.MyCharacterID == 0 {
		panic("missing character ID")
	}
	return &model.CharacterSkill{
		ActiveSkillLevel:   int(o.ActiveSkillLevel),
		EveType:            eveTypeFromDBModel(t, g, c),
		SkillPointsInSkill: int(o.SkillPointsInSkill),
		CharacterID:        int32(o.MyCharacterID),
		TrainedSkillLevel:  int(o.TrainedSkillLevel),
	}
}
