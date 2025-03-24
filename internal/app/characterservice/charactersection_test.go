package characterservice_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCharacterSection(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newCharacterService(st)
	section := app.SectionImplants
	ctx := context.Background()
	t.Run("should report true when changed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{et.ID}))
		// when
		changed, err := s.UpdateSectionIfNeeded(
			ctx, app.CharacterUpdateSectionParams{CharacterID: c.ID, Section: section})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
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
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
			CompletedAt: time.Now().Add(-6 * time.Hour),
			Data:        data,
		})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		changed, err := s.UpdateSectionIfNeeded(
			ctx, app.CharacterUpdateSectionParams{CharacterID: c.ID, Section: section})
		// then
		if assert.NoError(t, err) {
			assert.False(t, changed)
			x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.CompletedAt, 5*time.Second)
			}
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			xx, err := st.ListCharacterImplants(ctx, c.ID)
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
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{et.ID}))
		// when
		changed, err := s.UpdateSectionIfNeeded(
			ctx, app.CharacterUpdateSectionParams{
				CharacterID: c.ID,
				Section:     section,
			})
		// then
		if assert.NoError(t, err) {
			assert.False(t, changed)
			assert.Equal(t, 0, httpmock.GetTotalCallCount())
			xx, err := st.ListCharacterImplants(ctx, c.ID)
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
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(500, map[string]string{"error": "dummy error"}))
		// when
		_, err := s.UpdateSectionIfNeeded(
			ctx, app.CharacterUpdateSectionParams{CharacterID: c.ID, Section: section})
		// then
		if assert.Error(t, err) {
			x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
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
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{et.ID}))
		// when
		_, err := s.UpdateSectionIfNeeded(
			ctx, app.CharacterUpdateSectionParams{
				CharacterID: c.ID,
				Section:     section,
				ForceUpdate: true,
			})
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			xx, err := st.ListCharacterImplants(ctx, c.ID)
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
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
			CompletedAt: time.Now().Add(-6 * time.Hour),
			Data:        data,
		})
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))
		// when
		_, err := s.UpdateSectionIfNeeded(
			ctx, app.CharacterUpdateSectionParams{
				CharacterID: c.ID,
				Section:     section,
				ForceUpdate: true,
			})
		// then
		if assert.NoError(t, err) {
			x, err := st.GetCharacterSectionStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.CompletedAt, 5*time.Second)
			}
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			xx, err := st.ListCharacterImplants(ctx, c.ID)
			if assert.NoError(t, err) {
				assert.Len(t, xx, 1)
			}
		}
	})
}
