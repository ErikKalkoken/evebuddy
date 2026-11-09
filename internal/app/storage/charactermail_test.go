package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestCharacterMail(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can create new", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		f := factory.CreateEveEntity()
		recipient := factory.CreateEveEntity()
		label := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID})
		// when
		timestamp := time.Now()
		arg := storage.CreateCharacterMailParams{
			Body:         optional.New("body"),
			CharacterID:  c.ID,
			FromID:       f.ID,
			IsRead:       optional.New(false),
			LabelIDs:     []int64{label.LabelID},
			MailID:       42,
			RecipientIDs: []int64{recipient.ID},
			Subject:      optional.New("subject"),
			Timestamp:    timestamp,
		}
		_, err := st.CreateCharacterMail(ctx, arg)
		// then
		require.NoError(t, err)
		m, err := st.GetCharacterMail(ctx, c.ID, 42)
		assert.NoError(t, err)
		xassert.Equal(t, 42, m.MailID)
		xassert.EqualOptional(t, "body", m.Body)
		xassert.Equal(t, f, m.From)
		xassert.Equal(t, c.ID, m.CharacterID)
		xassert.EqualOptional(t, "subject", m.Subject)
		xassert.Equal(t, timestamp, m.Timestamp)
		xassert.Equal(t, []*app.EveEntity{recipient}, m.Recipients)
		xassert.Equal(t, label.Name, m.Labels[0].Name)
		xassert.Equal(t, label.LabelID, m.Labels[0].LabelID)
	})
	t.Run("can create mail with label that is not stored yet", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		f := factory.CreateEveEntity()
		recipient := factory.CreateEveEntity()
		// when
		_, err := st.CreateCharacterMail(ctx, storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			FromID:       f.ID,
			LabelIDs:     []int64{256},
			MailID:       42,
			RecipientIDs: []int64{recipient.ID},
			Timestamp:    time.Now(),
		})
		// then
		require.NoError(t, err)
		m, err := st.GetCharacterMail(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, []int64{256}, m.LabelIDs())
		l, err := st.GetCharacterMailLabel(ctx, c.ID, 256)
		require.NoError(t, err)
		xassert.EqualOptional(t, "Label 256", l.Name)
	})
	t.Run("should not create mail when creating fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		f := factory.CreateEveEntity()
		// when
		_, err := st.CreateCharacterMail(ctx, storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			FromID:       f.ID,
			MailID:       42,
			RecipientIDs: []int64{999_999}, // unknown recipient
			Timestamp:    time.Now(),
		})
		// then
		require.Error(t, err)
		_, err = st.GetCharacterMail(ctx, c.ID, 42)
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("can update is-read", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		m := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{})
		// when
		err := st.UpdateCharacterMailSetIsRead(ctx, m.CharacterID, m.ID, true)
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterMail(ctx, m.CharacterID, m.MailID)
		assert.NoError(t, err)
		assert.True(t, got.IsRead.ValueOrZero())
	})
	t.Run("can update body", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		m := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{})
		// when
		err := st.UpdateCharacterMailSetBody(ctx, m.CharacterID, m.MailID, optional.New("alpha"))
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterMail(ctx, m.CharacterID, m.MailID)
		assert.NoError(t, err)
		xassert.EqualOptional(t, "alpha", got.Body)
	})
	t.Run("can set processed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		m := factory.CreateCharacterMailWithBody()
		// when
		err := st.UpdateCharacterMailSetProcessed(ctx, m.ID)
		// then
		require.NoError(t, err)
		o, err := st.GetCharacterMail(ctx, m.CharacterID, m.MailID)
		require.NoError(t, err)
		assert.True(t, o.IsProcessed)
	})
	t.Run("should return correct error when not found", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		// when
		_, err := st.GetCharacterMail(ctx, c.ID, 99)
		// then
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("can list mail IDs", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		for i := range 3 {
			factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
				CharacterID: c.ID,
				MailID:      int64(10 + i),
			})
		}
		// when
		got, err := st.ListCharacterMailIDs(ctx, c.ID)
		// then
		assert.NoError(t, err)
		want := set.Of([]int64{10, 11, 12}...)
		xassert.Equal(t, want, got)
	})
	t.Run("can list IDs for mails withtout bodies", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		mailIDs := set.Of[int64](10, 11, 12)
		for i := range mailIDs.All() {
			factory.CreateCharacterMail(storage.CreateCharacterMailParams{
				CharacterID: c.ID,
				MailID:      i,
			})
		}
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
		})
		factory.CreateCharacterMail()
		// when
		got, err := st.ListCharacterMailsWithoutBody(ctx, c.ID)
		// then
		assert.NoError(t, err)
		xassert.Equal(t, mailIDs, got)
	})
	t.Run("can delete existing mail", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		m := factory.CreateCharacterMailWithBody()
		// when
		err := st.DeleteCharacterMail(ctx, m.CharacterID, m.MailID)
		// then
		require.NoError(t, err)
		_, err = st.GetCharacterMail(ctx, m.CharacterID, m.MailID)
		assert.ErrorIs(t, err, app.ErrNotFound)
	})
}

func TestFetchUnreadCounts(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can get mail label unread counts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		corp := factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: c.ID,
			LabelID:     app.MailLabelCorp,
		})
		inbox := factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: c.ID,
			LabelID:     app.MailLabelInbox,
		})
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: c.ID,
			LabelID:     app.MailLabelAlliance,
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{inbox.LabelID},
			IsRead:      optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{corp.LabelID},
			IsRead:      optional.New(true),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{corp.LabelID},
			IsRead:      optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{corp.LabelID},
			IsRead:      optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
		})
		// when
		r, err := st.CountCharacterMailsUnreadByLabel(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, map[int64]int{app.MailLabelCorp: 2, app.MailLabelInbox: 1}, r)
	})
	t.Run("can get mail list unread counts", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		l1 := factory.CreateCharacterMailList(c.ID)
		factory.CreateCharacterMailList(c.ID)
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int64{l1.ID},
			IsRead:       optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int64{l1.ID},
			IsRead:       optional.New(true),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: c.ID})
		// when
		r, err := st.CountCharacterMailsUnreadByList(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, map[int64]int{l1.ID: 1}, r)
	})

}

func TestUnreadMailCounts(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should return correct unread count when mails exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		corp := factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: c.ID,
			LabelID:     app.MailLabelCorp,
		})
		inbox := factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: c.ID,
			LabelID:     app.MailLabelInbox,
		})
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: c.ID,
			LabelID:     app.MailLabelAlliance,
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{inbox.LabelID},
			IsRead:      optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{corp.LabelID},
			IsRead:      optional.New(true),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{corp.LabelID},
			IsRead:      optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{corp.LabelID},
			IsRead:      optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
		})
		l1 := factory.CreateCharacterMailList(c.ID)
		factory.CreateCharacterMailList(c.ID)
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int64{l1.ID},
			IsRead:       optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int64{l1.ID},
			IsRead:       optional.New(true),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: c.ID})
		// when
		r, err := st.CountCharacterMailsUnread(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 6, r)
	})
	t.Run("should return null when no mail exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		// when
		r, err := st.CountCharacterMailsUnread(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0, r)
	})
	t.Run("unread count for all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character1 := factory.CreateCharacter()
		corp := factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: character1.ID,
			LabelID:     app.MailLabelCorp,
		})
		inbox := factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: character1.ID,
			LabelID:     app.MailLabelInbox,
		})
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: character1.ID,
			LabelID:     app.MailLabelAlliance,
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: character1.ID,
			LabelIDs:    []int64{inbox.LabelID},
			IsRead:      optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: character1.ID,
			LabelIDs:    []int64{corp.LabelID},
			IsRead:      optional.New(true),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: character1.ID,
			LabelIDs:    []int64{corp.LabelID},
			IsRead:      optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: character1.ID,
			LabelIDs:    []int64{corp.LabelID},
			IsRead:      optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: character1.ID,
		})
		l1 := factory.CreateCharacterMailList(character1.ID)
		factory.CreateCharacterMailList(character1.ID)
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID:  character1.ID,
			RecipientIDs: []int64{l1.ID},
			IsRead:       optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID:  character1.ID,
			RecipientIDs: []int64{l1.ID},
			IsRead:       optional.New(true),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: character1.ID,
		})
		character2 := factory.CreateCharacter()
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: character2.ID,
			IsRead:      optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: character2.ID,
			IsRead:      optional.New(true),
		})
		// when
		got, err := st.CountAllCharactersMailsUnread(ctx)
		require.NoError(t, err)
		xassert.Equal(t, 7, got)
	})
}

func TestMailCounts(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("character has mail", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: character.ID,
			IsRead:      optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: character.ID,
			IsRead:      optional.New(true),
		})
		factory.CreateCharacterMailWithBody()
		// when
		got, err := st.CountCharacterMails(ctx, character.ID)
		require.NoError(t, err)
		xassert.Equal(t, 2, got)
	})
	t.Run("character has no mail", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		factory.CreateCharacterMailWithBody()
		// when
		got, err := st.CountCharacterMails(ctx, character.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0, got)
	})
}

func TestCountCharacterMailsWithoutBody(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("counts only mails without body", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: character.ID})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: character.ID})
		factory.CreateCharacterMail() // other character
		// when
		got, err := st.CountCharacterMailsWithoutBody(ctx, character.ID)
		require.NoError(t, err)
		xassert.Equal(t, 1, got)
	})
	t.Run("character has no mail", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		factory.CreateCharacterMail() // other character
		// when
		got, err := st.CountCharacterMailsWithoutBody(ctx, character.ID)
		require.NoError(t, err)
		xassert.Equal(t, 0, got)
	})
}

func TestAllCharactersMails(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("can list mail headers for a label of all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacterFull()
		l1 := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c1.ID, LabelID: app.MailLabelInbox})
		m1 := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c1.ID,
			LabelIDs:    []int64{l1.LabelID},
			Timestamp:   time.Now().Add(-120 * time.Second),
		})
		c2 := factory.CreateCharacterFull()
		l2 := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c2.ID, LabelID: app.MailLabelInbox})
		m2 := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c2.ID,
			LabelIDs:    []int64{l2.LabelID},
			Timestamp:   time.Now().Add(-60 * time.Second),
		})
		l3 := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c2.ID, LabelID: app.MailLabelSent})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c2.ID,
			LabelIDs:    []int64{l3.LabelID},
		})
		// when
		xx, err := st.ListAllCharacterMailHeadersForLabelOrdered(ctx, app.MailLabelInbox)
		// then
		require.NoError(t, err)
		xassert.Equal(t, []int64{m2.MailID, m1.MailID}, mailIDsFromHeaders(xx))
		xassert.Equal(t, []int64{c2.ID, c1.ID}, []int64{xx[0].CharacterID, xx[1].CharacterID})
	})
	t.Run("can list mail headers for all labels of all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		m1 := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			Timestamp: time.Now().Add(-120 * time.Second),
		})
		m2 := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			Timestamp: time.Now().Add(-60 * time.Second),
		})
		// when
		xx, err := st.ListAllCharacterMailHeadersForLabelOrdered(ctx, app.MailLabelAll)
		// then
		require.NoError(t, err)
		xassert.Equal(t, []int64{m2.MailID, m1.MailID}, mailIDsFromHeaders(xx))
		xassert.Equal(t, []int64{m2.CharacterID, m1.CharacterID}, []int64{xx[0].CharacterID, xx[1].CharacterID})
	})
	t.Run("can list mail headers for a mailing list of all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacterFull()
		l := factory.CreateCharacterMailList(c1.ID)
		m1 := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID:  c1.ID,
			RecipientIDs: []int64{l.ID},
		})
		c2 := factory.CreateCharacterFull()
		assert.NoError(t, st.CreateCharacterMailList(ctx, c2.ID, l.ID))
		m2 := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID:  c2.ID,
			RecipientIDs: []int64{l.ID},
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: c2.ID})
		// when
		xx, err := st.ListAllCharacterMailHeadersForListOrdered(ctx, l.ID)
		// then
		require.NoError(t, err)
		xassert.Equal(t, []int64{m1.MailID, m2.MailID}, mailIDsFromHeaders(xx))
	})
	t.Run("can list mailing lists of all characters without duplicates", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacterFull()
		c2 := factory.CreateCharacterFull()
		e1 := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList, Name: "alpha"})
		assert.NoError(t, st.CreateCharacterMailList(ctx, c1.ID, e1.ID))
		assert.NoError(t, st.CreateCharacterMailList(ctx, c2.ID, e1.ID))
		e2 := factory.CreateEveEntity(app.EveEntity{Category: app.EveEntityMailList, Name: "bravo"})
		assert.NoError(t, st.CreateCharacterMailList(ctx, c2.ID, e2.ID))
		// when
		ll, err := st.ListAllCharacterMailListsOrdered(ctx)
		// then
		require.NoError(t, err)
		xassert.Equal(t, []int64{e1.ID, e2.ID}, []int64{ll[0].ID, ll[1].ID})
		assert.Len(t, ll, 2)
	})
	t.Run("can get label unread counts of all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		for range 2 {
			c := factory.CreateCharacter()
			inbox := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: app.MailLabelInbox})
			factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
				CharacterID: c.ID,
				LabelIDs:    []int64{inbox.LabelID},
				IsRead:      optional.New(false),
			})
			factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
				CharacterID: c.ID,
				LabelIDs:    []int64{inbox.LabelID},
				IsRead:      optional.New(true),
			})
		}
		// when
		got, err := st.CountAllCharactersMailsUnreadByLabel(ctx)
		// then
		require.NoError(t, err)
		xassert.Equal(t, map[int64]int{app.MailLabelInbox: 2}, got)
	})
	t.Run("can get list unread counts of all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacter()
		l := factory.CreateCharacterMailList(c1.ID)
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID:  c1.ID,
			RecipientIDs: []int64{l.ID},
			IsRead:       optional.New(false),
		})
		c2 := factory.CreateCharacter()
		assert.NoError(t, st.CreateCharacterMailList(ctx, c2.ID, l.ID))
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID:  c2.ID,
			RecipientIDs: []int64{l.ID},
			IsRead:       optional.New(false),
		})
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID:  c2.ID,
			RecipientIDs: []int64{l.ID},
			IsRead:       optional.New(true),
		})
		// when
		got, err := st.CountAllCharactersMailsUnreadByList(ctx)
		// then
		require.NoError(t, err)
		xassert.Equal(t, map[int64]int{l.ID: 2}, got)
	})
	t.Run("can count mails and mails without body of all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateCharacterMailWithBody()
		factory.CreateCharacterMailWithBody()
		factory.CreateCharacterMail()
		// when
		total, err1 := st.CountAllCharactersMails(ctx)
		missing, err2 := st.CountAllCharactersMailsWithoutBody(ctx)
		// then
		require.NoError(t, err1)
		require.NoError(t, err2)
		xassert.Equal(t, 3, total)
		xassert.Equal(t, 1, missing)
	})
}

func TestStorage_UpdateOrCreateCharacterMails(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	ctx := context.Background()
	t.Run("should create new mails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		from := factory.CreateEveEntity()
		recipient := factory.CreateEveEntity()
		label := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID})
		timestamp := time.Now().UTC()
		// when
		changed, err := st.UpdateOrCreateCharacterMails(ctx, c.ID, []storage.UpdateOrCreateCharacterMailParams{
			{
				FromID:       from.ID,
				IsRead:       optional.New(true),
				LabelIDs:     []int64{label.LabelID, 256},
				MailID:       42,
				RecipientIDs: []int64{recipient.ID},
				Subject:      optional.New("subject"),
				Timestamp:    timestamp,
			},
		})
		// then
		require.NoError(t, err)
		assert.True(t, changed)
		m, err := st.GetCharacterMail(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, from, m.From)
		xassert.EqualOptional(t, true, m.IsRead)
		assert.ElementsMatch(t, []int64{label.LabelID, 256}, m.LabelIDs())
		xassert.Equal(t, []*app.EveEntity{recipient}, m.Recipients)
		xassert.EqualOptional(t, "subject", m.Subject)
		xassert.Equal(t, timestamp, m.Timestamp)
		assert.True(t, m.Body.IsEmpty())
		l, err := st.GetCharacterMailLabel(ctx, c.ID, 256)
		require.NoError(t, err)
		xassert.EqualOptional(t, "Label 256", l.Name)
	})
	t.Run("should update read state of stored mails", func(t *testing.T) {
		cases := []struct {
			name    string
			stored  bool
			isRead  optional.Optional[bool]
			want    bool
			changed bool
		}{
			{"mark as read", false, optional.New(true), true, true},
			{"mark as unread", true, optional.New(false), false, true},
			{"keep read", true, optional.New(true), true, false},
			{"mark as unread when missing", true, optional.Optional[bool]{}, false, true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// given
				testutil.MustTruncateTables(db)
				m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
					IsRead: optional.New(tc.stored),
				})
				// when
				changed, err := st.UpdateOrCreateCharacterMails(ctx, m.CharacterID, []storage.UpdateOrCreateCharacterMailParams{
					{
						FromID:       m.From.ID,
						IsRead:       tc.isRead,
						MailID:       m.MailID,
						RecipientIDs: []int64{m.Recipients[0].ID},
						Timestamp:    m.Timestamp,
					},
				})
				// then
				require.NoError(t, err)
				got, err := st.GetCharacterMail(ctx, m.CharacterID, m.MailID)
				require.NoError(t, err)
				xassert.Equal(t, tc.want, got.IsRead.ValueOrZero())
				xassert.Equal(t, tc.changed, changed)
			})
		}
	})
	t.Run("should update labels of stored mails", func(t *testing.T) {
		cases := []struct {
			name    string
			stored  []int64
			labels  []int64
			changed bool
		}{
			{"add label", []int64{1}, []int64{1, 2}, true},
			{"replace label", []int64{1}, []int64{2}, true},
			{"remove all labels", []int64{1, 2}, nil, true},
			{"add labels to mail without labels", nil, []int64{1, 2}, true},
			{"keep labels", []int64{1, 2}, []int64{2, 1}, false},
			{"add label not stored yet", []int64{1}, []int64{1, 256}, true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// given
				testutil.MustTruncateTables(db)
				c := factory.CreateCharacter()
				factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: 1})
				factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: 2})
				m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
					CharacterID: c.ID,
					LabelIDs:    tc.stored,
				})
				// when
				changed, err := st.UpdateOrCreateCharacterMails(ctx, c.ID, []storage.UpdateOrCreateCharacterMailParams{
					{
						FromID:       m.From.ID,
						LabelIDs:     tc.labels,
						MailID:       m.MailID,
						RecipientIDs: []int64{m.Recipients[0].ID},
						Timestamp:    m.Timestamp,
					},
				})
				// then
				require.NoError(t, err)
				got, err := st.GetCharacterMail(ctx, c.ID, m.MailID)
				require.NoError(t, err)
				assert.ElementsMatch(t, tc.labels, got.LabelIDs())
				xassert.Equal(t, tc.changed, changed)
			})
		}
	})
	t.Run("should store duplicate label IDs of a new mail once", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		from := factory.CreateEveEntity()
		recipient := factory.CreateEveEntity()
		// when
		_, err := st.UpdateOrCreateCharacterMails(ctx, c.ID, []storage.UpdateOrCreateCharacterMailParams{
			{
				FromID:       from.ID,
				LabelIDs:     []int64{1, 1},
				MailID:       42,
				RecipientIDs: []int64{recipient.ID},
				Timestamp:    time.Now(),
			},
		})
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterMail(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, []int64{1}, got.LabelIDs())
	})
	t.Run("should store duplicate recipient IDs of a new mail once", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		from := factory.CreateEveEntity()
		recipient := factory.CreateEveEntity()
		// when
		_, err := st.UpdateOrCreateCharacterMails(ctx, c.ID, []storage.UpdateOrCreateCharacterMailParams{
			{
				FromID:       from.ID,
				MailID:       42,
				RecipientIDs: []int64{recipient.ID, recipient.ID},
				Timestamp:    time.Now(),
			},
		})
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterMail(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.Equal(t, []*app.EveEntity{recipient}, got.Recipients)
	})
	t.Run("should store duplicate label IDs of a stored mail once", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{1},
		})
		// when
		_, err := st.UpdateOrCreateCharacterMails(ctx, c.ID, []storage.UpdateOrCreateCharacterMailParams{
			{
				FromID:       m.From.ID,
				LabelIDs:     []int64{2, 2},
				MailID:       m.MailID,
				RecipientIDs: []int64{m.Recipients[0].ID},
				Timestamp:    m.Timestamp,
			},
		})
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterMail(ctx, c.ID, m.MailID)
		require.NoError(t, err)
		xassert.Equal(t, []int64{2}, got.LabelIDs())
	})
	t.Run("should keep stored mails missing from args", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: c.ID})
		// when
		_, err := st.UpdateOrCreateCharacterMails(ctx, c.ID, nil)
		// then
		require.NoError(t, err)
		got, err := st.ListCharacterMailIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of(m.MailID), got)
	})
	t.Run("should store duplicate args once with the last read state and labels", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		from := factory.CreateEveEntity()
		recipient := factory.CreateEveEntity()
		// when
		_, err := st.UpdateOrCreateCharacterMails(ctx, c.ID, []storage.UpdateOrCreateCharacterMailParams{
			{
				FromID:       from.ID,
				IsRead:       optional.New(false),
				LabelIDs:     []int64{1},
				MailID:       42,
				RecipientIDs: []int64{recipient.ID},
				Timestamp:    time.Now(),
			},
			{
				FromID:       from.ID,
				IsRead:       optional.New(true),
				LabelIDs:     []int64{2},
				MailID:       42,
				RecipientIDs: []int64{recipient.ID},
				Timestamp:    time.Now(),
			},
		})
		// then
		require.NoError(t, err)
		got, err := st.GetCharacterMail(ctx, c.ID, 42)
		require.NoError(t, err)
		xassert.EqualOptional(t, true, got.IsRead)
		xassert.Equal(t, []int64{2}, got.LabelIDs())
	})
	t.Run("should leave mails of other characters untouched", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c1 := factory.CreateCharacter()
		c2 := factory.CreateCharacter()
		m2 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c2.ID,
			IsRead:      optional.New(false),
			MailID:      42,
		})
		// when
		_, err := st.UpdateOrCreateCharacterMails(ctx, c1.ID, []storage.UpdateOrCreateCharacterMailParams{
			{
				FromID:       m2.From.ID,
				IsRead:       optional.New(true),
				MailID:       42,
				RecipientIDs: []int64{m2.Recipients[0].ID},
				Timestamp:    m2.Timestamp,
			},
		})
		// then
		require.NoError(t, err)
		got1, err := st.GetCharacterMail(ctx, c1.ID, 42)
		require.NoError(t, err)
		xassert.EqualOptional(t, true, got1.IsRead)
		got2, err := st.GetCharacterMail(ctx, c2.ID, 42)
		require.NoError(t, err)
		assert.False(t, got2.IsRead.ValueOrZero())
	})
	t.Run("should store nothing when a new mail has no mail ID", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: 1})
		m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsRead:      optional.New(false),
			LabelIDs:    []int64{1},
			MailID:      42,
		})
		// when
		_, err := st.UpdateOrCreateCharacterMails(ctx, c.ID, []storage.UpdateOrCreateCharacterMailParams{
			{
				FromID:       m.From.ID,
				MailID:       44,
				RecipientIDs: []int64{m.Recipients[0].ID},
				Timestamp:    time.Now(),
			},
			{
				FromID:       m.From.ID,
				RecipientIDs: []int64{m.Recipients[0].ID},
				Timestamp:    time.Now(),
			},
		})
		// then
		require.ErrorIs(t, err, app.ErrInvalid)
		ids, err := st.ListCharacterMailIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](42), ids)
		got, err := st.GetCharacterMail(ctx, c.ID, 42)
		require.NoError(t, err)
		assert.False(t, got.IsRead.ValueOrZero())
		xassert.Equal(t, []int64{1}, got.LabelIDs())
	})
	t.Run("should store nothing when a new mail has no recipients", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: 1})
		m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsRead:      optional.New(false),
			LabelIDs:    []int64{1},
			MailID:      42,
		})
		// when
		_, err := st.UpdateOrCreateCharacterMails(ctx, c.ID, []storage.UpdateOrCreateCharacterMailParams{
			{
				FromID:       m.From.ID,
				MailID:       44,
				RecipientIDs: []int64{m.Recipients[0].ID},
				Timestamp:    time.Now(),
			},
			{
				FromID:    m.From.ID,
				MailID:    43,
				Timestamp: time.Now(),
			},
		})
		// then
		require.ErrorIs(t, err, app.ErrInvalid)
		ids, err := st.ListCharacterMailIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](42), ids)
		got, err := st.GetCharacterMail(ctx, c.ID, 42)
		require.NoError(t, err)
		assert.False(t, got.IsRead.ValueOrZero())
		xassert.Equal(t, []int64{1}, got.LabelIDs())
	})
	t.Run("should store nothing when a stored mail gets an invalid label ID", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: 1})
		m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsRead:      optional.New(false),
			LabelIDs:    []int64{1},
			MailID:      42,
		})
		// when
		_, err := st.UpdateOrCreateCharacterMails(ctx, c.ID, []storage.UpdateOrCreateCharacterMailParams{
			{
				FromID:       m.From.ID,
				MailID:       44,
				RecipientIDs: []int64{m.Recipients[0].ID},
				Timestamp:    time.Now(),
			},
			{
				FromID:       m.From.ID,
				IsRead:       optional.New(true),
				LabelIDs:     []int64{0},
				MailID:       42,
				RecipientIDs: []int64{m.Recipients[0].ID},
				Timestamp:    m.Timestamp,
			},
		})
		// then
		require.ErrorIs(t, err, app.ErrInvalid)
		ids, err := st.ListCharacterMailIDs(ctx, c.ID)
		require.NoError(t, err)
		xassert.Equal(t, set.Of[int64](42), ids)
		got, err := st.GetCharacterMail(ctx, c.ID, 42)
		require.NoError(t, err)
		assert.False(t, got.IsRead.ValueOrZero())
		xassert.Equal(t, []int64{1}, got.LabelIDs())
	})
	t.Run("should update read state and labels of stored mails with chunking", func(t *testing.T) {
		// given
		old := st.MaxIDsPerQuery
		st.MaxIDsPerQuery = 1
		defer func() {
			st.MaxIDsPerQuery = old
		}()
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: 1})
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID, LabelID: 2})
		m1 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsRead:      optional.New(false),
			LabelIDs:    []int64{1},
		})
		m2 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsRead:      optional.New(true),
			LabelIDs:    []int64{1, 2},
		})
		// when
		_, err := st.UpdateOrCreateCharacterMails(ctx, c.ID, []storage.UpdateOrCreateCharacterMailParams{
			{
				FromID:       m1.From.ID,
				IsRead:       optional.New(true),
				LabelIDs:     []int64{2},
				MailID:       m1.MailID,
				RecipientIDs: []int64{m1.Recipients[0].ID},
				Timestamp:    m1.Timestamp,
			},
			{
				FromID:       m2.From.ID,
				IsRead:       optional.New(false),
				LabelIDs:     []int64{1},
				MailID:       m2.MailID,
				RecipientIDs: []int64{m2.Recipients[0].ID},
				Timestamp:    m2.Timestamp,
			},
		})
		// then
		require.NoError(t, err)
		got1, err := st.GetCharacterMail(ctx, c.ID, m1.MailID)
		require.NoError(t, err)
		xassert.EqualOptional(t, true, got1.IsRead)
		xassert.Equal(t, []int64{2}, got1.LabelIDs())
		got2, err := st.GetCharacterMail(ctx, c.ID, m2.MailID)
		require.NoError(t, err)
		assert.False(t, got2.IsRead.ValueOrZero())
		xassert.Equal(t, []int64{1}, got2.LabelIDs())
	})
	t.Run("should reject character ID 0", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		// when
		_, err := st.UpdateOrCreateCharacterMails(ctx, 0, nil)
		// then
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
}
