package characterservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestUpdateCharacterImplantsESI(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	s := NewFake(st)
	ctx := context.Background()
	t.Run("should create new implants from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		et1 := factory.CreateEveType()
		et2 := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/implants", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int64{et1.ID, et2.ID}))

		// when
		changed, err := s.updateImplantsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterImplants,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCharacterImplantIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(et1.ID, et2.ID)
		xassert.Equal(t, want, got)
	})

	t.Run("should update when implants have changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		ci := factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		et := factory.CreateEveType()
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/implants", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int64{et.ID, ci.EveType.ID}))

		// when
		changed, err := s.updateImplantsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterImplants,
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		got, err := st.ListCharacterImplantIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(et.ID, ci.EveType.ID)
		xassert.Equal(t, want, got)
	})

	t.Run("should do nothing when implants have not changed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		ci := factory.CreateCharacterImplant(storage.CreateCharacterImplantParams{CharacterID: c.ID})
		httpmock.RegisterResponder(
			"GET",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/implants", c.ID),
			httpmock.NewJsonResponderOrPanic(200, []int64{ci.EveType.ID}))

		// when
		changed, err := s.updateImplantsESI(ctx, characterSectionUpdateParams{
			characterID: c.ID,
			section:     app.SectionCharacterImplants,
		})
		// then
		require.NoError(t, err)
		assert.False(t, changed)
		got, err := st.ListCharacterImplantIDs(ctx, c.ID)
		require.NoError(t, err)
		want := set.Of(ci.EveType.ID)
		xassert.Equal(t, want, got)

	})
}
