package storage

import (
	"context"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/types"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

type SkillqueueItemParams struct {
	EveTypeID       int32
	FinishDate      time.Time
	FinishedLevel   int
	LevelEndSP      int
	LevelStartSP    int
	CharacterID     int32
	QueuePosition   int
	StartDate       time.Time
	TrainingStartSP int
}

func (r *Storage) GetTotalTrainingTime(ctx context.Context, characterID int32) (types.NullDuration, error) {
	var d types.NullDuration
	x, err := r.q.GetTotalTrainingTime(ctx, int64(characterID))
	if err != nil {
		return d, err
	}
	if !x.Valid {
		return d, nil
	}
	d.Duration = time.Duration(float64(time.Hour) * 24 * x.Float64)
	d.Valid = true
	return d, nil
}

func (r *Storage) CreateSkillqueueItem(ctx context.Context, arg SkillqueueItemParams) error {
	return createCharacterSkillqueueItem(ctx, r.q, arg)
}

func createCharacterSkillqueueItem(ctx context.Context, q *queries.Queries, arg SkillqueueItemParams) error {
	arg2 := queries.CreateCharacterSkillqueueItemParams{
		EveTypeID:     int64(arg.EveTypeID),
		FinishedLevel: int64(arg.FinishedLevel),
		CharacterID:   int64(arg.CharacterID),
		QueuePosition: int64(arg.QueuePosition),
	}
	if !arg.FinishDate.IsZero() {
		arg2.FinishDate.Time = arg.FinishDate
		arg2.FinishDate.Valid = true
	}
	if arg.LevelEndSP != 0 {
		arg2.LevelEndSp.Int64 = int64(arg.LevelEndSP)
		arg2.LevelEndSp.Valid = true
	}
	if arg.LevelStartSP != 0 {
		arg2.LevelStartSp.Int64 = int64(arg.LevelStartSP)
		arg2.LevelStartSp.Valid = true
	}
	if !arg.StartDate.IsZero() {
		arg2.StartDate.Time = arg.StartDate
		arg2.StartDate.Valid = true
	}
	if arg.TrainingStartSP != 0 {
		arg2.TrainingStartSp.Int64 = int64(arg.TrainingStartSP)
		arg2.TrainingStartSp.Valid = true
	}
	err := q.CreateCharacterSkillqueueItem(ctx, arg2)
	return err
}

func (r *Storage) GetSkillqueueItem(ctx context.Context, characterID int32, pos int) (*model.CharacterSkillqueueItem, error) {
	arg := queries.GetCharacterSkillqueueItemParams{
		CharacterID:   int64(characterID),
		QueuePosition: int64(pos),
	}
	row, err := r.q.GetCharacterSkillqueueItem(ctx, arg)
	if err != nil {
		return nil, err
	}
	return skillqueueItemFromDBModel(row.CharacterSkillqueueItem, row.SkillName, row.GroupName, row.SkillDescription), err
}

func (r *Storage) ListSkillqueueItems(ctx context.Context, characterID int32) ([]*model.CharacterSkillqueueItem, error) {
	rows, err := r.q.ListCharacterSkillqueueItems(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	ii2 := make([]*model.CharacterSkillqueueItem, len(rows))
	for i, row := range rows {
		ii2[i] = skillqueueItemFromDBModel(row.CharacterSkillqueueItem, row.SkillName, row.GroupName, row.SkillDescription)
	}
	return ii2, nil
}

func (r *Storage) ReplaceCharacterSkillqueueItems(ctx context.Context, characterID int32, args []SkillqueueItemParams) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := r.q.WithTx(tx)
	if err := qtx.DeleteCharacterSkillqueueItems(ctx, int64(characterID)); err != nil {
		return err
	}
	for _, arg := range args {
		err := createCharacterSkillqueueItem(ctx, qtx, arg)
		if err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func skillqueueItemFromDBModel(o queries.CharacterSkillqueueItem, skillName, groupName, description string) *model.CharacterSkillqueueItem {
	i2 := &model.CharacterSkillqueueItem{
		CharacterID:      int32(o.CharacterID),
		GroupName:        groupName,
		FinishedLevel:    int(o.FinishedLevel),
		ID:               o.ID,
		QueuePosition:    int(o.QueuePosition),
		SkillName:        skillName,
		SkillDescription: description,
	}
	if o.FinishDate.Valid {
		i2.FinishDate = o.FinishDate.Time
	}
	if o.LevelEndSp.Valid {
		i2.LevelEndSP = int(o.LevelEndSp.Int64)
	}
	if o.LevelStartSp.Valid {
		i2.LevelStartSP = int(o.LevelStartSp.Int64)
	}
	if o.StartDate.Valid {
		i2.StartDate = o.StartDate.Time
	}
	if o.TrainingStartSp.Valid {
		i2.TrainingStartSP = int(o.TrainingStartSp.Int64)
	}
	return i2
}