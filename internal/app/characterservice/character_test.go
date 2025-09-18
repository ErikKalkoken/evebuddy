package characterservice_test

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
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
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	ctx := context.Background()
	test.NewTempApp(t)
	t.Run("create new character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		corporation := factory.CreateEveCorporation()
		factory.CreateEveEntityWithCategory(app.EveEntityCorporation, app.EveEntity{ID: corporation.ID})
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{
			CorporationID: corporation.ID,
		})
		cs := characterservice.NewFake(st, characterservice.Params{
			SSOService: characterservice.SSOFake{Token: factory.CreateToken(app.Token{
				CharacterID:   ec.ID,
				CharacterName: ec.Name}),
			},
		})
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        ec.Birthday.Format(app.DateTimeFormatESI),
				"bloodline_id":    3,
				"corporation_id":  ec.Corporation.ID,
				"gender":          ec.Gender,
				"name":            ec.Name,
				"race_id":         ec.Race.ID,
				"security_status": ec.SecurityStatus,
				"title":           ec.Title,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/characters/affiliation/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   ec.ID,
					"corporation_id": ec.Corporation.ID,
				}}),
		)
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"ceo_id":       corporation.Ceo.ID,
				"creator_id":   corporation.Creator.ID,
				"date_founded": corporation.DateFounded.ValueOrZero().Format(app.DateTimeFormatESI),
				"description":  corporation.Description,
				"member_count": corporation.MemberCount,
				"name":         corporation.Name,
				"tax_rate":     corporation.TaxRate,
				"ticker":       corporation.Ticker,
				"url":          corporation.URL,
			}),
		)
		// when
		var info string
		got, err := cs.UpdateOrCreateCharacterFromSSO(ctx, func(s string) {
			info = s
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, ec.ID, got.ID)
			ok, err := cs.HasCharacter(ctx, ec.ID)
			if assert.NoError(t, err) {
				assert.True(t, ok)
			}
			token, err := st.GetCharacterToken(ctx, ec.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, token.CharacterID, ec.ID)
			}
			x, err := st.GetCorporation(ctx, corporation.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, corporation, x.EveCorporation)
			}
			assert.NotZero(t, info)
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
		httpmock.Reset()
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/characters/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"birthday":        ec.Birthday.Format(app.DateTimeFormatESI),
				"bloodline_id":    3,
				"corporation_id":  ec.Corporation.ID,
				"gender":          ec.Gender,
				"name":            ec.Name,
				"race_id":         ec.Race.ID,
				"security_status": ec.SecurityStatus,
				"title":           ec.Title,
			}),
		)
		httpmock.RegisterResponder(
			"POST",
			`=~^https://esi\.evetech\.net/v\d+/characters/affiliation/`,
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{
				{
					"character_id":   ec.ID,
					"corporation_id": ec.Corporation.ID,
				}}),
		)
		httpmock.RegisterResponder(
			"GET",
			`=~^https://esi\.evetech\.net/v\d+/corporations/\d+/`,
			httpmock.NewJsonResponderOrPanic(200, map[string]any{
				"ceo_id":       corporation.Ceo.ID,
				"creator_id":   corporation.Creator.ID,
				"date_founded": corporation.DateFounded.ValueOrZero().Format(app.DateTimeFormatESI),
				"description":  corporation.Description,
				"member_count": corporation.MemberCount,
				"name":         corporation.Name,
				"tax_rate":     corporation.TaxRate,
				"ticker":       corporation.Ticker,
				"url":          corporation.URL,
			}),
		)
		// when
		var info string
		got, err := cs.UpdateOrCreateCharacterFromSSO(ctx, func(s string) {
			info = s
		})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, c.ID, got.ID)
			token, err := st.GetCharacterToken(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Equal(t, token.CharacterID, c.ID)
			}
			assert.NotZero(t, info)
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
					ExtractorProductTypeID: optional.New(product.ID),
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
