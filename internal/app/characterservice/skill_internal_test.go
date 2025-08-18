package characterservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestGetAttributes(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	cs := NewFake(st)
	ctx := context.Background()
	t.Run("should return own error when object not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := cs.GetAttributes(ctx, 42)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return obj when found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x1 := factory.CreateCharacterAttributes()
		// when
		x2, err := cs.GetAttributes(ctx, x1.CharacterID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
}

func TestUpdateCharacterAttributesESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create attributes from ESI response", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		data := map[string]int{
			"charisma":     20,
			"intelligence": 21,
			"memory":       22,
			"perception":   23,
			"willpower":    24,
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/attributes/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateAttributesESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterAttributes,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCharacterAttributes(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, 20, x.Charisma)
				assert.Equal(t, 21, x.Intelligence)
				assert.Equal(t, 22, x.Memory)
				assert.Equal(t, 23, x.Perception)
				assert.Equal(t, 24, x.Willpower)
			}
		}
	})
}

func TestUpdateCharacterSkillsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should update skills from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
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
		changed, err := s.updateSkillsESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkills,
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
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
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
		changed, err := s.updateSkillsESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkills,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			ids, err := st.ListCharacterSkillIDs(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, []int32{41}, ids.Slice())
			}
		}
	})

}

// func TestListWalletJournalEntries(t *testing.T) {
// 	db, r, factory := testutil.NewDBOnDisk(t.TempDir())
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

func TestUpdateSkillqueueESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create new queue", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		t1 := factory.CreateEveType()
		t2 := factory.CreateEveType()
		data := []map[string]any{
			{
				"finish_date":    "2016-06-29T10:47:00Z",
				"finished_level": 3,
				"queue_position": 0,
				"skill_id":       t1.ID,
				"start_date":     "2016-06-29T10:46:00Z",
			},
			{
				"finish_date":    "2016-07-15T10:47:00Z",
				"finished_level": 4,
				"queue_position": 1,
				"skill_id":       t1.ID,
				"start_date":     "2016-06-29T10:47:00Z",
			},
			{
				"finish_date":    "2016-08-30T10:47:00Z",
				"finished_level": 2,
				"queue_position": 2,
				"skill_id":       t2.ID,
				"start_date":     "2016-07-15T10:47:00Z",
			}}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v2/characters/%d/skillqueue/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateSkillqueueESI(ctx, app.CharacterSectionUpdateParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkillqueue,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			ii, err := st.ListCharacterSkillqueueItems(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ii, 3)
			}
		}
	})
}
