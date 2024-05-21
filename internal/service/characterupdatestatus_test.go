package service_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestCharacterUpdateStatus(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := service.NewService(r)
	t.Run("Can retrieve updated at for section", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		updateAt := sql.NullTime{Time: time.Now().Add(3 * time.Hour), Valid: true}
		o := factory.CreateCharacterUpdateStatus(storage.CharacterUpdateStatusParams{
			CharacterID:   c.ID,
			Section:       model.CharacterSectionSkillqueue,
			LastUpdatedAt: updateAt,
		})
		// when
		x, err := s.CharacterSectionUpdatedAt(c.ID, model.CharacterSectionSkillqueue)
		// then
		if assert.NoError(t, err) {

			assert.Equal(t, o.LastUpdatedAt.Time.UTC(), x.Time.UTC())
		}
	})
	t.Run("Can report when updated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		updateAt := sql.NullTime{Time: time.Now().Add(3 * time.Hour), Valid: true}
		factory.CreateCharacterUpdateStatus(storage.CharacterUpdateStatusParams{
			CharacterID:   c.ID,
			Section:       model.CharacterSectionSkillqueue,
			LastUpdatedAt: updateAt,
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
		updateAt := sql.NullTime{Time: time.Now().Add(-3 * time.Hour), Valid: true}
		factory.CreateCharacterUpdateStatus(storage.CharacterUpdateStatusParams{
			CharacterID:   c.ID,
			Section:       model.CharacterSectionSkillqueue,
			LastUpdatedAt: updateAt,
		})
		// when
		x, err := s.CharacterSectionIsUpdateExpired(c.ID, model.CharacterSectionSkillqueue)
		// then
		if assert.NoError(t, err) {
			assert.True(t, x)
		}
	})
}
