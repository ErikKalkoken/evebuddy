package screens

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestMails_Refresh(t *testing.T) {
	type fixture struct {
		a         *Mails
		st        *storage.Storage
		factory   testutil.Factory
		character *app.Character
		mail1     *app.CharacterMail
		inbox     *mailFolderNode
	}
	createCharacter := func(factory testutil.Factory) *app.Character {
		c := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterMailHeaders,
			CompletedAt: time.Now().UTC(),
		})
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: c.ID,
			LabelID:     app.MailLabelInbox,
			Name:        optional.New("Inbox"),
		})
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID}) // custom label
		return c
	}
	createInboxMail := func(f fixture) *app.CharacterMail {
		return f.factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: f.character.ID,
			IsRead:      optional.New(true),
			LabelIDs:    []int64{app.MailLabelInbox},
		})
	}
	// setup shows mail1 from the inbox, as if the user had opened it.
	setup := func(t *testing.T) fixture {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		f := fixture{st: st, factory: factory, character: createCharacter(factory)}
		f.mail1 = createInboxMail(f)
		createInboxMail(f)
		f.a = NewMails(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
		f.a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), f.character)
		f.a.NavigationPane.folders.Data().Walk(nil, func(n *mailFolderNode) bool {
			if n.Type == folderNodeInbox {
				f.inbox = n
			}
			return true
		})
		require.NotNil(t, f.inbox)
		f.a.MessagePane.setCurrentFolder(t.Context(), f.inbox)
		require.Len(t, f.a.MessagePane.rowsFiltered, 2)
		p := f.a.ReadingPane
		p.requested.characterID, p.requested.mailID = f.character.ID, f.mail1.MailID
		p.loadMail(t.Context(), f.character.ID, f.mail1.MailID)
		require.NotNil(t, p.mail)
		return f
	}
	emitMailHeadersChanged := func(t *testing.T, f fixture) {
		f.a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: f.character.ID,
			Section:     app.SectionCharacterMailHeaders,
		})
	}

	t.Run("keeps folder and open mail when new mail arrives", func(t *testing.T) {
		f := setup(t)
		createInboxMail(f)
		emitMailHeadersChanged(t, f)
		assert.Equal(t, f.inbox.UID(), f.a.MessagePane.currentFolder.Load().UID())
		assert.Len(t, f.a.MessagePane.rowsFiltered, 3)
		require.NotNil(t, f.a.ReadingPane.mail)
		assert.Equal(t, f.mail1.MailID, f.a.ReadingPane.mail.MailID)
		assert.Equal(t, f.mail1.Subject.ValueOrZero(), f.a.ReadingPane.subject.Text)
	})
	t.Run("keeps open branches when new mail arrives", func(t *testing.T) {
		f := setup(t)
		var labels *mailFolderNode
		f.a.NavigationPane.folders.Data().Walk(nil, func(n *mailFolderNode) bool {
			if n.isBranch() && n.Type == folderNodeLabel {
				labels = n
			}
			return true
		})
		require.NotNil(t, labels)
		f.a.NavigationPane.folders.OpenBranchNode(labels)
		createInboxMail(f)
		emitMailHeadersChanged(t, f)
		assert.True(t, f.a.NavigationPane.folders.IsBranchOpenNode(labels))
	})
	t.Run("clears reading pane when open mail is gone", func(t *testing.T) {
		f := setup(t)
		err := f.st.DeleteCharacterMail(t.Context(), f.character.ID, f.mail1.MailID)
		require.NoError(t, err)
		emitMailHeadersChanged(t, f)
		assert.Equal(t, f.inbox.UID(), f.a.MessagePane.currentFolder.Load().UID())
		assert.Len(t, f.a.MessagePane.rowsFiltered, 1)
		assert.Nil(t, f.a.ReadingPane.mail)
		assert.Empty(t, f.a.ReadingPane.subject.Text)
	})
	t.Run("resets to Inbox and clears reading pane on character switch", func(t *testing.T) {
		f := setup(t)
		character2 := createCharacter(f.factory)
		f.a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), character2)
		folder := f.a.MessagePane.currentFolder.Load()
		require.NotNil(t, folder)
		assert.Equal(t, character2.ID, folder.CharacterID)
		assert.Equal(t, folderNodeInbox, folder.Type)
		assert.Nil(t, f.a.ReadingPane.mail)
		assert.Empty(t, f.a.ReadingPane.subject.Text)
	})
}

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

func TestMailsMessagePane_FilterAndSort(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	character := factory.CreateCharacterFull()
	factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
		CharacterID: character.ID,
		Section:     app.SectionCharacterMailHeaders,
		CompletedAt: time.Now().UTC(),
	})
	factory.CreateCharacterMailLabel(app.CharacterMailLabel{
		CharacterID: character.ID,
		LabelID:     app.MailLabelInbox,
		Name:        optional.New("Inbox"),
	})
	amarr := factory.CreateEveEntityCharacter(app.EveEntity{Name: "Amarr"})
	caldari := factory.CreateEveEntityCharacter(app.EveEntity{Name: "Caldari"})
	now := time.Now().UTC()
	createMail := func(subject string, from *app.EveEntity, isRead bool, age time.Duration) *app.CharacterMail {
		return factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: character.ID,
			FromID:      from.ID,
			IsRead:      optional.New(isRead),
			LabelIDs:    []int64{app.MailLabelInbox},
			Subject:     optional.New(subject),
			Timestamp:   now.Add(-age),
		})
	}
	createMail("Alpha", amarr, true, 3*time.Hour)
	bravo := createMail("Bravo", caldari, false, 2*time.Hour)
	createMail("Charlie", amarr, false, time.Hour)

	u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
	a := NewMails(u)
	u.Signals().CurrentCharacterExchanged.Emit(t.Context(), character) // shows Inbox
	mp := a.MessagePane
	inbox := mp.currentFolder.Load()
	require.NotNil(t, inbox)

	subjects := func() []string {
		var s []string
		for _, r := range mp.rowsFiltered {
			s = append(s, r.subject)
		}
		return s
	}
	reset := func(t *testing.T) {
		mp.columnSorter.Set("Date", xwidget.SortDesc)
		mp.setCurrentFolder(t.Context(), inbox)
		require.Equal(t, []string{"Charlie", "Bravo", "Alpha"}, subjects())
	}

	t.Run("shows all mails newest first", func(t *testing.T) {
		reset(t)
		assert.Equal(t, "Showing 3 / 3 messages", mp.footerLabel.Text)
		assert.Equal(t, inbox.Name, mp.topLabel.Text)
	})
	t.Run("search by subject", func(t *testing.T) {
		reset(t)
		mp.searchEntry.SetText("brav")
		assert.Equal(t, []string{"Bravo"}, subjects())
		assert.Equal(t, "Showing 1 / 3 messages", mp.footerLabel.Text)
	})
	t.Run("search by sender", func(t *testing.T) {
		reset(t)
		mp.searchEntry.SetText("amarr")
		assert.Equal(t, []string{"Charlie", "Alpha"}, subjects())
	})
	t.Run("filter by status", func(t *testing.T) {
		reset(t)
		mp.filterChip.SetSelected(map[string]string{mailsFilterStatus: mailsFilterStatusRead})
		mp.filterRowsAsync()
		assert.Equal(t, []string{"Alpha"}, subjects())
	})
	t.Run("filter by from", func(t *testing.T) {
		reset(t)
		mp.filterChip.SetSelected(map[string]string{mailsFilterFrom: "Caldari"})
		mp.filterRowsAsync()
		assert.Equal(t, []string{"Bravo"}, subjects())
	})
	t.Run("sort by subject", func(t *testing.T) {
		reset(t)
		mp.columnSorter.Set("Subject", xwidget.SortAsc)
		mp.filterRowsAsync()
		assert.Equal(t, []string{"Alpha", "Bravo", "Charlie"}, subjects())
	})
	t.Run("changing folder resets search and filters", func(t *testing.T) {
		reset(t)
		mp.searchEntry.SetText("brav")
		mp.filterChip.SetSelected(map[string]string{mailsFilterFrom: "Caldari"})
		mp.filterRowsAsync()
		require.Equal(t, []string{"Bravo"}, subjects())
		mp.setCurrentFolder(t.Context(), inbox)
		assert.Empty(t, mp.searchEntry.Text)
		assert.Empty(t, mp.filterChip.Selected()[mailsFilterFrom])
		assert.Len(t, mp.rowsFiltered, 3)
	})
	t.Run("keeps opened mail under unread filter after it was read", func(t *testing.T) {
		reset(t)
		mp.filterChip.SetSelected(map[string]string{mailsFilterStatus: mailsFilterStatusUnread})
		mp.updateIsRead()
		mp.filterRowsAsync()
		require.Equal(t, []string{"Charlie", "Bravo"}, subjects())
		p := a.ReadingPane
		p.requested.characterID, p.requested.mailID = character.ID, bravo.MailID
		err := st.UpdateCharacterMailSetIsRead(t.Context(), character.ID, bravo.ID, true)
		require.NoError(t, err)

		mp.update(t.Context()) // reload triggered after marking as read

		assert.Equal(t, []string{"Charlie", "Bravo"}, subjects())
		assert.True(t, mp.rowsFiltered[1].isRead2)
		assert.Equal(t, bravo.MailID, p.requested.mailID)
	})
}
