package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
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

// GetCharacterTotalTrainingTime returns the total training time for a character
// and reports whether the training is active.
func (st *Storage) GetCharacterTotalTrainingTime(ctx context.Context, characterID int32) (time.Duration, error) {
	x, err := st.qRO.GetTotalTrainingTime(ctx, int64(characterID))
	if err != nil {
		return 0, fmt.Errorf("fetching total training time for character %d: %w", characterID, convertGetError(err))
	}
	if !x.Valid {
		return 0, nil
	}
	d := time.Duration(float64(time.Hour) * 24 * x.Float64)
	return d, nil
}

func (st *Storage) CreateCharacterSkillqueueItem(ctx context.Context, arg SkillqueueItemParams) error {
	return createCharacterSkillqueueItem(ctx, st.qRW, arg)
}

func createCharacterSkillqueueItem(ctx context.Context, q *queries.Queries, arg SkillqueueItemParams) error {
	if arg.CharacterID == 0 || arg.EveTypeID == 0 {
		return fmt.Errorf("createCharacterSkillqueueItem: %+v: %w", arg, app.ErrInvalid)
	}
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
	if err != nil {
		return fmt.Errorf("create skill queue item for character %d: %w", arg.CharacterID, err)
	}
	return err
}

func (st *Storage) GetCharacterSkillqueueItem(ctx context.Context, characterID int32, pos int) (*app.CharacterSkillqueueItem, error) {
	if characterID == 0 {
		return nil, fmt.Errorf("GetCharacterSkillqueueItem: %w", app.ErrInvalid)
	}
	arg := queries.GetCharacterSkillqueueItemParams{
		CharacterID:   int64(characterID),
		QueuePosition: int64(pos),
	}
	r, err := st.qRO.GetCharacterSkillqueueItem(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("get skill queue item for character %d: %w", characterID, convertGetError(err))
	}
	return skillqueueItemFromDBModel(skillqueueItemFromDBModelParams{
		description: r.SkillDescription,
		groupName:   r.GroupName,
		o:           r.CharacterSkillqueueItem,
		skillName:   r.SkillName,
	}), err
}

// ListCharacterSkillqueueItems returns the skillqueue for a character. Items are ordered by queue position.
func (st *Storage) ListCharacterSkillqueueItems(ctx context.Context, characterID int32) ([]*app.CharacterSkillqueueItem, error) {
	if characterID == 0 {
		return nil, fmt.Errorf("ListCharacterSkillqueueItems: %w", app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterSkillqueueItems(ctx, int64(characterID))
	if err != nil {
		return nil, fmt.Errorf("list skill queue items for character %d: %w", characterID, err)
	}
	ii2 := make([]*app.CharacterSkillqueueItem, len(rows))
	for i, r := range rows {
		ii2[i] = skillqueueItemFromDBModel(skillqueueItemFromDBModelParams{
			description: r.SkillDescription,
			groupName:   r.GroupName,
			o:           r.CharacterSkillqueueItem,
			skillName:   r.SkillName,
		})
	}
	return ii2, nil
}

func (st *Storage) ReplaceCharacterSkillqueueItems(ctx context.Context, characterID int32, args []SkillqueueItemParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("replaceCharacterSkillqueueItems for ID %d: %+v: %w", characterID, args, err)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	if err := qtx.DeleteCharacterSkillqueueItems(ctx, int64(characterID)); err != nil {
		return wrapErr(err)
	}
	for _, arg := range args {
		err := createCharacterSkillqueueItem(ctx, qtx, arg)
		if err != nil {
			return wrapErr(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return wrapErr(err)
	}
	return nil
}

type skillqueueItemFromDBModelParams struct {
	description string
	groupName   string
	o           queries.CharacterSkillqueueItem
	skillName   string
}

func skillqueueItemFromDBModel(arg skillqueueItemFromDBModelParams) *app.CharacterSkillqueueItem {
	i2 := &app.CharacterSkillqueueItem{
		CharacterID:      int32(arg.o.CharacterID),
		GroupName:        arg.groupName,
		FinishedLevel:    int(arg.o.FinishedLevel),
		ID:               arg.o.ID,
		QueuePosition:    int(arg.o.QueuePosition),
		SkillID:          int32(arg.o.EveTypeID),
		SkillName:        arg.skillName,
		SkillDescription: arg.description,
	}
	if arg.o.FinishDate.Valid {
		i2.FinishDate = arg.o.FinishDate.Time
	}
	if arg.o.LevelEndSp.Valid {
		i2.LevelEndSP = int(arg.o.LevelEndSp.Int64)
	}
	if arg.o.LevelStartSp.Valid {
		i2.LevelStartSP = int(arg.o.LevelStartSp.Int64)
	}
	if arg.o.StartDate.Valid {
		i2.StartDate = arg.o.StartDate.Time
	}
	if arg.o.TrainingStartSp.Valid {
		i2.TrainingStartSP = int(arg.o.TrainingStartSp.Int64)
	}
	return i2
}
