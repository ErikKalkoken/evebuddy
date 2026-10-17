package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCharacterSectionStatus(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can list", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkillqueue,
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterImplants,
		})
		// when
		oo, err := st.ListCharacterSectionStatus(ctx, c.ID)
		// then
		require.NoError(t, err)
		assert.Len(t, oo, 2)
	})
	t.Run("can count characters with completed section", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c1.ID,
			Section:     app.SectionCharacterMailHeaders,
		})
		c2 := factory.CreateCharacterFull()
		x := "error"
		_, err := st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID:  c2.ID,
			Section:      app.SectionCharacterMailHeaders,
			ErrorMessage: &x,
		}) // never completed
		require.NoError(t, err)
		c3 := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c3.ID,
			Section:     app.SectionCharacterImplants,
		})
		// when
		got, err := st.CountCharactersWithSection(ctx, app.SectionCharacterMailHeaders)
		// then
		require.NoError(t, err)
		assert.Equal(t, 1, got)
	})
	t.Run("can set from scratch", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		// when
		x := "error"
		arg := storage.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      app.SectionCharacterImplants,
			ErrorMessage: &x,
		}
		x1, err := st.UpdateOrCreateCharacterSectionStatus(ctx, arg)
		// then
		require.NoError(t, err)
		require.NoError(t, err)
		xassert.Equal(t, "", x1.ContentHash)
		xassert.Equal(t, "error", x1.ErrorMessage)
		assert.True(t, x1.CompletedAt.IsZero())
		assert.False(t, x1.UpdatedAt.IsZero())
		x2, err := st.GetCharacterSectionStatus(ctx, c.ID, app.SectionCharacterImplants)
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})
	t.Run("can set existing", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		x := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterImplants,
		})
		// when
		s := "error"
		arg := storage.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      x.Section.(app.CharacterSection),
			ErrorMessage: &s,
		}
		x1, err := st.UpdateOrCreateCharacterSectionStatus(ctx, arg)
		// then
		require.NoError(t, err)
		xassert.Equal(t, x.ContentHash, x1.ContentHash)
		xassert.Equal(t, "error", x1.ErrorMessage)
		xassert.Equal(t, x.CompletedAt, x1.CompletedAt)
		xassert.Equal(t, x.StartedAt, x1.StartedAt)
		assert.False(t, x1.UpdatedAt.IsZero())
		x2, err := st.GetCharacterSectionStatus(ctx, c.ID, x.Section.(app.CharacterSection))
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})
	t.Run("can set udpated at", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		x := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterImplants,
		})
		// when
		ua := time.Now().UTC().Add(-12 * time.Hour)
		arg := storage.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     x.Section.(app.CharacterSection),
			UpdatedAt:   &ua,
		}
		x1, err := st.UpdateOrCreateCharacterSectionStatus(ctx, arg)
		// then
		require.NoError(t, err)
		xassert.Equal(t, x.ContentHash, x1.ContentHash)
		xassert.Equal(t, x.ErrorMessage, x1.ErrorMessage)
		xassert.Equal(t, x.CompletedAt, x1.CompletedAt)
		xassert.Equal(t, x.StartedAt, x1.StartedAt)
		xassert.Equal(t, ua, x1.UpdatedAt)
		x2, err := st.GetCharacterSectionStatus(ctx, c.ID, x.Section.(app.CharacterSection))
		require.NoError(t, err)
		xassert.Equal(t, x1, x2)
	})
	t.Run("can clear started at", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		running := factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID:  c.ID,
			Section:      app.SectionCharacterImplants,
			ErrorMessage: "error",
		})
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkillqueue,
		})
		_, err := st.UpdateOrCreateCharacterSectionStatus(ctx, storage.UpdateOrCreateCharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterSkillqueue,
			StartedAt:   new(optional.Optional[time.Time]),
		})
		require.NoError(t, err)
		// when
		oo, err := st.ClearCharacterSectionStatusStartedAt(ctx)
		// then
		require.NoError(t, err)
		require.Len(t, oo, 1)
		got := oo[0]
		assert.True(t, got.StartedAt.IsZero())
		assert.Equal(t, running.Section, got.Section)
		assert.True(t, running.CompletedAt.Equal(got.CompletedAt))
		assert.Equal(t, running.ContentHash, got.ContentHash)
		assert.Equal(t, running.ErrorMessage, got.ErrorMessage)
		assert.True(t, running.UpdatedAt.Equal(got.UpdatedAt))
		x, err := st.GetCharacterSectionStatus(ctx, c.ID, app.SectionCharacterImplants)
		require.NoError(t, err)
		assert.True(t, x.StartedAt.IsZero())
	})
}
