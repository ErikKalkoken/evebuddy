package character_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/service/character"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestCharacterUpdateStatus(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	s := character.New(r, nil, nil, nil, nil, nil)
	ctx := context.Background()
	t.Run("Can report when updated", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		updateAt := time.Now().Add(3 * time.Hour)
		factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID:   c.ID,
			Section:       model.CharacterSectionSkillqueue,
			LastUpdatedAt: updateAt,
		})
		// when
		x, err := s.CharacterSectionWasUpdated(ctx, c.ID, model.CharacterSectionSkillqueue)
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
		x, err := s.CharacterSectionWasUpdated(ctx, c.ID, model.CharacterSectionSkillqueue)
		// then
		if assert.NoError(t, err) {
			assert.False(t, x)
		}
	})
}

func TestUpdateCharacterSection(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := character.New(r, nil, nil, nil, nil, nil)
	section := model.CharacterSectionImplants
	ctx := context.Background()
	t.Run("should report true when changed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 100})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{100}))
		// when
		changed, err := s.UpdateCharacterSection(
			ctx, character.UpdateCharacterSectionParams{CharacterID: c.ID, Section: section})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.True(t, x.IsOK())
			}
		}
	})
	t.Run("should not update and report false when not changed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		data := []int32{100}
		factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID:   c.ID,
			Section:       section,
			LastUpdatedAt: time.Now().Add(-6 * time.Hour),
			Data:          data,
		})
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 100})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.UpdateCharacterSection(
			ctx, character.UpdateCharacterSectionParams{CharacterID: c.ID, Section: section})
		// then
		if assert.NoError(t, err) {
			assert.False(t, changed)
			x, err := r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.LastUpdatedAt, 5*time.Second)
			}
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			xx, err := r.ListCharacterImplants(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 0)
			}
		}
	})
	t.Run("should not fetch or update when not expired and report false", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 100})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{100}))
		// when
		changed, err := s.UpdateCharacterSection(
			ctx, character.UpdateCharacterSectionParams{
				CharacterID: c.ID,
				Section:     section,
			})
		// then
		if assert.NoError(t, err) {
			assert.False(t, changed)
			assert.Equal(t, 0, httpmock.GetTotalCallCount())
			xx, err := r.ListCharacterImplants(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 0)
			}
		}
	})
	t.Run("should record when update failed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 100})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(500, map[string]string{"error": "dummy error"}))
		// when
		_, err := s.UpdateCharacterSection(
			ctx, character.UpdateCharacterSectionParams{CharacterID: c.ID, Section: section})
		// then
		if assert.Error(t, err) {
			x, err := r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.False(t, x.IsOK())
				assert.Equal(t, "500: dummy error", x.ErrorMessage)
			}
		}
	})
	t.Run("should fetch and update when not expired and force update requested", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 100})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{100}))
		// when
		_, err := s.UpdateCharacterSection(
			ctx, character.UpdateCharacterSectionParams{
				CharacterID: c.ID,
				Section:     section,
				ForceUpdate: true,
			})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			xx, err := r.ListCharacterImplants(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 1)
			}
		}
	})
	t.Run("should update when not changed and force update requested", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		data := []int32{100}
		factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID:   c.ID,
			Section:       section,
			LastUpdatedAt: time.Now().Add(-6 * time.Hour),
			Data:          data,
		})
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 100})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		_, err := s.UpdateCharacterSection(
			ctx, character.UpdateCharacterSectionParams{
				CharacterID: c.ID,
				Section:     section,
				ForceUpdate: true,
			})
		// then
		if assert.NoError(t, err) {
			x, err := r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.LastUpdatedAt, 5*time.Second)
			}
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			xx, err := r.ListCharacterImplants(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 1)
			}
		}
	})
}
