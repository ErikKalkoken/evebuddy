package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/helper/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"github.com/ErikKalkoken/evebuddy/internal/storage"
	"github.com/stretchr/testify/assert"
)

func TestListMailHeaders(t *testing.T) {
	db, r, factory := testutil.New()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return mail for selected label only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c.ID})
		l2 := factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c.ID})
		m1 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{l1.LabelID}, Timestamp: time.Now().Add(time.Second * -120)})
		m2 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{l1.LabelID}, Timestamp: time.Now().Add(time.Second * -60)})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{l2.LabelID}})
		// when
		xx, err := r.ListCharacterMailHeadersForLabelOrdered(ctx, c.ID, l1.LabelID)
		// then
		if assert.NoError(t, err) {
			assert.Len(t, xx, 2)
			want := []int32{m2.MailID, m1.MailID}
			got := mailIDsFromHeaders(xx)
			assert.Equal(t, want, got)
		}
	})
	t.Run("can fetch for all labels", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c.ID})
		l2 := factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c.ID})
		m1 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{l1.LabelID}, Timestamp: time.Now().Add(time.Second * -120)})
		m2 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{l1.LabelID}, Timestamp: time.Now().Add(time.Second * -60)})
		m3 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{l2.LabelID}, Timestamp: time.Now().Add(time.Second * -240)})
		m4 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, Timestamp: time.Now().Add(time.Second * -360)})
		// when
		xx, err := r.ListCharacterMailHeadersForLabelOrdered(ctx, c.ID, model.MailLabelAll)
		// then
		if assert.NoError(t, err) {
			want := []int32{m2.MailID, m1.MailID, m3.MailID, m4.MailID}
			got := mailIDsFromHeaders(xx)
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return mail without label only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		l := factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c.ID})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			LabelIDs:    []int32{l.LabelID},
			Timestamp:   time.Now().Add(time.Second * -120),
		})
		m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID})
		// when
		xx, err := r.ListCharacterMailHeadersForLabelOrdered(ctx, c.ID, model.MailLabelNone)
		// then
		if assert.NoError(t, err) {
			want := []int32{m.MailID}
			got := mailIDsFromHeaders(xx)
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return empty when no match", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		// when
		mm, err := r.ListCharacterMailHeadersForLabelOrdered(ctx, c.ID, 99)
		// then
		if assert.NoError(t, err) {
			assert.Empty(t, mm)
		}
	})
	t.Run("different characters can have same label ID", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c1 := factory.CreateCharacter()
		l1 := factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c1.ID, LabelID: 1})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c1.ID,
			LabelIDs:    []int32{l1.LabelID},
		})
		c2 := factory.CreateCharacter()
		l2 := factory.CreateCharacterMailLabel(model.CharacterMailLabel{CharacterID: c2.ID, LabelID: 1})
		from := factory.CreateEveEntity()
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			FromID:      from.ID,
			CharacterID: c2.ID,
			LabelIDs:    []int32{l2.LabelID},
		})
		// when
		mm, err := r.ListCharacterMailHeadersForLabelOrdered(ctx, c2.ID, l2.LabelID)
		if assert.NoError(t, err) {
			assert.Len(t, mm, 1)
		}
	})
	t.Run("should return mail for selected list only", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateCharacterMailList(c.ID)
		m1 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int32{l1.ID},
		})
		l2 := factory.CreateCharacterMailList(c.ID)
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int32{l2.ID},
		})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID})
		// when
		xx, err := r.ListCharacterMailHeadersForListOrdered(ctx, c.ID, l1.ID)
		// then
		if assert.NoError(t, err) {
			want := []int32{m1.MailID}
			got := mailIDsFromHeaders(xx)
			assert.Equal(t, want, got)
		}
	})
}

func mailIDsFromHeaders(hh []*model.CharacterMailHeader) []int32 {
	ids := make([]int32, len(hh))
	for i, h := range hh {
		ids[i] = h.MailID
	}
	return ids
}
