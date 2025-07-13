package characterservice_test

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestGetCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := characterservice.NewFake(st)
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
		x1 := factory.CreateCharacterFull()
		// when
		x2, err := cs.GetCharacter(ctx, x1.ID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1.ID, x2.ID)
		}
	})
}

func TestGetAnyCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := characterservice.NewFake(st)
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
		x1 := factory.CreateCharacterFull()
		// when
		x2, err := cs.GetAnyCharacter(ctx)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
}

func TestUpdateOrCreateCharacterFromSSO(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	test.NewTempApp(t)
	t.Run("create new character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		corporation := factory.CreateEveCorporation()
		factory.CreateEveEntityWithCategory(app.EveEntityCorporation, app.EveEntity{ID: corporation.ID})
		character := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			CorporationID: corporation.ID,
		})
		cs := characterservice.NewFake(st, characterservice.Params{
			SSOService: characterservice.SSOFake{Token: factory.CreateToken(app.Token{
				CharacterID:   character.ID,
				CharacterName: character.Name})},
		})
		var info string
		b := binding.BindString(&info)
		got, err := cs.UpdateOrCreateCharacterFromSSO(ctx, b)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, character.ID, got)
			ok, err := cs.HasCharacter(ctx, character.ID)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
			token, err := st.GetCharacterToken(ctx, character.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, token.CharacterID, character.ID)
			}
			x, err := st.GetCorporation(ctx, corporation.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, corporation, x.EveCorporation)
			}
		}
	})
	t.Run("update existing character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		corporation := factory.CreateEveCorporation()
		factory.CreateEveEntityWithCategory(app.EveEntityCorporation, app.EveEntity{ID: corporation.ID})
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			CorporationID: corporation.ID,
		})
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{ID: ec.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{
			AccessToken: "oldToken",
			CharacterID: c.ID,
		})
		cs := characterservice.NewFake(st, characterservice.Params{
			SSOService: characterservice.SSOFake{Token: factory.CreateToken(app.Token{
				CharacterID:   c.ID,
				CharacterName: c.EveCharacter.Name})},
		})
		var info string
		b := binding.BindString(&info)
		got, err := cs.UpdateOrCreateCharacterFromSSO(ctx, b)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c.ID, got)
			token, err := st.GetCharacterToken(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, token.CharacterID, c.ID)
			}
		}
	})
}

func TestTrainingWatchers(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("should enable watchers for characters with active queues only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacterFull()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c1.ID})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c1.ID,
			Section:     app.SectionSkillqueue,
			CompletedAt: time.Now().UTC(),
		})
		c2 := factory.CreateCharacterFull()
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
		c1 := factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
		c2 := factory.CreateCharacterFull()
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
		c1 := factory.CreateCharacterFull()
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c1.ID})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c1.ID,
			Section:     app.SectionSkillqueue,
			CompletedAt: time.Now().UTC(),
		})
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
		c1 := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c1.ID,
			Section:     app.SectionSkillqueue,
			CompletedAt: time.Now().UTC(),
		})
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

func TestNotifyUpdatedContracts(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	const characterID = 7
	earliest := time.Now().UTC().Add(-6 * time.Hour)
	now := time.Now().UTC()
	cases := []struct {
		name           string
		acceptorID     int32
		status         app.ContractStatus
		statusNotified app.ContractStatus
		typ            app.ContractType
		updatedAt      time.Time
		shouldNotify   bool
	}{
		{"notify new courier 1", 42, app.ContractStatusInProgress, app.ContractStatusUndefined, app.ContractTypeCourier, now, true},
		{"notify new courier 2", 42, app.ContractStatusFinished, app.ContractStatusUndefined, app.ContractTypeCourier, now, true},
		{"notify new courier 3", 42, app.ContractStatusFailed, app.ContractStatusUndefined, app.ContractTypeCourier, now, true},
		{"don't notify courier", 0, app.ContractStatusOutstanding, app.ContractStatusUndefined, app.ContractTypeCourier, now, false},
		{"notify new item exchange", 42, app.ContractStatusFinished, app.ContractStatusUndefined, app.ContractTypeItemExchange, now, true},
		{"don't notify again", 42, app.ContractStatusInProgress, app.ContractStatusInProgress, app.ContractTypeCourier, now, false},
		{"don't notify when acceptor is character", characterID, app.ContractStatusInProgress, app.ContractStatusUndefined, app.ContractTypeCourier, now, false},
		{"don't notify when contract is too old", 42, app.ContractStatusInProgress, app.ContractStatusUndefined, app.ContractTypeCourier, now.Add(-12 * time.Hour), false},
		{"don't notify item exchange", 0, app.ContractStatusOutstanding, app.ContractStatusUndefined, app.ContractTypeItemExchange, now, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.TruncateTables(db)
			if tc.acceptorID != 0 {
				factory.CreateEveEntityCharacter(app.EveEntity{ID: tc.acceptorID})
			}
			ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{ID: characterID})
			c := factory.CreateCharacterFull(storage.CreateCharacterParams{ID: ec.ID})
			o := factory.CreateCharacterContract(storage.CreateCharacterContractParams{
				AcceptorID:     tc.acceptorID,
				CharacterID:    c.ID,
				Status:         tc.status,
				StatusNotified: tc.statusNotified,
				Type:           tc.typ,
				UpdatedAt:      tc.updatedAt,
			})
			var sendCount int
			// when
			err := cs.NotifyUpdatedContracts(ctx, o.CharacterID, earliest, func(title string, content string) {
				sendCount++
			})
			// then
			if assert.NoError(t, err) {
				assert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}

func TestNotifyCommunications(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	now := time.Now().UTC()
	earliest := now.Add(-12 * time.Hour)
	typesEnabled := set.Of(string(evenotification.StructureUnderAttack))
	cases := []struct {
		name         string
		typ          evenotification.Type
		timestamp    time.Time
		isProcessed  bool
		shouldNotify bool
	}{
		{"send unprocessed", evenotification.StructureUnderAttack, now, false, true},
		{"don't send old unprocessed", evenotification.StructureUnderAttack, now.Add(-16 * time.Hour), false, false},
		{"don't send not enabled types", evenotification.SkyhookOnline, now, false, false},
		{"don't resend already processed", evenotification.StructureUnderAttack, now, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.TruncateTables(db)
			n := factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
				IsProcessed: tc.isProcessed,
				Title:       optional.From("title"),
				Body:        optional.From("body"),
				Type:        string(tc.typ),
				Timestamp:   tc.timestamp,
			})
			var sendCount int
			// when
			err := cs.NotifyCommunications(ctx, n.CharacterID, earliest, typesEnabled, func(title string, content string) {
				sendCount++
			})
			// then
			if assert.NoError(t, err) {
				assert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}

func TestCountNotifications(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	// given
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	c := factory.CreateCharacterFull()
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		CharacterID: c.ID,
		Type:        string(evenotification.StructureDestroyed),
	})
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		CharacterID: c.ID,
		Type:        string(evenotification.MoonminingExtractionStarted),
	})
	factory.CreateCharacterNotification(storage.CreateCharacterNotificationParams{
		CharacterID: c.ID,
		Type:        string(evenotification.MoonminingExtractionStarted),
		IsRead:      true,
	})
	factory.CreateCharacterNotification()
	// when
	got, err := cs.CountNotifications(ctx, c.ID)
	if assert.NoError(t, err) {
		want := map[app.NotificationGroup][]int{
			app.GroupStructure:  {1, 1},
			app.GroupMoonMining: {2, 1},
		}
		assert.Equal(t, want, got)
	}
}

func TestNotifyExpiredExtractions(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	now := time.Now().UTC()
	earliest := now.Add(-24 * time.Hour)
	cases := []struct {
		name         string
		isExtractor  bool
		expiryTime   time.Time
		lastNotified time.Time
		shouldNotify bool
	}{
		{"extraction expired and not yet notified", true, now.Add(-3 * time.Hour), time.Time{}, true},
		{"extraction expired and already notified", true, now.Add(-3 * time.Hour), now.Add(-3 * time.Hour), false},
		{"extraction not expired", true, now.Add(3 * time.Hour), time.Time{}, false},
		{"extraction expired old", true, now.Add(-48 * time.Hour), time.Time{}, false},
		{"no expiration date", true, time.Time{}, time.Time{}, false},
		{"non extractor expired", false, now.Add(-3 * time.Hour), time.Time{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			testutil.TruncateTables(db)
			product := factory.CreateEveType()
			p := factory.CreateCharacterPlanet(storage.CreateCharacterPlanetParams{
				LastNotified: tc.lastNotified,
			})
			if tc.isExtractor {
				factory.CreatePlanetPinExtractor(storage.CreatePlanetPinParams{
					CharacterPlanetID:      p.ID,
					ExpiryTime:             tc.expiryTime,
					ExtractorProductTypeID: optional.From(product.ID),
				})
			} else {
				factory.CreatePlanetPin(storage.CreatePlanetPinParams{
					CharacterPlanetID: p.ID,
					ExpiryTime:        tc.expiryTime,
				})
			}
			var sendCount int
			// when
			err := cs.NotifyExpiredExtractions(ctx, p.CharacterID, earliest, func(title string, content string) {
				sendCount++
			})
			// then
			if assert.NoError(t, err) {
				assert.Equal(t, tc.shouldNotify, sendCount == 1)
			}
		})
	}
}

func TestUpdateTickerNotifyExpiredTraining(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("send notification when watched & expired", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
		var sendCount int
		// when
		err := cs.NotifyExpiredTraining(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, sendCount, 1)
		}
	})
	t.Run("do nothing when not watched", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull()
		var sendCount int
		// when
		err := cs.NotifyExpiredTraining(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, sendCount, 0)
		}
	})
	t.Run("don't send notification when watched and training ongoing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{IsTrainingWatched: true})
		factory.CreateCharacterSkillqueueItem(storage.SkillqueueItemParams{CharacterID: c.ID})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionSkillqueue,
			CompletedAt: time.Now().UTC(),
		})
		var sendCount int
		// when
		err := cs.NotifyExpiredTraining(ctx, c.ID, func(title, content string) {
			sendCount++
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, sendCount, 0)
		}
	})
}

func TestDeleteCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()
	t.Run("delete character and delete corporation when it has no members anymore", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		ec := factory.CreateEveCorporation()
		corporation := factory.CreateCorporation(ec.ID)
		factory.CreateEveEntityWithCategory(app.EveEntityCorporation, app.EveEntity{ID: ec.ID})
		x := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: ec.ID})
		character := factory.CreateCharacterFull(storage.CreateCharacterParams{ID: x.ID})
		// when
		got, err := cs.DeleteCharacter(ctx, character.ID)
		// then
		if assert.NoError(t, err) {
			_, err = st.GetCharacter(ctx, character.ID)
			assert.ErrorIs(t, err, app.ErrNotFound)
			_, err = st.GetCorporation(ctx, corporation.ID)
			assert.ErrorIs(t, err, app.ErrNotFound)
			assert.True(t, got)
		}
	})
	t.Run("delete character and keep corporation when it still has members", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		ec := factory.CreateEveCorporation()
		corporation := factory.CreateCorporation(ec.ID)
		factory.CreateEveEntityWithCategory(app.EveEntityCorporation, app.EveEntity{ID: ec.ID})
		x1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: ec.ID})
		character := factory.CreateCharacterFull(storage.CreateCharacterParams{ID: x1.ID})
		x2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: ec.ID})
		factory.CreateCharacterFull(storage.CreateCharacterParams{ID: x2.ID})
		// when
		got, err := cs.DeleteCharacter(ctx, character.ID)
		// then
		if assert.NoError(t, err) {
			_, err = st.GetCharacter(ctx, character.ID)
			assert.ErrorIs(t, err, app.ErrNotFound)
			_, err = st.GetCorporation(ctx, corporation.ID)
			assert.NoError(t, err)
			assert.False(t, got)
		}
	})
}

func TestListAllCharactersIndustrySlots(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	cs := characterservice.NewFake(st)
	ctx := context.Background()

	t.Run("empty when no data", func(t *testing.T) {
		testutil.TruncateTables(db)
		got, err := cs.ListAllCharactersIndustrySlots(ctx, app.ManufacturingJob)
		if assert.NoError(t, err) {
			assert.Len(t, got, 0)
		}
	})

	t.Run("manufacturing slots for one character", func(t *testing.T) {
		testutil.TruncateTables(db)
		character := factory.CreateCharacterMinimal()
		industry := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeIndustry})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			EveTypeID:        industry.ID,
			ActiveSkillLevel: 5,
		})
		massProduction := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeMassProduction})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			EveTypeID:        massProduction.ID,
			ActiveSkillLevel: 5,
		})
		advancedMassProduction := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeAdvancedMassProduction})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			EveTypeID:        advancedMassProduction.ID,
			ActiveSkillLevel: 3,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int32(app.Manufacturing),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int32(app.Manufacturing),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int32(app.Manufacturing),
			Status:      app.JobReady,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int32(app.Manufacturing),
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
		testutil.TruncateTables(db)
		character := factory.CreateCharacterFull()
		laboratoryOperation := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeLaboratoryOperation})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			EveTypeID:        laboratoryOperation.ID,
			ActiveSkillLevel: 5,
		})
		advancedLaboratoryOperation := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeAdvancedLaboratoryOperation})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			EveTypeID:        advancedLaboratoryOperation.ID,
			ActiveSkillLevel: 3,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int32(app.TimeEfficiencyResearch),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int32(app.MaterialEfficiencyResearch),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int32(app.Copying),
			Status:      app.JobReady,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int32(app.Copying),
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
		testutil.TruncateTables(db)
		character := factory.CreateCharacterFull()
		massReactions := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeMassReactions})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			EveTypeID:        massReactions.ID,
			ActiveSkillLevel: 5,
		})
		advancedMassReactions := factory.CreateEveType(storage.CreateEveTypeParams{ID: app.EveTypeAdvancedMassReactions})
		factory.CreateCharacterSkill(storage.UpdateOrCreateCharacterSkillParams{
			CharacterID:      character.ID,
			EveTypeID:        advancedMassReactions.ID,
			ActiveSkillLevel: 3,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int32(app.Reactions1),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int32(app.Reactions2),
			Status:      app.JobActive,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int32(app.Reactions2),
			Status:      app.JobReady,
		})
		factory.CreateCharacterIndustryJob(storage.UpdateOrCreateCharacterIndustryJobParams{
			CharacterID: character.ID,
			ActivityID:  int32(app.Reactions2),
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
