package character

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestUpdateCharacterSkillsESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := New(r, nil, nil, nil, nil, nil)
	ctx := context.Background()
	t.Run("should update skills from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 41})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 42})
		data := `{
			"skills": [
			  {
				"active_skill_level": 3,
				"skill_id": 41,
				"skillpoints_in_skill": 10000,
				"trained_skill_level": 4
			  },
			  {
				"active_skill_level": 1,
				"skill_id": 42,
				"skillpoints_in_skill": 20000,
				"trained_skill_level": 2
			  }
			],
			"total_sp": 90000
		  }`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/skills/", c.ID),
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		changed, err := s.updateCharacterSkillsESI(ctx, UpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionSkills,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			c2, err := r.GetCharacter(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.True(t, c2.TotalSP.Valid)
				assert.Equal(t, int64(90000), c2.TotalSP.Int64)
			}
			o1, err := r.GetCharacterSkill(ctx, c.ID, 41)
			if assert.NoError(t, err) {
				assert.Equal(t, 3, o1.ActiveSkillLevel)
				assert.Equal(t, 10000, o1.SkillPointsInSkill)
				assert.Equal(t, 4, o1.TrainedSkillLevel)
			}
			o2, err := r.GetCharacterSkill(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, 1, o2.ActiveSkillLevel)
				assert.Equal(t, 20000, o2.SkillPointsInSkill)
				assert.Equal(t, 2, o2.TrainedSkillLevel)
			}
		}
	})
	t.Run("should delete skills not returned from ESI", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		// old := factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 41})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 42})
		data := `{
			"skills": [
			  {
				"active_skill_level": 3,
				"skill_id": 41,
				"skillpoints_in_skill": 10000,
				"trained_skill_level": 4
			  }
			],
			"total_sp": 90000
		  }`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/skills/", c.ID),
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		changed, err := s.updateCharacterSkillsESI(ctx, UpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionSkills,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			_, err := r.GetCharacterSkill(ctx, c.ID, 42)
			assert.Error(t, err, storage.ErrNotFound)
			_, err = r.GetCharacterSkill(ctx, c.ID, 41)
			assert.NoError(t, err)
		}
	})

}

// func TestListWalletJournalEntries(t *testing.T) {
// 	db, r, factory := testutil.New()
// 	defer db.Close()
// 	s := New(r, nil, nil, nil, nil, nil)
// 	t.Run("can list existing entries", func(t *testing.T) {
// 		// given
// 		testutil.TruncateTables(db)
// 		c := factory.CreateCharacter()
// 		factory.CreateWalletJournalEntry(storage.CreateWalletJournalEntryParams{CharacterID: c.ID})
// 		factory.CreateWalletJournalEntry(storage.CreateWalletJournalEntryParams{CharacterID: c.ID})
// 		factory.CreateWalletJournalEntry(storage.CreateWalletJournalEntryParams{CharacterID: c.ID})
// 		// when
// 		ee, err := s.ListWalletJournalEntries(c.ID)
// 		// then
// 		if assert.NoError(t, err) {
// 			assert.Len(t, ee, 3)
// 		}
// 	})
// }
