package storage_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/ErikKalkoken/evebuddy/internal/storage/testutil"
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
		_, err := r.UpdateOrCreateCharacterMailLabel(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterMailLabel(ctx, c.ID, 42)
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
		factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c.ID, LabelID: 42})
		arg := storage.MailLabelParams{
			CharacterID: c.ID,
			Color:       "xyz",
			LabelID:     42,
			Name:        "Dummy",
			UnreadCount: 99,
		}
		// when
		_, err := r.UpdateOrCreateCharacterMailLabel(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterMailLabel(ctx, c.ID, 42)
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
		factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c.ID, LabelID: 42, Name: "Dummy"})
		arg := storage.MailLabelParams{
			CharacterID: c.ID,
			Color:       "xyz",
			LabelID:     42,
			Name:        "Johnny",
			UnreadCount: 99,
		}
		// when
		_, err := r.GetOrCreateCharacterMailLabel(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterMailLabel(ctx, c.ID, 42)
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
		_, err := r.GetOrCreateCharacterMailLabel(ctx, arg)
		// then
		if assert.NoError(t, err) {
			l, err := r.GetCharacterMailLabel(ctx, c.ID, 42)
			if assert.NoError(t, err) {
				assert.Equal(t, "Johnny", l.Name)
			}
		}
	})
	t.Run("can return all mail labels for a character ordered by name", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c.ID, Name: "bravo"})
		l2 := factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c.ID, Name: "alpha"})
		factory.CreateCharacterMailLabel()
		// when
		got, err := r.ListCharacterMailLabelsOrdered(ctx, c.ID)
		if assert.NoError(t, err) {
			want := []*model.CharacterMailLabel{l2, l1}
			assert.Equal(t, want, got)
		}
	})
	t.Run("can return all mail labels for a character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c.ID, Name: "bravo"})
		l2 := factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c.ID, Name: "alpha"})
		factory.CreateCharacterMailLabel()
		// when
		got, err := r.ListCharacterMailLabelsOrdered(ctx, c.ID)
		if assert.NoError(t, err) {
			want := []*model.CharacterMailLabel{l2, l1}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return empty list when character has no mail labels", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterMailLabel()
		// when
		labels, err := r.ListCharacterMailLabelsOrdered(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Len(t, labels, 0)
		}
	})
}

func TestDeleteObsoleteMailLabels(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can delete obsolete mail labels for a character", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		l1 := factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c1.ID})
		factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c1.ID}) // to delete
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c1.ID, LabelIDs: []int32{l1.LabelID}})
		c2 := factory.CreateCharacter()
		l2 := factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c2.ID})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c2.ID, LabelIDs: []int32{l2.LabelID}})
		// when
		err := r.DeleteObsoleteCharacterMailLabels(ctx, c1.ID)
		if assert.NoError(t, err) {
			ids1, err := r.ListCharacterMailLabelsOrdered(ctx, c1.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids1, 1)
				assert.Equal(t, l1.LabelID, ids1[0].LabelID)
			}
			ids2, err := r.ListCharacterMailLabelsOrdered(ctx, c2.ID)
			if assert.NoError(t, err) {
				assert.Len(t, ids2, 1)
				assert.Equal(t, l2.LabelID, ids2[0].LabelID)
			}
		}
	})
}
