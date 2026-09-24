package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestMailsReadingPane_LoadMail(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	character := factory.CreateCharacterFull()
	mail1 := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: character.ID})
	mail2 := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: character.ID})
	a := NewMails(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	}))
	a.character.Store(character)
	p := a.ReadingPane

	// request mimics showMail without starting the async load,
	// so tests can control the order in which loads complete.
	request := func(mailID int64) {
		p.clear()
		p.requested.characterID, p.requested.mailID = character.ID, mailID
	}

	t.Run("shows requested mail", func(t *testing.T) {
		request(mail1.MailID)
		p.loadMail(t.Context(), character.ID, mail1.MailID)
		require.NotNil(t, p.mail)
		assert.Equal(t, mail1.MailID, p.mail.MailID)
		assert.Equal(t, mail1.Subject.ValueOrZero(), p.subject.Text)
	})
	t.Run("ignores earlier request completing after later one", func(t *testing.T) {
		request(mail1.MailID)
		request(mail2.MailID)
		p.loadMail(t.Context(), character.ID, mail2.MailID)
		p.loadMail(t.Context(), character.ID, mail1.MailID)
		require.NotNil(t, p.mail)
		assert.Equal(t, mail2.MailID, p.mail.MailID)
		assert.Equal(t, mail2.Subject.ValueOrZero(), p.subject.Text)
	})
	t.Run("ignores earlier request completing before later one", func(t *testing.T) {
		request(mail1.MailID)
		request(mail2.MailID)
		p.loadMail(t.Context(), character.ID, mail1.MailID)
		assert.Nil(t, p.mail)
		assert.Empty(t, p.subject.Text)
		p.loadMail(t.Context(), character.ID, mail2.MailID)
		require.NotNil(t, p.mail)
		assert.Equal(t, mail2.MailID, p.mail.MailID)
	})
	t.Run("ignores result after pane was cleared", func(t *testing.T) {
		request(mail1.MailID)
		p.clear()
		p.loadMail(t.Context(), character.ID, mail1.MailID)
		assert.Nil(t, p.mail)
		assert.Empty(t, p.subject.Text)
	})
	t.Run("ignores result for other character", func(t *testing.T) {
		request(mail1.MailID)
		p.loadMail(t.Context(), character.ID+1, mail1.MailID)
		assert.Nil(t, p.mail)
		assert.Empty(t, p.body.Text)
	})
	t.Run("ignores error of stale request", func(t *testing.T) {
		request(mail2.MailID)
		p.loadMail(t.Context(), character.ID, 999_999_999) // does not exist
		assert.Nil(t, p.mail)
		assert.Empty(t, p.body.Text)
	})
	t.Run("shows error of current request", func(t *testing.T) {
		request(999_999_999)
		p.loadMail(t.Context(), character.ID, 999_999_999)
		assert.Nil(t, p.mail)
		assert.Contains(t, p.body.Text, "ERROR")
	})
}
