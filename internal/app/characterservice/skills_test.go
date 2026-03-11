package characterservice_test

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

func TestIsTrainingActive(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	cs := testdouble.NewCharacterService(characterservice.Params{Storage: st})
	ctx := context.Background()
	t.Run("should return true when training is active", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		now := time.Now().UTC()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{
			CharacterID: character.ID,
			StartDate:   optional.New(now.Add(-1 * time.Hour)),
			FinishDate:  optional.New(now.Add(3 * time.Hour)),
		})
		// when
		got, err := cs.IsTrainingActive(ctx, character.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, true, got)
	})
	t.Run("should return false when training is inactive", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		// when
		got, err := cs.IsTrainingActive(ctx, character.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, false, got)
	})
}

func TestUpdateTickerNotifyExpiredTraining(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	ctx := context.Background()
	t.Run("send notification when watched & expired", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		cs := testdouble.NewCharacterService(characterservice.Params{Storage: st})
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
		var sendCount int
		// when
		err := cs.NotifyExpiredTrainingForWatched(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		// then
		require.NoError(t, err)
		xassert.Equal(t, sendCount, 1)
	})
	t.Run("do nothing when not watched", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		cs := testdouble.NewCharacterService(characterservice.Params{Storage: st})
		c := factory.CreateCharacterFull()
		var sendCount int
		// when
		err := cs.NotifyExpiredTrainingForWatched(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		// then
		require.NoError(t, err)
		xassert.Equal(t, sendCount, 0)
	})
	t.Run("don't send notification when watched and training ongoing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		cs := testdouble.NewCharacterService(characterservice.Params{Storage: st})
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		var sendCount int
		// when
		err := cs.NotifyExpiredTrainingForWatched(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		// then
		require.NoError(t, err)
		xassert.Equal(t, sendCount, 0)
	})
	t.Run("should only send one notification", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		cs := testdouble.NewCharacterService(characterservice.Params{Storage: st})
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
		var sendCount int
		err := cs.NotifyExpiredTrainingForWatched(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		require.NoError(t, err)
		// when
		err = cs.NotifyExpiredTrainingForWatched(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		// then
		require.NoError(t, err)
		xassert.Equal(t, sendCount, 1)
	})
}

func TestCharacterService_ListSkills(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := testdouble.NewCharacterService(characterservice.Params{Storage: st})
	t.Run("should return character skill for all existing eve skill", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategorySkill})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID, IsPublished: true})
		es1 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID,
			TypeID:      es1.ID,
		})
		es2 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		// when
		oo, err := cs.ListSkills(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		want := set.Of(es1.ID, es2.ID)
		got := set.Collect(xiter.MapSlice(oo, func(x *app.CharacterSkill2) int64 {
			return x.Skill.Type.ID
		}))
		xassert.Equal(t, want, got)
	})

	t.Run("should report true when rerequisites are met", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategorySkill})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID, IsPublished: true})
		es1 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		es2 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		primarySkillType := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributePrimarySkillID,
		})
		primarySkillLevel := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributePrimarySkillLevel,
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        es1.ID,
			DogmaAttributeID: primarySkillType.ID,
			Value:            float64(es2.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        es1.ID,
			DogmaAttributeID: primarySkillLevel.ID,
			Value:            float64(2),
		})

		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID,
			TypeID:      es1.ID,
		})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      c.ID,
			TypeID:           es2.ID,
			ActiveSkillLevel: 3,
		})
		// when
		oo, err := cs.ListSkills(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		m := maps.Collect(xiter.MapSlice2(oo, func(x *app.CharacterSkill2) (int64, *app.CharacterSkill2) {
			return x.Skill.Type.ID, x
		}))
		want := set.Of(es1.ID, es2.ID)
		got := set.Collect(maps.Keys(m))
		xassert.Equal(t, want, got)

		o1 := m[es1.ID]
		assert.True(t, o1.HasPrerequisites)
	})
	t.Run("should report false when rerequisites are not met", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		category := factory.CreateEveCategory(storage.CreateEveCategoryParams{ID: app.EveCategorySkill})
		group := factory.CreateEveGroup(storage.CreateEveGroupParams{CategoryID: category.ID, IsPublished: true})
		es1 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		es2 := factory.CreateEveType(storage.CreateEveTypeParams{GroupID: group.ID, IsPublished: true})
		primarySkillType := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributePrimarySkillID,
		})
		primarySkillLevel := factory.CreateEveDogmaAttribute(storage.CreateEveDogmaAttributeParams{
			ID: app.EveDogmaAttributePrimarySkillLevel,
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        es1.ID,
			DogmaAttributeID: primarySkillType.ID,
			Value:            float64(es2.ID),
		})
		factory.CreateEveTypeDogmaAttribute(storage.CreateEveTypeDogmaAttributeParams{
			EveTypeID:        es1.ID,
			DogmaAttributeID: primarySkillLevel.ID,
			Value:            float64(2),
		})

		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID,
			TypeID:      es1.ID,
		})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      c.ID,
			TypeID:           es2.ID,
			ActiveSkillLevel: 1,
		})
		// when
		oo, err := cs.ListSkills(t.Context(), c.ID)
		// then
		require.NoError(t, err)
		m := maps.Collect(xiter.MapSlice2(oo, func(x *app.CharacterSkill2) (int64, *app.CharacterSkill2) {
			return x.Skill.Type.ID, x
		}))
		want := set.Of(es1.ID, es2.ID)
		got := set.Collect(maps.Keys(m))
		xassert.Equal(t, want, got)

		o1 := m[es1.ID]
		assert.False(t, o1.HasPrerequisites)
	})
}
