package storage_test

import (
	"context"
	"testing"
	"time"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

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
		if assert.NoError(t, err) {
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
		}
	})
	t.Run("can update is-read", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		m := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{})
		// when
		err := st.UpdateCharacterMailSetIsRead(ctx, m.CharacterID, m.ID, true)
		// then
		if assert.NoError(t, err) {
			got, err := st.GetCharacterMail(ctx, m.CharacterID, m.MailID)
			assert.NoError(t, err)
			assert.True(t, got.IsRead.ValueOrZero())
		}
	})
	t.Run("can update labels", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		m := factory.CreateCharacterMailWithBody()
		label := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: m.CharacterID})
		// when
		err := st.UpdateCharacterMailSetLabels(ctx, m.CharacterID, m.ID, []int64{label.LabelID})
		// then
		if assert.NoError(t, err) {
			got, err := st.GetCharacterMail(ctx, m.CharacterID, m.MailID)
			assert.NoError(t, err)
			assert.Contains(t, got.Labels, label)
		}
	})
	t.Run("can update body", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		m := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{})
		// when
		err := st.UpdateCharacterMailSetBody(ctx, m.CharacterID, m.MailID, optional.New("alpha"))
		// then
		if assert.NoError(t, err) {
			got, err := st.GetCharacterMail(ctx, m.CharacterID, m.MailID)
			assert.NoError(t, err)
			xassert.EqualOptional(t, "alpha", got.Body)
		}
	})
	t.Run("can set processed", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		m := factory.CreateCharacterMailWithBody()
		// when
		err := st.UpdateCharacterMailSetProcessed(ctx, m.ID)
		// then
		if assert.NoError(t, err) {
			o, err := st.GetCharacterMail(ctx, m.CharacterID, m.MailID)
			if assert.NoError(t, err) {
				assert.True(t, o.IsProcessed)
			}
		}
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
		if assert.NoError(t, err) {
			_, err := st.GetCharacterMail(ctx, m.CharacterID, m.MailID)
			assert.ErrorIs(t, err, app.ErrNotFound)
		}
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
		r, err := st.GetCharacterMailLabelUnreadCounts(ctx, c.ID)
		if assert.NoError(t, err) {
			xassert.Equal(t, map[int64]int{app.MailLabelCorp: 2, app.MailLabelInbox: 1}, r)
		}
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
		r, err := st.GetCharacterMailListUnreadCounts(ctx, c.ID)
		if assert.NoError(t, err) {
			xassert.Equal(t, map[int64]int{l1.ID: 1}, r)
		}
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
		r, err := st.GetCharacterMailUnreadCount(ctx, c.ID)
		if assert.NoError(t, err) {
			xassert.Equal(t, 6, r)
		}
	})
	t.Run("should return null when no mail exists", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		c := factory.CreateCharacter()
		// when
		r, err := st.GetCharacterMailUnreadCount(ctx, c.ID)
		if assert.NoError(t, err) {
			xassert.Equal(t, 0, r)
		}
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
		got, err := st.GetAllCharactersMailUnreadCount(ctx)
		if assert.NoError(t, err) {
			xassert.Equal(t, 7, got)
		}
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
		got, err := st.GetCharacterMailCount(ctx, character.ID)
		if assert.NoError(t, err) {
			xassert.Equal(t, 2, got)
		}
	})
	t.Run("character has no mail", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		character := factory.CreateCharacter()
		factory.CreateCharacterMailWithBody()
		// when
		got, err := st.GetCharacterMailCount(ctx, character.ID)
		if assert.NoError(t, err) {
			xassert.Equal(t, 0, got)
		}
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
		if assert.NoError(t, err) {
			xassert.Equal(t, []int64{m2.MailID, m1.MailID}, mailIDsFromHeaders(xx))
			xassert.Equal(t, []int64{c2.ID, c1.ID}, []int64{xx[0].CharacterID, xx[1].CharacterID})
		}
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
		if assert.NoError(t, err) {
			xassert.Equal(t, []int64{m2.MailID, m1.MailID}, mailIDsFromHeaders(xx))
			xassert.Equal(t, []int64{m2.CharacterID, m1.CharacterID}, []int64{xx[0].CharacterID, xx[1].CharacterID})
		}
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
		if assert.NoError(t, err) {
			xassert.Equal(t, []int64{m1.MailID, m2.MailID}, mailIDsFromHeaders(xx))
		}
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
		if assert.NoError(t, err) {
			xassert.Equal(t, []int64{e1.ID, e2.ID}, []int64{ll[0].ID, ll[1].ID})
			assert.Len(t, ll, 2)
		}
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
		got, err := st.GetAllCharactersMailLabelUnreadCounts(ctx)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, map[int64]int{app.MailLabelInbox: 2}, got)
		}
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
		got, err := st.GetAllCharactersMailListUnreadCounts(ctx)
		// then
		if assert.NoError(t, err) {
			xassert.Equal(t, map[int64]int{l.ID: 2}, got)
		}
	})
	t.Run("can count mails and mails without body of all characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		factory.CreateCharacterMailWithBody()
		factory.CreateCharacterMailWithBody()
		factory.CreateCharacterMail()
		// when
		total, err1 := st.GetAllCharactersMailCount(ctx)
		missing, err2 := st.GetAllCharactersMailWithoutBodyCount(ctx)
		// then
		if assert.NoError(t, err1) && assert.NoError(t, err2) {
			xassert.Equal(t, 3, total)
			xassert.Equal(t, 1, missing)
		}
	})
}
