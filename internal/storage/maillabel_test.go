package storage_test

import (
	"context"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
	"example/evebuddy/internal/testutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMailLabel(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		arg := storage.MailLabelParams{
			CharacterID: c.ID,
			Color:       "xyz",
			LabelID:     42,
			Name:        "Dummy",
			UnreadCount: 99,
		}
		// when
		_, err := r.UpdateOrCreateMailLabel(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetMailLabel(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Dummy", l.Name)
				assert.Equal(t, "xyz", l.Color)
				assert.Equal(t, 99, l.UnreadCount)
			}
		}
	})
	t.Run("can update existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID, LabelID: 42})
		arg := storage.MailLabelParams{
			CharacterID: c.ID,
			Color:       "xyz",
			LabelID:     42,
			Name:        "Dummy",
			UnreadCount: 99,
		}
		// when
		_, err := r.UpdateOrCreateMailLabel(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetMailLabel(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Dummy", l.Name)
				assert.Equal(t, "xyz", l.Color)
				assert.Equal(t, 99, l.UnreadCount)
			}
		}
	})
	t.Run("can get or create existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID, LabelID: 42, Name: "Dummy"})
		arg := storage.MailLabelParams{
			CharacterID: c.ID,
			Color:       "xyz",
			LabelID:     42,
			Name:        "Johnny",
			UnreadCount: 99,
		}
		// when
		_, err := r.GetOrCreateMailLabel(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetMailLabel(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Dummy", l.Name)
			}
		}
	})
	t.Run("can get or create when not existing", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		arg := storage.MailLabelParams{
			CharacterID: c.ID,
			Color:       "xyz",
			LabelID:     42,
			Name:        "Johnny",
			UnreadCount: 99,
		}
		// when
		_, err := r.GetOrCreateMailLabel(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetMailLabel(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Johnny", l.Name)
			}
		}
	})
	t.Run("can return all mail labels for a character ordered by name", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID, Name: "bravo"})
		l2 := factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID, Name: "alpha"})
		factory.CreateMailLabel()
		// when
		got, err := r.ListMailLabelsOrdered(ctx, c.ID)
		if assert.NoError(t, err) {
			want := []model.MailLabel{l2, l1}
			assert.Equal(t, want, got)
		}
	})
	t.Run("can return all mail labels for a character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID, Name: "bravo"})
		l2 := factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID, Name: "alpha"})
		factory.CreateMailLabel()
		// when
		got, err := r.ListMailLabelsOrdered(ctx, c.ID)
		if assert.NoError(t, err) {
			want := []model.MailLabel{l2, l1}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return empty list when character ha no mail labels", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateMailLabel()
		// when
		labels, err := r.ListMailLabelsOrdered(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Len(t, labels, 0)
		}
	})
}
