package storage_test

import (
	"context"
	"example/evebuddy/internal/helper/set"
	"example/evebuddy/internal/model"
	"example/evebuddy/internal/storage"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMailCreate(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		c := factory.CreateCharacter()
		f := factory.CreateEveEntity()
		recipient := factory.CreateEveEntity()
		label := factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID})
		// when
		arg := storage.CreateMailParams{
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
		_, err := r.CreateMail(ctx, arg)
		// then
		if assert.NoError(t, err) {
			m, err := r.GetMail(ctx, c.ID, 42)
			assert.NoError(t, err)
			assert.Equal(t, int32(42), m.MailID)
			assert.Equal(t, "body", m.Body)
			assert.Equal(t, f, m.From)
			assert.Equal(t, c.ID, m.CharacterID)
			assert.Equal(t, "subject", m.Subject)
			assert.False(t, m.Timestamp.IsZero())
			assert.Equal(t, []model.EveEntity{recipient}, m.Recipients)
			assert.Equal(t, label.Name, m.Labels[0].Name)
			assert.Equal(t, label.LabelID, m.Labels[0].LabelID)
		}
	})
}

func TestMail(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		_, err := r.GetMail(ctx, c.ID, 99)
		// then
		assert.ErrorIs(t, err, storage.ErrNotFound)
	})
	t.Run("can list mail IDs", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		c := factory.CreateCharacter()
		for i := range 3 {
			factory.CreateMail(storage.CreateMailParams{
				CharacterID: c.ID,
				MailID:      int32(10 + i),
			})
		}
		// when
		ids, err := r.ListMailIDs(ctx, c.ID)
		// then
		assert.NoError(t, err)
		got := set.NewFromSlice(ids)
		want := set.NewFromSlice([]int32{10, 11, 12})
		assert.Equal(t, want, got)
	})
	t.Run("can delete existing mail", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		m := factory.CreateMail()
		// when
		err := r.DeleteMail(ctx, m.CharacterID, m.MailID)
		// then
		if assert.NoError(t, err) {
			_, err := r.GetMail(ctx, m.CharacterID, m.MailID)
			assert.ErrorIs(t, err, storage.ErrNotFound)
		}
	})
}

func TestListMailID(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return mail for selected label only", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID})
		l2 := factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID})
		m1 := factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID, LabelIDs: []int32{l1.LabelID}, Timestamp: time.Now().Add(time.Second * -120)})
		m2 := factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID, LabelIDs: []int32{l1.LabelID}, Timestamp: time.Now().Add(time.Second * -60)})
		factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID, LabelIDs: []int32{l2.LabelID}})
		// when
		got, err := r.ListMailIDsForLabelOrdered(ctx, c.ID, l1.LabelID)
		// then
		if assert.NoError(t, err) {
			want := []int32{m2.MailID, m1.MailID}
			assert.Equal(t, want, got)
		}
	})
	t.Run("can fetch for all labels", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID})
		l2 := factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID})
		m1 := factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID, LabelIDs: []int32{l1.LabelID}, Timestamp: time.Now().Add(time.Second * -120)})
		m2 := factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID, LabelIDs: []int32{l1.LabelID}, Timestamp: time.Now().Add(time.Second * -60)})
		m3 := factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID, LabelIDs: []int32{l2.LabelID}, Timestamp: time.Now().Add(time.Second * -240)})
		m4 := factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID, Timestamp: time.Now().Add(time.Second * -360)})
		// when
		got, err := r.ListMailIDsForLabelOrdered(ctx, c.ID, model.MailLabelAll)
		// then
		if assert.NoError(t, err) {
			want := []int32{m2.MailID, m1.MailID, m3.MailID, m4.MailID}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return mail without label only", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		c := factory.CreateCharacter()
		l := factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID})
		factory.CreateMail(storage.CreateMailParams{
			CharacterID: c.ID,
			LabelIDs:    []int32{l.LabelID},
			Timestamp:   time.Now().Add(time.Second * -120),
		})
		m := factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID})
		// when
		got, err := r.ListMailIDsForLabelOrdered(ctx, c.ID, model.MailLabelNone)
		// then
		if assert.NoError(t, err) {
			want := []int32{m.MailID}
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return empty when no match", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		mm, err := r.ListMailIDsForLabelOrdered(ctx, c.ID, 99)
		// then
		if assert.NoError(t, err) {
			assert.Empty(t, mm)
		}
	})
	t.Run("different characters can have same label ID", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		c1 := factory.CreateCharacter()
		l1 := factory.CreateMailLabel(model.MailLabel{CharacterID: c1.ID, LabelID: 1})
		factory.CreateMail(storage.CreateMailParams{
			CharacterID: c1.ID,
			LabelIDs:    []int32{l1.LabelID},
		})
		c2 := factory.CreateCharacter()
		l2 := factory.CreateMailLabel(model.MailLabel{CharacterID: c2.ID, LabelID: 1})
		from := factory.CreateEveEntity()
		factory.CreateMail(storage.CreateMailParams{
			FromID:      from.ID,
			CharacterID: c2.ID,
			LabelIDs:    []int32{l2.LabelID},
		})
		// when
		mm, err := r.ListMailIDsForLabelOrdered(ctx, c2.ID, l2.LabelID)
		if assert.NoError(t, err) {
			assert.Len(t, mm, 1)
		}
	})
	t.Run("should return mail for selected list only", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateMailList(c.ID)
		m1 := factory.CreateMail(storage.CreateMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int32{l1.ID},
		})
		l2 := factory.CreateMailList(c.ID)
		factory.CreateMail(storage.CreateMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int32{l2.ID},
		})
		factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID})
		// when
		got, err := r.ListMailIDsForListOrdered(ctx, c.ID, l1.ID)
		// then
		if assert.NoError(t, err) {
			want := []int32{m1.MailID}
			assert.Equal(t, want, got)
		}
	})
}

func TestFetchUnreadCounts(t *testing.T) {
	db, r, factory := setUpDB()
	defer db.Close()
	ctx := context.Background()
	t.Run("can get mail label unread counts", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		c := factory.CreateCharacter()
		corp := factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID, LabelID: model.MailLabelCorp})
		inbox := factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID, LabelID: model.MailLabelInbox})
		factory.CreateMailLabel(model.MailLabel{CharacterID: c.ID, LabelID: model.MailLabelAlliance})
		factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID, LabelIDs: []int32{inbox.LabelID}, IsRead: false})
		factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID, LabelIDs: []int32{corp.LabelID}, IsRead: true})
		factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID, LabelIDs: []int32{corp.LabelID}, IsRead: false})
		factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID, LabelIDs: []int32{corp.LabelID}, IsRead: false})
		factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID})
		// when
		r, err := r.GetMailLabelUnreadCounts(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, map[int32]int{model.MailLabelCorp: 2, model.MailLabelInbox: 1}, r)
		}
	})
	t.Run("can get mail list unread counts", func(t *testing.T) {
		// given
		storage.TruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateMailList(c.ID)
		factory.CreateMailList(c.ID)
		factory.CreateMail(storage.CreateMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int32{l1.ID},
			IsRead:       false,
		})
		factory.CreateMail(storage.CreateMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int32{l1.ID},
			IsRead:       true,
		})
		factory.CreateMail(storage.CreateMailParams{CharacterID: c.ID})
		// when
		r, err := r.GetMailListUnreadCounts(ctx, c.ID)
		if assert.NoError(t, err) {
			assert.Equal(t, map[int32]int{l1.ID: 1}, r)
		}
	})
}
