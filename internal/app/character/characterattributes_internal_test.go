package character

import (
	"context"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

func TestUpdateCharacterAttributesESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newCharacterService(st)
	ctx := context.Background()
	t.Run("should create attributes from ESI response", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
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
		changed, err := s.updateCharacterAttributesESI(ctx, app.CharacterUpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionAttributes,
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

func TestGetCharacterAttributes(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	cs := newCharacterService(st)
	ctx := context.Background()
	t.Run("should return own error when object not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		// when
		_, err := cs.GetCharacterAttributes(ctx, 42)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("should return obj when found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		x1 := factory.CreateCharacterAttributes()
		// when
		x2, err := cs.GetCharacterAttributes(ctx, x1.CharacterID)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, x1, x2)
		}
	})
}
