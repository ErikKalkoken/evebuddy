package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestListMailHeaders(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return mail for selected label only", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		l1 := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID})
		l2 := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID})
		m1 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{l1.LabelID}, Timestamp: time.Now().Add(time.Second * -120)})
		m2 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{l1.LabelID}, Timestamp: time.Now().Add(time.Second * -60)})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{l2.LabelID}})
		// when
		xx, err := st.ListCharacterMailHeadersForLabelOrdered(ctx, c.ID, l1.LabelID)
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
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		l1 := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID})
		l2 := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID})
		m1 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{l1.LabelID}, Timestamp: time.Now().Add(time.Second * -120)})
		m2 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{l1.LabelID}, Timestamp: time.Now().Add(time.Second * -60)})
		m3 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, LabelIDs: []int32{l2.LabelID}, Timestamp: time.Now().Add(time.Second * -240)})
		m4 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID, Timestamp: time.Now().Add(time.Second * -360)})
		// when
		xx, err := st.ListCharacterMailHeadersForLabelOrdered(ctx, c.ID, app.MailLabelAll)
		// then
		if assert.NoError(t, err) {
			want := []int32{m2.MailID, m1.MailID, m3.MailID, m4.MailID}
			got := mailIDsFromHeaders(xx)
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return mail without label only", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		l := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			LabelIDs:    []int32{l.LabelID},
			Timestamp:   time.Now().Add(time.Second * -120),
		})
		m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID})
		// when
		xx, err := st.ListCharacterMailHeadersForLabelOrdered(ctx, c.ID, app.MailLabelNone)
		// then
		if assert.NoError(t, err) {
			want := []int32{m.MailID}
			got := mailIDsFromHeaders(xx)
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return empty when no match", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
		// when
		mm, err := st.ListCharacterMailHeadersForLabelOrdered(ctx, c.ID, 99)
		// then
		if assert.NoError(t, err) {
			assert.Empty(t, mm)
		}
	})
	t.Run("different characters can have same label ID", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacterFull()
		l1 := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c1.ID, LabelID: 1})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c1.ID,
			LabelIDs:    []int32{l1.LabelID},
		})
		c2 := factory.CreateCharacterFull()
		l2 := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c2.ID, LabelID: 1})
		from := factory.CreateEveEntity()
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			FromID:      from.ID,
			CharacterID: c2.ID,
			LabelIDs:    []int32{l2.LabelID},
		})
		// when
		mm, err := st.ListCharacterMailHeadersForLabelOrdered(ctx, c2.ID, l2.LabelID)
		if assert.NoError(t, err) {
			assert.Len(t, mm, 1)
		}
	})
	t.Run("should return mail for selected list only", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacterFull()
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
		xx, err := st.ListCharacterMailHeadersForListOrdered(ctx, c.ID, l1.ID)
		// then
		if assert.NoError(t, err) {
			want := []int32{m1.MailID}
			got := mailIDsFromHeaders(xx)
			assert.Equal(t, want, got)
		}
	})
	t.Run("should return unprocessed mails only and ignore sent mails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		now := time.Now().UTC()
		c := factory.CreateCharacterFull()
		m1 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsProcessed: false,
		})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsProcessed: true,
		})
		l := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: app.MailLabelSent})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsProcessed: false,
			LabelIDs:    []int32{l.LabelID},
		})
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsProcessed: false,
			Timestamp:   now.Add(-10 * time.Hour),
		})
		factory.CreateCharacterMail()
		// when
		xx, err := st.ListCharacterMailHeadersForUnprocessed(ctx, c.ID, now.Add(-5*time.Hour))
		// then
		if assert.NoError(t, err) {
			want := []int32{m1.MailID}
			got := mailIDsFromHeaders(xx)
			assert.Equal(t, want, got)
		}
	})
}

func mailIDsFromHeaders(hh []*app.CharacterMailHeader) []int32 {
	ids := make([]int32, len(hh))
	for i, h := range hh {
		ids[i] = h.MailID
	}
	return ids
}
