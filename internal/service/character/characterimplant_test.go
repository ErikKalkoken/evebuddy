package character

import (
	"context"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/ErikKalkoken/evebuddy/internal/storage/testutil"
)

func TestUpdateCharacterImplantsESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := New(r, nil, nil, nil, nil, nil)
	ctx := context.Background()
	t.Run("should create new implants from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 100})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 101})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{100, 101}))

		// when
		changed, err := s.updateCharacterImplantsESI(ctx, UpdateSectionParams{
			CharacterID: c.ID,
			Section:     model.SectionImplants,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			oo, err := r.ListCharacterImplants(ctx, c.ID)
			if assert.NoError(t, err) {
				got := set.New[int32]()
				for _, o := range oo {
					got.Add(o.EveType.ID)
				}
				want := set.NewFromSlice([]int32{100, 101})
				assert.Equal(t, want, got)
			}

		}
	})
}
