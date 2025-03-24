package characterservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestUpdateCharacterSkillsESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newCharacterService(st)
	ctx := context.Background()
	t.Run("should update skills from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 41})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 42})
		data := map[string]any{
			"skills": []map[string]any{
				{
					"active_skill_level":   3,
					"skill_id":             41,
					"skillpoints_in_skill": 10000,
					"trained_skill_level":  4,
				},
				{
					"active_skill_level":   1,
					"skill_id":             42,
					"skillpoints_in_skill": 20000,
					"trained_skill_level":  2,
				},
			},
			"total_sp": 90000,
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/skills/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateCharacterSkillsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionSkills,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			c2, err := st.GetCharacter(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 90000, c2.TotalSP.ValueOrZero())
			}
			o1, err := st.GetCharacterSkill(ctx, c.ID, 41)
			if assert.NoError(t, err) {
				assert.Equal(t, 3, o1.ActiveSkillLevel)
				assert.Equal(t, 10000, o1.SkillPointsInSkill)
				assert.Equal(t, 4, o1.TrainedSkillLevel)
			}
			o2, err := st.GetCharacterSkill(ctx, c.ID, 42)
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
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 41})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 42})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID,
			EveTypeID:   41,
		})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID: c.ID,
			EveTypeID:   42,
		})
		data := map[string]any{
			"skills": []map[string]any{
				{
					"active_skill_level":   3,
					"skill_id":             41,
					"skillpoints_in_skill": 10000,
					"trained_skill_level":  4,
				},
			},
			"total_sp": 90000,
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v4/characters/%d/skills/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateCharacterSkillsESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionSkills,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			ids, err := st.ListCharacterSkillIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, []int32{41}, ids.ToSlice())
			}
		}
	})

}

// func TestListWalletJournalEntries(t *testing.T) {
// 	db, r, factory := testutil.New()
// 	defer db.Close()
// 	s := newCharacterService(st)
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
