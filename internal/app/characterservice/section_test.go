package characterservice_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestCharacterService_HasSection(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st})
	section := app.SectionCharacterImplants
	ctx := context.Background()
	t.Run("should report true when exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		// when
		got, err := s.HasSection(ctx, c.ID, section)
		// then
		require.NoError(t, err)
		assert.True(t, got)
	})
	t.Run("should report false when not exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		// when
		got, err := s.HasSection(ctx, c.ID, section)
		// then
		require.NoError(t, err)
		assert.False(t, got)
	})
	t.Run("should report false when exist, but incomplete", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
		})
		_, err := st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     section,
			CompletedAt: &sql.NullTime{},
		})
		require.NoError(t, err)
		// when
		got, err := s.HasSection(ctx, c.ID, section)
		// then
		require.NoError(t, err)
		assert.False(t, got)
	})
}

func TestCharacterService_Init(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	c := factory.CreateCharacterFull()
	o := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: c.ID,
		Section:     app.SectionCharacterImplants,
	})
	require.False(t, o.StartedAt.IsZero())
	scs := new(statuscache.StatusCache)
	require.NoError(t, scs.Init(ctx, st))
	x0, _ := scs.CharacterSection(c.ID, app.SectionCharacterImplants)
	require.True(t, x0.IsRunning())
	s := testdouble.NewCharacterServiceFake(characterservice.Params{Storage: st, StatusCacheService: scs})
	defer s.Stop() // waits for background jobs started by Init
	// when
	err := s.Init(ctx)
	// then
	require.NoError(t, err)
	got, err := st.GetCharacterSectionStatus(ctx, c.ID, app.SectionCharacterImplants)
	require.NoError(t, err)
	assert.True(t, got.StartedAt.IsZero())
	x, ok := scs.CharacterSection(c.ID, app.SectionCharacterImplants)
	require.True(t, ok)
	assert.False(t, x.IsRunning())
}
