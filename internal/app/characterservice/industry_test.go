package characterservice_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
)

func TestListAllCharactersIndustrySlots(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()

	t.Run("empty when no data", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		got, err := cs.ListAllCharactersIndustrySlots(ctx, app.ManufacturingJob)
		if assert.NoError(t, err) {
			assert.Len(t, got, 0)
		}
	})

	t.Run("manufacturing slots for one character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		industry := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeIndustry})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			TypeID:        industry.ID,
			ActiveSkillLevel: 5,
		})
		massProduction := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeMassProduction})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			TypeID:        massProduction.ID,
			ActiveSkillLevel: 5,
		})
		advancedMassProduction := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeAdvancedMassProduction})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			TypeID:        advancedMassProduction.ID,
			ActiveSkillLevel: 3,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int64(app.Manufacturing),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int64(app.Manufacturing),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int64(app.Manufacturing),
			Status:      app.JobReady,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int64(app.Manufacturing),
			Status:      app.JobDelivered,
		})
		got, err := cs.ListAllCharactersIndustrySlots(ctx, app.ManufacturingJob)
		if assert.NoError(t, err) {
			want := []app.CharacterIndustrySlots{
				{
					Type:          app.ManufacturingJob,
					CharacterID:   character.ID,
					CharacterName: character.EveCharacter.Name,
					Busy:          2,
					Ready:         1,
					Total:         9,
					Free:          6,
				},
			}
			assert.ElementsMatch(t, want, got)
		}
	})

	t.Run("research slots for one character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacterFull()
		laboratoryOperation := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeLaboratoryOperation})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			TypeID:        laboratoryOperation.ID,
			ActiveSkillLevel: 5,
		})
		advancedLaboratoryOperation := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeAdvancedLaboratoryOperation})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			TypeID:        advancedLaboratoryOperation.ID,
			ActiveSkillLevel: 3,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int64(app.TimeEfficiencyResearch),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int64(app.MaterialEfficiencyResearch),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int64(app.Copying),
			Status:      app.JobReady,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int64(app.Copying),
			Status:      app.JobDelivered,
		})
		got, err := cs.ListAllCharactersIndustrySlots(ctx, app.ScienceJob)
		if assert.NoError(t, err) {
			want := []app.CharacterIndustrySlots{
				{
					Type:          app.ScienceJob,
					CharacterID:   character.ID,
					CharacterName: character.EveCharacter.Name,
					Busy:          2,
					Ready:         1,
					Total:         9,
					Free:          6,
				},
			}
			assert.ElementsMatch(t, want, got)
		}
	})
	t.Run("reactions slots for one character", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacterFull()
		massReactions := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeMassReactions})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			TypeID:        massReactions.ID,
			ActiveSkillLevel: 5,
		})
		advancedMassReactions := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeAdvancedMassReactions})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			TypeID:        advancedMassReactions.ID,
			ActiveSkillLevel: 3,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int64(app.Reactions1),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int64(app.Reactions2),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int64(app.Reactions2),
			Status:      app.JobReady,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int64(app.Reactions2),
			Status:      app.JobDelivered,
		})
		got, err := cs.ListAllCharactersIndustrySlots(ctx, app.ReactionJob)
		if assert.NoError(t, err) {
			want := []app.CharacterIndustrySlots{
				{
					Type:          app.ReactionJob,
					CharacterID:   character.ID,
					CharacterName: character.EveCharacter.Name,
					Busy:          2,
					Ready:         1,
					Total:         9,
					Free:          6,
				},
			}
			assert.ElementsMatch(t, want, got)
		}
	})
}
