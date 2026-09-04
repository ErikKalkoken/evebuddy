package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/queries"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type SkillqueueItemParams struct {
	EveTypeID       int64
	FinishDate      optional.Optional[time.Time]
	FinishedLevel   int64
	LevelEndSP      optional.Optional[int64]
	LevelStartSP    optional.Optional[int64]
	CharacterID     int64
	QueuePosition   int64
	StartDate       optional.Optional[time.Time]
	TrainingStartSP optional.Optional[int64]
}

// GetCharacterTotalTrainingTime returns the total training time for a character
// and reports whether the training is active.
func (st *Storage) GetCharacterTotalTrainingTime(ctx context.Context, characterID int64) (time.Duration, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterTotalTrainingTime: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return 0, wrapErr(app.ErrInvalid)
	}
	x, err := st.qRO.GetTotalTrainingTime(ctx, characterID)
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
		CharacterID:     arg.CharacterID,
		EveTypeID:       arg.EveTypeID,
		FinishDate:      optional.ToNullTime(arg.FinishDate),
		FinishedLevel:   arg.FinishedLevel,
		LevelEndSp:      optional.ToNullInt64(arg.LevelEndSP),
		LevelStartSp:    optional.ToNullInt64(arg.LevelStartSP),
		QueuePosition:   arg.QueuePosition,
		StartDate:       optional.ToNullTime(arg.StartDate),
		TrainingStartSp: optional.ToNullInt64(arg.TrainingStartSP),
	})
	if err != nil {
		return wrapErr(err)
	}
	return err
}

func (st *Storage) GetCharacterSkillqueueItem(ctx context.Context, characterID int64, pos int64) (*app.CharacterSkillqueueItem, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("GetCharacterSkillqueueItem: %d %d: %w", characterID, pos, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	arg := queries.GetCharacterSkillqueueItemParams{
		CharacterID:   characterID,
		QueuePosition: pos,
	}
	r, err := st.qRO.GetCharacterSkillqueueItem(ctx, arg)
	if err != nil {
		return nil, wrapErr(convertGetError(err))
	}
	o := skillqueueItemFromDBModel(
		description(r.SkillDescription),
		groupName(r.GroupName),
		r.CharacterSkillqueueItem,
		skillName(r.SkillName),
	)
	return o, nil
}

// ListCharacterSkillqueueItems returns the skillqueue for a character. Items are ordered by queue position.
func (st *Storage) ListCharacterSkillqueueItems(ctx context.Context, characterID int64) ([]*app.CharacterSkillqueueItem, error) {
	wrapErr := func(err error) error {
		return fmt.Errorf("ListCharacterSkillqueueItems: %d: %w", characterID, err)
	}
	if characterID == 0 {
		return nil, wrapErr(app.ErrInvalid)
	}
	rows, err := st.qRO.ListCharacterSkillqueueItems(ctx, characterID)
	if err != nil {
		return nil, wrapErr(err)
	}
	var oo []*app.CharacterSkillqueueItem
	for _, r := range rows {
		oo = append(oo, skillqueueItemFromDBModel(
			description(r.SkillDescription),
			groupName(r.GroupName),
			r.CharacterSkillqueueItem,
			skillName(r.SkillName),
		))
	}
	return oo, nil
}

func (st *Storage) ReplaceCharacterSkillqueueItems(ctx context.Context, characterID int64, items []SkillqueueItemParams) error {
	wrapErr := func(err error) error {
		return fmt.Errorf("ReplaceCharacterSkillqueueItems for %d: %+v: %w", characterID, items, err)
	}
	tx, err := st.dbRW.Begin()
	if err != nil {
		return wrapErr(err)
	}
	defer tx.Rollback()
	qtx := st.qRW.WithTx(tx)
	if err := qtx.DeleteCharacterSkillqueueItems(ctx, characterID); err != nil {
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

func skillqueueItemFromDBModel(
	description description,
	groupName groupName,
	it queries.CharacterSkillqueueItem,
	skillName skillName,

) *app.CharacterSkillqueueItem {
	o := &app.CharacterSkillqueueItem{
		CharacterID:      it.CharacterID,
		FinishDate:       optional.FromNullTime(it.FinishDate),
		FinishedLevel:    it.FinishedLevel,
		GroupName:        string(groupName),
		ID:               it.ID,
		LevelEndSP:       optional.FromNullInt64(it.LevelEndSp),
		LevelStartSP:     optional.FromNullInt64(it.LevelStartSp),
		QueuePosition:    it.QueuePosition,
		SkillDescription: string(description),
		SkillID:          it.EveTypeID,
		SkillName:        string(skillName),
		StartDate:        optional.FromNullTime(it.StartDate),
		TrainingStartSP:  optional.FromNullInt64(it.TrainingStartSp),
	}
	return o
}
