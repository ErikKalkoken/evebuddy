package characterservice_test

import (
	"context"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	"github.com/stretchr/testify/assert"
)

func TestGetCharacter(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := newCharacterService(st)
	ctx := context.Background()
	t.Run("should return own error when object not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := cs.GetCharacter(ctx, 42)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return obj when found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x1 := factory.CreateCharacter()
		// when
		x2, err := cs.GetCharacter(ctx, x1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
}

func TestGetAnyCharacter(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := newCharacterService(st)
	ctx := context.Background()
	t.Run("should return own error when object not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := cs.GetAnyCharacter(ctx)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return obj when found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x1 := factory.CreateCharacter()
		// when
		x2, err := cs.GetAnyCharacter(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
}

func TestTrainingWatchers(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := newCharacterService(st)
	ctx := context.Background()
	t.Run("should enable watchers for characters with active queues only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c1.ID})
		c2 := factory.CreateCharacter()
		// when
		err := cs.EnableAllTrainingWatchers(ctx)
		// then
		if assert.NoError(t, err) {
			c1x, err := cs.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.True(t, c1x.IsTrainingWatched)
			}
			c2x, err := cs.GetCharacter(ctx, c2.ID)
			if assert.NoError(t, err) {
				assert.False(t, c2x.IsTrainingWatched)
			}
		}
	})
	t.Run("should disable all training watchers", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{IsTrainingWatched: true})
		c2 := factory.CreateCharacter()
		// when
		err := cs.DisableAllTrainingWatchers(ctx)
		// then
		if assert.NoError(t, err) {
			c1x, err := cs.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.False(t, c1x.IsTrainingWatched)
			}
			c2x, err := cs.GetCharacter(ctx, c2.ID)
			if assert.NoError(t, err) {
				assert.False(t, c2x.IsTrainingWatched)
			}
		}
	})
	t.Run("should enable watchers for character with active queues", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c1.ID})
		// when
		err := cs.EnableTrainingWatcher(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			c1a, err := cs.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.True(t, c1a.IsTrainingWatched)
			}
		}
	})
	t.Run("should not enable watchers for character without active queues", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		// when
		err := cs.EnableTrainingWatcher(ctx, c1.ID)
		// then
		if assert.NoError(t, err) {
			c1a, err := cs.GetCharacter(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.False(t, c1a.IsTrainingWatched)
			}
		}
	})
}

func newCharacterService(st *storage.Storage) *characterservice.CharacterService {
	scs := statuscacheservice.New(memcache.New())
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		StatusCacheService: scs,
		Storage:            st,
	})
	s := characterservice.New(characterservice.Params{
		EveUniverseService: eus,
		StatusCacheService: scs,
		Storage:            st,
	})
	return s
}
