package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/ErikKalkoken/evebuddy/internal/storage/testutil"
)

func TestMailCreate(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		f := factory.CreateEveEntity()
		recipient := factory.CreateEveEntity()
		label := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID})
		// when
		arg := storage.CreateCharacterMailParams{
			Body:         "body",
			CharacterID:  c.ID,
			FromID:       f.ID,
			IsRead:       false,
			LabelIDs:     []int32{label.LabelID},
			MailID:       42,
			RecipientIDs: []int32{recipient.ID},
			Subject:      "subject",
			Timestamp:    time.Now(),
		}
		_, err := r.CreateCharacterMail(ctx, arg)
		// then
		if assert.NoError(t, err) {
			m, err := r.GetCharacterMail(ctx, c.ID, 42)
			assert.NoError(t, err)
			assert.Equal(t, int32(42), m.MailID)
			assert.Equal(t, "body", m.Body)
			assert.Equal(t, f, m.From)
			assert.Equal(t, c.ID, m.CharacterID)
			assert.Equal(t, "subject", m.Subject)
			assert.False(t, m.Timestamp.IsZero())
			assert.Equal(t, []*app.EveEntity{recipient}, m.Recipients)
			assert.Equal(t, label.Name, m.Labels[0].Name)
			assert.Equal(t, label.LabelID, m.Labels[0].LabelID)
		}
	})
}

func TestMail(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		_, err := r.GetCharacterMail(ctx, c.ID, 99)
		// then
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("can list mail IDs", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		for i := range 3 {
			factory.CreateCharacterMail(storage.CreateCharacterMailParams{
				CharacterID: c.ID,
				MailID:      int32(10 + i),
			})
		}
		// when
		ids, err := r.ListCharacterMailIDs(ctx, c.ID)
		// then
		assert.NoError(t, err)
		got := set.NewFromSlice(ids)
		want := set.NewFromSlice([]int32{10, 11, 12})
		assert.Equal(t, want, got)
	})
	t.Run("can delete existing mail", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		m := factory.CreateCharacterMail()
		// when
		err := r.DeleteCharacterMail(ctx, m.CharacterID, m.MailID)
		// then
		if assert.NoError(t, err) {
			_, err := r.GetCharacterMail(ctx, m.CharacterID, m.MailID)
			assert.ErrorIs(t, err, storage.ErrNotFound)
		}
	})
}

func TestFetchUnreadCounts(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("can get mail label unread counts", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		corp := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: app.MailLabelCorp})
		inbox := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: app.MailLabelInbox})
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: app.MailLabelAlliance})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{inbox.LabelID}, IsRead: false})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{corp.LabelID}, IsRead: true})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{corp.LabelID}, IsRead: false})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{corp.LabelID}, IsRead: false})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID})
		// when
		r, err := r.GetCharacterMailLabelUnreadCounts(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, map[int32]int{app.MailLabelCorp: 2, app.MailLabelInbox: 1}, r)
		}
	})
	t.Run("can get mail list unread counts", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateCharacterMailList(c.ID)
		factory.CreateCharacterMailList(c.ID)
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int32{l1.ID},
			IsRead:       false,
		})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int32{l1.ID},
			IsRead:       true,
		})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID})
		// when
		r, err := r.GetCharacterMailListUnreadCounts(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, map[int32]int{l1.ID: 1}, r)
		}
	})

}

func TestGetMailUnreadCount(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return correct unread count when mails exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		corp := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: app.MailLabelCorp})
		inbox := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: app.MailLabelInbox})
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: app.MailLabelAlliance})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{inbox.LabelID}, IsRead: false})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{corp.LabelID}, IsRead: true})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{corp.LabelID}, IsRead: false})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{corp.LabelID}, IsRead: false})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID})
		l1 := factory.CreateCharacterMailList(c.ID)
		factory.CreateCharacterMailList(c.ID)
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int32{l1.ID},
			IsRead:       false,
		})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int32{l1.ID},
			IsRead:       true,
		})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID})
		// when
		r, err := r.GetCharacterMailUnreadCount(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, 6, r)
		}
	})
	t.Run("should return null when no mail exists", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		r, err := r.GetCharacterMailUnreadCount(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, 0, r)
		}
	})
}
