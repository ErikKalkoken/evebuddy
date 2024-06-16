package character

import (
	"context"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite"
	"github.com/ErikKalkoken/evebuddy/internal/app/sqlite/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

func TestUpdateCharacterImplantsESI(t *testing.T) {
	db, st, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := newCharacterService(st)
	ctx := context.Background()
	t.Run("should create new implants from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(app.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(sqlite.CreateEveTypeParams{ID: 100})
		factory.CreateEveType(sqlite.CreateEveTypeParams{ID: 101})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int32{100, 101}))

		// when
		changed, err := s.updateCharacterImplantsESI(ctx, UpdateSectionParams{
			CharacterID: c.ID,
			Section:     app.SectionImplants,
		})
		// then
		if assert.NoError(t, err) {
			assert.True(t, changed)
			oo, err := st.ListCharacterImplants(ctx, c.ID)
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
