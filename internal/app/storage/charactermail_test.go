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
			xassert.Equal(t, int64(42), m.MailID)
			xassert.Equal(t, "body", m.Body.ValueOrZero())
			xassert.Equal(t, f, m.From)
			xassert.Equal(t, c.ID, m.CharacterID)
			xassert.Equal(t, "subject", m.Subject.ValueOrZero())
			xassert.Equal2(t, timestamp, m.Timestamp)
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
			xassert.Equal(t, "alpha", got.Body.ValueOrZero())
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
		xassert.Equal2(t, want, got)
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
		xassert.Equal2(t, mailIDs, got)
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
