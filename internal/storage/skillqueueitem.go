package storage

import (
	"context"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage/queries"
)

func skillqueueItemFromDBModel(i queries.SkillqueueItem, skillName string, groupName string) *model.SkillqueueItem {
	i2 := &model.SkillqueueItem{
		GroupName:     groupName,
		FinishedLevel: int(i.FinishedLevel),
		MyCharacterID: int32(i.MyCharacterID),
		QueuePosition: int(i.QueuePosition),
		SkillName:     skillName,
	}
	if i.FinishDate.Valid {
		i2.FinishDate = i.FinishDate.Time
	}
	if i.LevelEndSp.Valid {
		i2.LevelEndSP = int(i.LevelEndSp.Int64)
	}
	if i.LevelStartSp.Valid {
		i2.LevelStartSP = int(i.LevelStartSp.Int64)
	}
	if i.StartDate.Valid {
		i2.StartDate = i.StartDate.Time
	}
	if i.TrainingStartSp.Valid {
		i2.TrainingStartSP = int(i.TrainingStartSp.Int64)
	}
	return i2
}

type SkillqueueItemParams struct {
	EveTypeID       int32
	FinishDate      time.Time
	FinishedLevel   int
	LevelEndSP      int
	LevelStartSP    int
	MyCharacterID   int32
	QueuePosition   int
	StartDate       time.Time
	TrainingStartSP int
}

func (r *Storage) CreateSkillqueueItem(ctx context.Context, arg SkillqueueItemParams) error {
	return createSkillqueueItem(ctx, r.q, arg)
}

func createSkillqueueItem(ctx context.Context, q *queries.Queries, arg SkillqueueItemParams) error {
	arg2 := queries.CreateSkillqueueItemParams{
		EveTypeID:     int64(arg.EveTypeID),
		FinishedLevel: int64(arg.FinishedLevel),
		MyCharacterID: int64(arg.MyCharacterID),
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
	err := q.CreateSkillqueueItem(ctx, arg2)
	return err
}

func (r *Storage) GetSkillqueueItem(ctx context.Context, characterID int32, pos int) (*model.SkillqueueItem, error) {
	arg := queries.GetSkillqueueItemParams{
		MyCharacterID: int64(characterID),
		QueuePosition: int64(pos),
	}
	row, err := r.q.GetSkillqueueItem(ctx, arg)
	if err != nil {
		return nil, err
	}
	return skillqueueItemFromDBModel(row.SkillqueueItem, row.SkillName, row.GroupName), err
}

func (r *Storage) ListSkillqueueItems(ctx context.Context, characterID int32) ([]*model.SkillqueueItem, error) {
	rows, err := r.q.ListSkillqueueItems(ctx, int64(characterID))
	if err != nil {
		return nil, err
	}
	ii2 := make([]*model.SkillqueueItem, len(rows))
	for i, row := range rows {
		ii2[i] = skillqueueItemFromDBModel(row.SkillqueueItem, row.SkillName, row.GroupName)
	}
	return ii2, nil
}

func (r *Storage) ReplaceSkillqueueItems(ctx context.Context, characterID int32, args []SkillqueueItemParams) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	qtx := r.q.WithTx(tx)
	if err := qtx.DeleteSkillqueueItems(ctx, int64(characterID)); err != nil {
		return err
	}
	for _, arg := range args {
		err := createSkillqueueItem(ctx, qtx, arg)
		if err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}