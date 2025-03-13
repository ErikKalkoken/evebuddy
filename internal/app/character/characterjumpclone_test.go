package character

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/stretchr/testify/assert"
)

func TestCharacterImplant(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := newCharacterService(st)
	ctx := context.Background()
	t.Run("should return time of next available jump with skill", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		now := time.Now().UTC()
		c := factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{
			LastCloneJumpAt: optional.New(now.Add(-6 * time.Hour)),
		})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeInfomorphSynchronizing})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      c.ID,
			EveTypeID:        app.EveTypeInfomorphSynchronizing,
			ActiveSkillLevel: 3,
		})
		x, err := cs.CharacterNextCloneJump(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.WithinDuration(t, now.Add(15*time.Hour), x, 10*time.Second)
		}
	})
	t.Run("should return time of next available jump without skill", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		now := time.Now().UTC()
		c := factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{
			LastCloneJumpAt: optional.New(now.Add(-6 * time.Hour)),
		})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeInfomorphSynchronizing})
		x, err := cs.CharacterNextCloneJump(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.WithinDuration(t, now.Add(18*time.Hour), x, 10*time.Second)
		}
	})
	t.Run("should return zero time when next jump available now", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		now := time.Now().UTC()
		c := factory.CreateCharacter(storage.UpdateOrCreateCharacterParams{
			LastCloneJumpAt: optional.New(now.Add(-20 * time.Hour)),
		})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeInfomorphSynchronizing})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      c.ID,
			EveTypeID:        app.EveTypeInfomorphSynchronizing,
			ActiveSkillLevel: 5,
		})
		x, err := cs.CharacterNextCloneJump(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, time.Time{}, x)
		}
	})
	t.Run("should return error when last jump not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeInfomorphSynchronizing})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      c.ID,
			EveTypeID:        app.EveTypeInfomorphSynchronizing,
			ActiveSkillLevel: 5,
		})
		_, err := cs.CharacterNextCloneJump(ctx, c.ID)
		assert.Error(t, err)
	})
}
