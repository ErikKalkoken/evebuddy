package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestSkillqueueItems(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new minimal", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		eveType := factory.CreateEveType()
		arg := storage.SkillqueueItemParams{
			CharacterID:   c.ID,
			EveTypeID:     eveType.ID,
			FinishedLevel: 5,
			QueuePosition: 4,
		}
		// when
		err := st.CreateCharacterSkillqueueItem(ctx, arg)
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterSkillqueueItem(ctx, c.ID, 4)
		require.NoError(t, err)
		xassert.Equal(t, arg.CharacterID, got.CharacterID)
		xassert.Equal(t, arg.EveTypeID, got.SkillID)
		xassert.Equal(t, arg.FinishedLevel, got.FinishedLevel)
		xassert.Equal(t, arg.QueuePosition, got.QueuePosition)
	})
	t.Run("can create new full", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		eveType := factory.CreateEveType()
		finishDate := time.Now().Add(10 * time.Hour)
		startDate := time.Now().Add(-4 * time.Hour)
		// when
		arg := storage.SkillqueueItemParams{
			CharacterID:     c.ID,
			EveTypeID:       eveType.ID,
			FinishDate:      optional.New(finishDate),
			FinishedLevel:   5,
			LevelEndSP:      optional.New[int64](10000),
			LevelStartSP:    optional.New[int64](100),
			QueuePosition:   4,
			StartDate:       optional.New(startDate),
			TrainingStartSP: optional.New[int64](50),
		}
		err := st.CreateCharacterSkillqueueItem(ctx, arg)
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterSkillqueueItem(ctx, c.ID, 4)
		require.NoError(t, err)
		xassert.Equal(t, arg.CharacterID, got.CharacterID)
		xassert.Equal(t, arg.EveTypeID, got.SkillID)
		xassert.Equal(t, arg.FinishedLevel, got.FinishedLevel)
		xassert.Equal(t, arg.LevelEndSP, got.LevelEndSP)
		xassert.Equal(t, arg.QueuePosition, got.QueuePosition)
		xassert.Equal(t, arg.TrainingStartSP, got.TrainingStartSP)
		xassert.EqualOptional(t, arg.FinishDate.ValueOrZero(), got.FinishDate)
		xassert.EqualOptional(t, arg.StartDate.ValueOrZero(), got.StartDate)
	})
	t.Run("can list items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		// when
		ii, err := st.ListCharacterSkillqueueItems(ctx, c.ID)
		// then
		require.NoError(t, err)
		assert.Len(t, ii, 3)
	})
	t.Run("can replace items", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		eveType := factory.CreateEveType()
		arg := storage.SkillqueueItemParams{
			EveTypeID:     eveType.ID,
			FinishedLevel: 5,
			CharacterID:   c.ID,
			QueuePosition: 0,
			StartDate:     optional.New(time.Now().Add(1 * time.Hour)),
			FinishDate:    optional.New(time.Now().Add(3 * time.Hour)),
		}
		// when
		err := st.ReplaceCharacterSkillqueueItems(ctx, c.ID, []storage.SkillqueueItemParams{arg})
		// then
		require.NoError(t, err)
		ii, err := st.ListCharacterSkillqueueItems(ctx, c.ID)
		require.NoError(t, err)
		assert.Len(t, ii, 1)
	})
	t.Run("should keep existing items unchanged when replacing fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		x1 := factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		eveType := factory.CreateEveType()
		// when
		err := st.ReplaceCharacterSkillqueueItems(ctx, c.ID, []storage.SkillqueueItemParams{
			{
				EveTypeID:     eveType.ID,
				FinishedLevel: 5,
				CharacterID:   c.ID,
				QueuePosition: 0,
			},
			{
				EveTypeID:     eveType.ID,
				CharacterID:   c.ID,
				QueuePosition: 1,
			}, // invalid
		})
		// then
		require.ErrorIs(t, err, app.ErrInvalid)
		ii, err := st.ListCharacterSkillqueueItems(ctx, c.ID)
		require.NoError(t, err)
		got := xslices.Map(ii, func(x *app.CharacterSkillqueueItem) int64 {
			return x.ID
		})
		xassert.Equal(t, []int64{x1.ID}, got)
	})
	t.Run("should return invalid error when replacing with item of other character", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacter()
		c2 := factory.CreateCharacter()
		x1 := factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c1.ID})
		eveType := factory.CreateEveType()
		// when
		err := st.ReplaceCharacterSkillqueueItems(ctx, c1.ID, []storage.SkillqueueItemParams{
			{
				EveTypeID:     eveType.ID,
				FinishedLevel: 5,
				CharacterID:   c2.ID,
				QueuePosition: 0,
			},
		})
		// then
		require.ErrorIs(t, err, app.ErrInvalid)
		ii1, err := st.ListCharacterSkillqueueItems(ctx, c1.ID)
		require.NoError(t, err)
		got := xslices.Map(ii1, func(x *app.CharacterSkillqueueItem) int64 {
			return x.ID
		})
		xassert.Equal(t, []int64{x1.ID}, got)
		ii2, err := st.ListCharacterSkillqueueItems(ctx, c2.ID)
		require.NoError(t, err)
		assert.Empty(t, ii2)
	})
	t.Run("should return invalid error when replacing items for character ID 0", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		err := st.ReplaceCharacterSkillqueueItems(ctx, 0, nil)
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}

func TestSkillqueueItems_GetCharacterTotalTrainingTime(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can calculate total training time", func(t *testing.T) {
		// given
		now := time.Now()
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   optional.New(now.Add(1 * time.Hour)),
			FinishDate:  optional.New(now.Add(3 * time.Hour)),
		})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   optional.New(now.Add(3 * time.Hour)),
			FinishDate:  optional.New(now.Add(4 * time.Hour)),
		})
		// when
		v, err := st.GetCharacterTotalTrainingTime(ctx, c.ID)
		// then
		require.NoError(t, err)
		xassert.EqualDuration(t, 3*time.Hour, v, time.Second)
	})
	t.Run("can calculate total training time when times are not in UTC", func(t *testing.T) {
		// given
		now := time.Now().In(time.FixedZone("UTC-8", -8*3600))
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   optional.New(now.Add(1 * time.Hour)),
			FinishDate:  optional.New(now.Add(3 * time.Hour)),
		})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   optional.New(now.Add(3 * time.Hour)),
			FinishDate:  optional.New(now.Add(4 * time.Hour)),
		})
		// when
		v, err := st.GetCharacterTotalTrainingTime(ctx, c.ID)
		// then
		require.NoError(t, err)
		xassert.EqualDuration(t, 3*time.Hour, v, time.Second)
	})
	t.Run("should count only remaining time of skill in training", func(t *testing.T) {
		// given
		now := time.Now().UTC()
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   optional.New(now.Add(-1 * time.Hour)),
			FinishDate:  optional.New(now.Add(2 * time.Hour)),
		})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   optional.New(now.Add(2 * time.Hour)),
			FinishDate:  optional.New(now.Add(3 * time.Hour)),
		})
		// when
		v, err := st.GetCharacterTotalTrainingTime(ctx, c.ID)
		// then
		require.NoError(t, err)
		xassert.EqualDuration(t, 3*time.Hour, v, time.Second)
	})
	t.Run("should report when training is not active", func(t *testing.T) {
		// given
		now := time.Now()
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: c.ID,
			StartDate:   optional.New(now.Add(-3 * time.Hour)),
			FinishDate:  optional.New(now.Add(-1 * time.Hour)),
		})
		// when
		v, err := st.GetCharacterTotalTrainingTime(ctx, c.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, 0, v)
	})
}
