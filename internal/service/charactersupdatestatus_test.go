package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestMyCharacterUpdateStatus(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := service.NewService(r)
	ctx := context.Background()
	t.Run("Can set updated for section", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		err := s.CharacterSectionSetUpdated(c.ID, model.CharacterSectionSkillqueue)
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCharacterUpdateStatus(ctx, c.ID, model.CharacterSectionSkillqueue)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.UpdatedAt, 30*time.Second)
			}
		}
	})
	t.Run("Can retrieve updated at for section", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		updateAt := time.Now().Add(3 * time.Hour)
		t1 := factory.CreateCharacterUpdateStatus(storage.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionSkillqueue,
			UpdatedAt:   updateAt,
		})
		// when
		t2, err := s.CharacterSectionUpdatedAt(c.ID, model.CharacterSectionSkillqueue)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, t1.UpdatedAt.Unix(), t2.Unix())
		}
	})
	t.Run("Can report when updated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		updateAt := time.Now().Add(3 * time.Hour)
		factory.CreateCharacterUpdateStatus(storage.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionSkillqueue,
			UpdatedAt:   updateAt,
		})
		// when
		x, err := s.CharacterSectionWasUpdated(c.ID, model.CharacterSectionSkillqueue)
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}
	})
	t.Run("Can report when not yet updated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		x, err := s.CharacterSectionWasUpdated(c.ID, model.CharacterSectionSkillqueue)
		// then
		if assert.NoError(t, err) {
			assert.False(t, x)
		}
	})
	t.Run("Can report when section update is expired", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		updateAt := time.Now().Add(-3 * time.Hour)
		factory.CreateCharacterUpdateStatus(storage.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionSkillqueue,
			UpdatedAt:   updateAt,
		})
		// when
		x, err := s.CharacterSectionIsUpdateExpired(c.ID, model.CharacterSectionSkillqueue)
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}
	})
}