package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
)

func TestUpdateCharacterAttributesESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewService(r)
	ctx := context.Background()
	t.Run("should create attributes from ESI response", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		data := `{
			"charisma": 20,
			"intelligence": 21,
			"memory": 22,
			"perception": 23,
			"willpower": 24
		  }`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/attributes/", c.ID),
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		changed, err := s.updateCharacterAttributesESI(ctx, UpdateCharacterSectionParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionAttributes,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			x, err := r.GetCharacterAttributes(ctx, c.ID)
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
