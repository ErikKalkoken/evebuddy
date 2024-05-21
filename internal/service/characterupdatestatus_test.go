package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestHasCharacterSectionChanged(t *testing.T) {
	db, r, factory := testutil.New()
	s := NewService(r)
	ctx := context.Background()
	t.Run("should report as changed when new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		section := model.CharacterSectionImplants
		// when
		changed, err := s.recordCharacterSectionUpdate(ctx, c.ID, section, "new")
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.WithinDuration(t, time.Now(), x.LastUpdatedAt.Time, 5*time.Second)
				assert.True(t, x.IsOK())
			}
		}
	})
	t.Run("should report as changed when data has changed and store update and reset error", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		section := model.CharacterSectionImplants
		x1 := factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     section,
			Error:       "error",
		})
		// when
		changed, err := s.recordCharacterSectionUpdate(ctx, c.ID, section, "new")
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x2, err := r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.LastUpdatedAt.Time, x1.LastUpdatedAt.Time)
				assert.True(t, x2.IsOK())
			}
		}
	})
	t.Run("should report as unchanged when data has not changed and store update", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		section := model.CharacterSectionImplants
		x1 := factory.CreateCharacterUpdateStatus(testutil.CharacterUpdateStatusParams{
			CharacterID: c.ID,
			Section:     section,
			Data:        "old",
		})
		// when
		changed, err := s.recordCharacterSectionUpdate(ctx, c.ID, section, "old")
		if assert.NoError(t, err) {
			assert.False(t, changed)
			x2, err := r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.Greater(t, x2.LastUpdatedAt.Time, x1.LastUpdatedAt.Time)
				assert.True(t, x2.IsOK())
			}
		}
	})
}
func TestUpdateCharacterSectionIfExpired(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewService(r)
	section := model.CharacterSectionImplants
	ctx := context.Background()
	t.Run("should report true when changed", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 100})
		data := `[100]`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))
		// when
		changed, err := s.UpdateCharacterSectionIfExpired(c.ID, section)
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := s.r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.True(t, x.IsOK())
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
		data := `{
			"error": "dummy error"
		}`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewStringResponder(500, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))
		// when
		_, err := s.UpdateCharacterSectionIfExpired(c.ID, section)
		// then
		if assert.Error(t, err) {
			x, err := s.r.GetCharacterUpdateStatus(ctx, c.ID, section)
			if assert.NoError(t, err) {
				assert.False(t, x.IsOK())
				assert.Equal(t, "500: dummy error", x.ErrorMessage)
			}
		}
	})

}
