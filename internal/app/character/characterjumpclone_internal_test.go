package character

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

func TestUpdateCharacterJumpClonesESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newCharacterService(st)
	ctx := context.Background()
	t.Run("should create new clones from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 22118})
		factory.CreateLocationStructure(storage.UpdateOrCreateLocationParams{ID: 60003463})
		factory.CreateLocationStructure(storage.UpdateOrCreateLocationParams{ID: 1021348135816})
		data := map[string]any{
			"home_location": map[string]any{
				"location_id":   1021348135816,
				"location_type": "structure",
			},
			"jump_clones": []map[string]any{
				{
					"implants":      []int{22118},
					"jump_clone_id": 12345,
					"location_id":   60003463,
					"location_type": "station",
				},
			},
		}
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v3/characters/%d/clones/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, data))

		// when
		changed, err := s.updateCharacterJumpClonesESI(ctx, UpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionJumpClones,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			oo, err := st.ListCharacterJumpClones(ctx, c.ID)
			if assert.NoError(t, err) {
				if assert.Len(t, oo, 1) {
					o := oo[0]
					assert.Equal(t, int32(12345), o.JumpCloneID)
					assert.Equal(t, int64(60003463), o.Location.ID)
					if assert.Len(t, o.Implants, 1) {
						x := o.Implants[0]
						assert.Equal(t, int32(22118), x.EveType.ID)
					}
				}
			}
		}
	})
}
