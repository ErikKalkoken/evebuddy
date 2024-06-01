package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/helper/set"
	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
)

func TestUpdateCharacterImplantsESI(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewService(r)
	ctx := context.Background()
	t.Run("should create new implants from scratch", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(model.CharacterToken{CharacterID: c.ID})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 100})
		factory.CreateEveType(storage.CreateEveTypeParams{ID: 101})
		data := `[
			100,
			101
		  ]`
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/v1/characters/%d/implants/", c.ID),
			httpmock.NewStringResponder(200, data).HeaderSet(http.Header{"Content-Type": []string{"application/json"}}))

		// when
		changed, err := s.updateCharacterImplantsESI(ctx, UpdateCharacterSectionParams{
			CharacterID: c.ID,
			Section:     model.CharacterSectionImplants,
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
