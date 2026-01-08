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
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterTotalTrainingTime: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	x, err := st.qRO.GetTotalTrainingTime(ctx, int64(characterID))
	if err != nil {
		return 0, wrapErr(convertGetError(err))
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
	wrapErr := func(err error) error {
		return fmt.Errorf("createCharacterSkillqueueItem: %+v: %w", arg, err)
	}
	if arg.CharacterID == 0 || arg.EveTypeID == 0 || arg.FinishedLevel == 0 {
		return wrapErr(app.ErrInvalid)
	}
	err := q.CreateCharacterSkillqueueItem(ctx, queries.CreateCharacterSkillqueueItemParams{
		CharacterID:     int64(arg.CharacterID),
		EveTypeID:       int64(arg.EveTypeID),
		FinishDate:      NewNullTimeFromTime(arg.FinishDate),
		FinishedLevel:   int64(arg.FinishedLevel),
		LevelEndSp:      NewNullInt64(arg.LevelEndSP),
		LevelStartSp:    NewNullInt64(arg.LevelStartSP),
		QueuePosition:   int64(arg.QueuePosition),
		StartDate:       NewNullTimeFromTime(arg.StartDate),
		TrainingStartSp: NewNullInt64(arg.TrainingStartSP),
	})
	if err != nil {
		return wrapErr(err)
	}
	return err
}

func (st *Storage) GetCharacterSkillqueueItem(ctx context.Context, characterID int32, pos int) (*app.CharacterSkillqueueItem, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterSkillqueueItem: %d %d: %w", characterID, pos, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	arg := queries.GetCharacterSkillqueueItemParams{
		CharacterID:   int64(characterID),
		QueuePosition: int64(pos),
	}
	r, err := st.qRO.GetCharacterSkillqueueItem(ctx, arg)
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	o := skillqueueItemFromDBModel(skillqueueItemFromDBModelParams{
		description: r.SkillDescription,
		groupName:   r.GroupName,
		o:           r.CharacterSkillqueueItem,
		skillName:   r.SkillName,
	})
	return o, nil
}

// ListCharacterSkillqueueItems returns the skillqueue for a character. Items are ordered by queue position.
func (st *Storage) ListCharacterSkillqueueItems(ctx context.Context, characterID int32) ([]*app.CharacterSkillqueueItem, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterSkillqueueItems: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterSkillqueueItems(ctx, int64(characterID))
	if err != nil {
		return nil, wrapErr(err)
	}
	oo := make([]*app.CharacterSkillqueueItem, 0)
	for _, r := range rows {
		oo = append(oo, skillqueueItemFromDBModel(skillqueueItemFromDBModelParams{
			description: r.SkillDescription,
			groupName:   r.GroupName,
			o:           r.CharacterSkillqueueItem,
			skillName:   r.SkillName,
		}))
	}
	return oo, nil
}

func (st *Storage) ReplaceCharacterSkillqueueItems(ctx context.Context, characterID int32, items []SkillqueueItemParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceCharacterSkillqueueItems for %d: %+v: %w", characterID, items, err)
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
	for _, it := range items {
		err := createCharacterSkillqueueItem(ctx, qtx, it)
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
		FinishDate:       NewTimeFromNullTime(arg.o.FinishDate),
		FinishedLevel:    int(arg.o.FinishedLevel),
		GroupName:        arg.groupName,
		ID:               arg.o.ID,
		LevelEndSP:       NewIntegerFromNullInt64[int](arg.o.LevelEndSp),
		LevelStartSP:     NewIntegerFromNullInt64[int](arg.o.LevelStartSp),
		QueuePosition:    int(arg.o.QueuePosition),
		SkillDescription: arg.description,
		SkillID:          int32(arg.o.EveTypeID),
		SkillName:        arg.skillName,
		StartDate:        NewTimeFromNullTime(arg.o.StartDate),
		TrainingStartSP:  NewIntegerFromNullInt64[int](arg.o.TrainingStartSp),
	}
	return i2
}
