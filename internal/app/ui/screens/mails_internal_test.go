package screens

import (
	"database/sql"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/jarcoal/httpmock"
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
		f.a = NewMailsForCharacter(testdouble.NewUIFake(testdouble.UIParams{
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
		assert.Equal(t, f.mail1.Subject.ValueOrZero(), f.a.ReadingPane.subjectLabel.Text)
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
		assert.Empty(t, f.a.ReadingPane.subjectLabel.Text)
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
		assert.Empty(t, f.a.ReadingPane.subjectLabel.Text)
	})
}

func TestMailsReadingPane_LoadMail(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	character := factory.CreateCharacterFull()
	mail1 := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: character.ID})
	mail2 := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: character.ID})
	a := NewMailsForCharacter(testdouble.NewUIFake(testdouble.UIParams{
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
		assert.Equal(t, mail1.Subject.ValueOrZero(), p.subjectLabel.Text)
	})
	t.Run("ignores earlier request completing after later one", func(t *testing.T) {
		request(mail1.MailID)
		request(mail2.MailID)
		p.loadMail(t.Context(), character.ID, mail2.MailID)
		p.loadMail(t.Context(), character.ID, mail1.MailID)
		require.NotNil(t, p.mail)
		assert.Equal(t, mail2.MailID, p.mail.MailID)
		assert.Equal(t, mail2.Subject.ValueOrZero(), p.subjectLabel.Text)
	})
	t.Run("ignores earlier request completing before later one", func(t *testing.T) {
		request(mail1.MailID)
		request(mail2.MailID)
		p.loadMail(t.Context(), character.ID, mail1.MailID)
		assert.Nil(t, p.mail)
		assert.Empty(t, p.subjectLabel.Text)
		p.loadMail(t.Context(), character.ID, mail2.MailID)
		require.NotNil(t, p.mail)
		assert.Equal(t, mail2.MailID, p.mail.MailID)
	})
	t.Run("ignores result after pane was cleared", func(t *testing.T) {
		request(mail1.MailID)
		p.clear()
		p.loadMail(t.Context(), character.ID, mail1.MailID)
		assert.Nil(t, p.mail)
		assert.Empty(t, p.subjectLabel.Text)
	})
	t.Run("ignores result for other character", func(t *testing.T) {
		request(mail1.MailID)
		p.loadMail(t.Context(), character.ID+1, mail1.MailID)
		assert.Nil(t, p.mail)
		assert.Empty(t, p.bodyLabel.Text)
	})
	t.Run("ignores error of stale request", func(t *testing.T) {
		request(mail2.MailID)
		p.loadMail(t.Context(), character.ID, 999_999_999) // does not exist
		assert.Nil(t, p.mail)
		assert.Empty(t, p.bodyLabel.Text)
	})
	t.Run("shows error of current request", func(t *testing.T) {
		request(999_999_999)
		p.loadMail(t.Context(), character.ID, 999_999_999)
		assert.Nil(t, p.mail)
		assert.Contains(t, p.bodyLabel.Text, "ERROR")
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
	a := NewMailsForCharacter(u)
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
	t.Run("changing folder with active filter does not filter stale rows", func(t *testing.T) {
		reset(t)
		var sent *mailFolderNode
		a.NavigationPane.folders.Data().Walk(nil, func(n *mailFolderNode) bool {
			if n.Type == folderNodeSent {
				sent = n
			}
			return true
		})
		require.NotNil(t, sent)
		mp.filterChip.SetSelected(map[string]string{mailsFilterFrom: "Caldari"})
		require.Equal(t, []string{"Bravo"}, subjects())
		var pending []func()
		orig := runAsync
		runAsync = func(f func()) { pending = append(pending, f) }
		t.Cleanup(func() { runAsync = orig })

		mp.setCurrentFolder(t.Context(), sent)

		assert.Len(t, pending, 1) // only the run for the newly loaded rows
		for _, f := range pending {
			f()
		}
		assert.Empty(t, mp.rowsFiltered)
		assert.Empty(t, mp.filterChip.Selected()[mailsFilterFrom])
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

func TestMails_UnreadCount(t *testing.T) {
	setup := func(t *testing.T) (testutil.Factory, *storage.Storage, *app.Character, *app.CharacterMailLabel) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
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
		custom := factory.CreateCharacterMailLabel(app.CharacterMailLabel{CharacterID: c.ID})
		return factory, st, c, custom
	}
	unreadCount := func(t *testing.T, st *storage.Storage, c *app.Character) int64 {
		a := NewMailsForCharacter(testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}))
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		return a.unreadCount.Load()
	}

	t.Run("counts unread mail with one label once", func(t *testing.T) {
		factory, st, c, _ := setup(t)
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsRead:      optional.New(false),
			LabelIDs:    []int64{app.MailLabelInbox},
		})
		assert.EqualValues(t, 1, unreadCount(t, st, c))
	})
	t.Run("counts unread mail with two labels once", func(t *testing.T) {
		factory, st, c, custom := setup(t)
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsRead:      optional.New(false),
			LabelIDs:    []int64{app.MailLabelInbox, custom.LabelID},
		})
		assert.EqualValues(t, 1, unreadCount(t, st, c))
	})
	t.Run("counts unread mailing list mail with label once", func(t *testing.T) {
		factory, st, c, _ := setup(t)
		list := factory.CreateCharacterMailList(c.ID)
		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			IsRead:       optional.New(false),
			LabelIDs:     []int64{app.MailLabelInbox},
			RecipientIDs: []int64{list.ID},
		})
		assert.EqualValues(t, 1, unreadCount(t, st, c))
	})
}

func TestMailsMessagePane_MarkRead(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)
	setup := func(t *testing.T, esiStatus int) (*Mails, *app.CharacterMail, *[]string) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		httpmock.Reset()
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
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
		m := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsRead:      optional.New(false),
			LabelIDs:    []int64{app.MailLabelInbox},
		})
		var snackbars []string
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:                 test.NewTempApp(t),
			Storage:             st,
			DisplaySnackbarFunc: func(s string) { snackbars = append(snackbars, s) },
		})
		a := NewMailsForCharacter(u)
		u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		require.Len(t, a.MessagePane.rowsFiltered, 1)
		require.False(t, a.MessagePane.rowsFiltered[0].isRead2)
		require.EqualValues(t, 1, a.unreadCount.Load())
		httpmock.RegisterResponder(
			"PUT",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail/%d", c.ID, m.MailID),
			func(req *http.Request) (*http.Response, error) {
				// UI must already show the mail as read while ESI is called
				assert.True(t, a.MessagePane.rowsFiltered[0].isRead2)
				assert.EqualValues(t, 0, a.unreadCount.Load())
				return httpmock.NewStringResponse(esiStatus, ""), nil
			})
		return a, m, &snackbars
	}

	t.Run("shows mail as read before ESI is updated", func(t *testing.T) {
		a, _, snackbars := setup(t, 204)
		a.MessagePane.markRead(a.MessagePane.rowsFiltered[0])
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		assert.True(t, a.MessagePane.rowsFiltered[0].isRead2)
		assert.EqualValues(t, 0, a.unreadCount.Load())
		assert.Empty(t, *snackbars)
	})
	t.Run("shows mail as unread again when ESI update fails", func(t *testing.T) {
		a, _, snackbars := setup(t, 500)
		a.MessagePane.markRead(a.MessagePane.rowsFiltered[0])
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		assert.False(t, a.MessagePane.rowsFiltered[0].isRead2)
		assert.EqualValues(t, 1, a.unreadCount.Load())
		assert.Len(t, *snackbars, 1)
	})
}

func TestUnifiedMails(t *testing.T) {
	type fixture struct {
		a       *Mails
		u       baseUI
		st      *storage.Storage
		factory testutil.Factory
		c1, c2  *app.Character
		list    *app.EveEntity
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
	now := time.Now().UTC()
	createInboxMail := func(f fixture, c *app.Character, mailID int64, age time.Duration) *app.CharacterMail {
		return f.factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			IsRead:      optional.New(false),
			LabelIDs:    []int64{app.MailLabelInbox},
			MailID:      mailID,
			Timestamp:   now.Add(-age),
		})
	}
	setup := func(t *testing.T) fixture {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		f := fixture{st: st, factory: factory}
		f.c1 = createCharacter(factory)
		f.c2 = createCharacter(factory)
		f.list = factory.CreateCharacterMailList(f.c1.ID)
		require.NoError(t, st.CreateCharacterMailList(t.Context(), f.c2.ID, f.list.ID))
		f.u = testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		f.a = NewUnifiedMails(f.u)
		return f
	}
	characterIDs := func(a *Mails) []int64 {
		var ids []int64
		for _, r := range a.MessagePane.rowsFiltered {
			ids = append(ids, r.characterID)
		}
		return ids
	}

	t.Run("shows inbox mails of all characters", func(t *testing.T) {
		f := setup(t)
		createInboxMail(f, f.c1, 1, 2*time.Hour)
		createInboxMail(f, f.c2, 2, time.Hour)
		f.u.Signals().AppInit.Emit(t.Context(), struct{}{})
		folder := f.a.MessagePane.currentFolder.Load()
		require.NotNil(t, folder)
		assert.Equal(t, folderNodeInbox, folder.Type)
		assert.Equal(t, []int64{f.c2.ID, f.c1.ID}, characterIDs(f.a))
		assert.Equal(t, f.c2.EveCharacter.Name, f.a.MessagePane.rowsFiltered[0].characterName)
		assert.EqualValues(t, 2, f.a.unreadCount.Load())
	})
	t.Run("shows merged mailing lists and no custom labels", func(t *testing.T) {
		f := setup(t)
		f.u.Signals().AppInit.Emit(t.Context(), struct{}{})
		var lists []int64
		var hasLabels bool
		f.a.NavigationPane.folders.Data().Walk(nil, func(n *mailFolderNode) bool {
			if n.Category == nodeCategoryList {
				lists = append(lists, n.ObjID)
			}
			if n.Type == folderNodeLabel {
				hasLabels = true
			}
			return true
		})
		assert.Equal(t, []int64{f.list.ID}, lists)
		assert.False(t, hasLabels)
		assert.True(t, f.a.NavigationPane.compose.Visible())
	})
	t.Run("can filter by character", func(t *testing.T) {
		f := setup(t)
		createInboxMail(f, f.c1, 1, 2*time.Hour)
		createInboxMail(f, f.c2, 2, time.Hour)
		f.u.Signals().AppInit.Emit(t.Context(), struct{}{})
		mp := f.a.MessagePane
		mp.filterChip.SetSelected(map[string]string{mailsFilterCharacter: f.c1.EveCharacter.Name})
		mp.filterRowsAsync()
		assert.Equal(t, []int64{f.c1.ID}, characterIDs(f.a))
	})
	t.Run("shows the selected copy of a mail received by several characters", func(t *testing.T) {
		f := setup(t)
		createInboxMail(f, f.c1, 42, time.Hour)
		createInboxMail(f, f.c2, 42, time.Hour)
		f.u.Signals().AppInit.Emit(t.Context(), struct{}{})
		mp := f.a.MessagePane
		idx := slices.IndexFunc(mp.rowsFiltered, func(r mailRow) bool {
			return r.characterID == f.c2.ID
		})
		require.NotEqual(t, -1, idx)
		p := f.a.ReadingPane
		p.showMail(mp.rowsFiltered[idx])
		require.NotNil(t, p.mail)
		assert.Equal(t, f.c2.ID, p.mail.CharacterID)
	})
	t.Run("clears reading pane when the shown copy of a mail is gone", func(t *testing.T) {
		f := setup(t)
		createInboxMail(f, f.c1, 42, time.Hour)
		createInboxMail(f, f.c2, 42, time.Hour)
		f.u.Signals().AppInit.Emit(t.Context(), struct{}{})
		mp := f.a.MessagePane
		idx := slices.IndexFunc(mp.rowsFiltered, func(r mailRow) bool {
			return r.characterID == f.c2.ID
		})
		require.NotEqual(t, -1, idx)
		p := f.a.ReadingPane
		var cleared int
		p.OnCleared = func() { cleared++ }
		p.showMail(mp.rowsFiltered[idx])
		require.NoError(t, f.st.DeleteCharacterMail(t.Context(), f.c2.ID, 42))

		mp.update(t.Context())

		assert.Equal(t, []int64{f.c1.ID}, characterIDs(f.a))
		assert.Zero(t, p.requested.mailID)
		assert.Equal(t, 1, cleared)
	})
	t.Run("is refreshed when a mail is read in the character screen", func(t *testing.T) {
		f := setup(t)
		m := createInboxMail(f, f.c1, 1, time.Hour)
		character := NewMailsForCharacter(f.u)
		f.u.Signals().AppInit.Emit(t.Context(), struct{}{})
		require.EqualValues(t, 1, f.a.unreadCount.Load())
		require.NoError(t, f.st.UpdateCharacterMailSetIsRead(t.Context(), f.c1.ID, m.ID, true))

		character.mailsChanged(t.Context(), f.c1.ID)

		assert.EqualValues(t, 0, f.a.unreadCount.Load())
		assert.True(t, f.a.MessagePane.rowsFiltered[0].isRead2)
	})
	t.Run("shows status when there are no characters", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		a := NewUnifiedMails(u)
		u.Signals().AppInit.Emit(t.Context(), struct{}{})
		assert.Equal(t, "No characters", a.MessagePane.footerLabel.Text)
		assert.True(t, a.MessagePane.searchEntry.Disabled())
		assert.Nil(t, a.MessagePane.currentFolder.Load())
	})
}

func TestMailsReadingPane_Owner(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	c := factory.CreateCharacterFull()
	m := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: c.ID})
	r := mailRow{characterID: c.ID, characterName: c.EveCharacter.Name, mailID: m.MailID}
	t.Run("unified shows owner", func(t *testing.T) {
		a := NewUnifiedMails(testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}))
		a.ReadingPane.showMail(r)
		require.NotNil(t, a.ReadingPane.mail)
		assert.True(t, headerShowsOwner(a.ReadingPane.headerWidget, c.EveCharacter.Name))
	})
	t.Run("character view does not show owner", func(t *testing.T) {
		a := NewMailsForCharacter(testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}))
		a.ReadingPane.showMail(r)
		require.NotNil(t, a.ReadingPane.mail)
		assert.False(t, headerShowsOwner(a.ReadingPane.headerWidget, c.EveCharacter.Name))
	})
}

func TestUnifiedMails_ComposeMenu(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	c1 := factory.CreateCharacterFull()
	c2 := factory.CreateCharacterFull()
	for _, c := range []*app.Character{c1, c2} {
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterMailHeaders,
			CompletedAt: time.Now().UTC(),
		})
	}
	u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
	a := NewUnifiedMails(u)
	labels := func() []string {
		var s []string
		for _, it := range a.composeMenu.Items {
			if it.IsSeparator {
				s = append(s, "---")
				continue
			}
			s = append(s, it.Label)
		}
		return s
	}
	t.Run("lists all characters alphabetically and cancel", func(t *testing.T) {
		u.Signals().AppInit.Emit(t.Context(), struct{}{})
		want := []string{c1.EveCharacter.Name, c2.EveCharacter.Name}
		slices.Sort(want)
		want = append(want, "---", "Cancel")
		assert.Equal(t, want, labels())
		items := a.composeMenu.Items
		assert.NotNil(t, items[len(items)-1].Action, "cancel needs an action to close the menu")
		assert.NotNil(t, items[0].Icon)
		assert.False(t, a.NavigationPane.compose.Disabled())
	})
	t.Run("includes added character", func(t *testing.T) {
		c := factory.CreateCharacterFull()
		u.Signals().CharacterAdded.Emit(t.Context(), c)
		assert.Contains(t, labels(), c.EveCharacter.Name)
		assert.Len(t, a.composeCharacters, 3)
	})
}

func TestMails_DownloadProgress(t *testing.T) {
	type env struct {
		a       *Mails
		c       *app.Character
		db      *sql.DB
		factory testutil.Factory
		missing []int // values received by OnUpdate
		st      *storage.Storage
		unread  []int // values received by OnUpdate
	}
	setup := func(t *testing.T) *env {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		e := &env{c: factory.CreateCharacterFull(), db: db, factory: factory, st: st}
		e.a = NewMailsForCharacter(testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}))
		e.a.character.Store(e.c)
		e.a.OnUpdate = func(unread, missing int) {
			e.missing = append(e.missing, missing)
			e.unread = append(e.unread, unread)
		}
		return e
	}
	// setupHalfDownloaded returns an env with one of two mail bodies downloaded and the missing mail's ID.
	setupHalfDownloaded := func(t *testing.T) (*env, int64) {
		e := setup(t)
		e.factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: e.c.ID})
		m := e.factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: e.c.ID})
		e.a.NavigationPane.updateDownloaded(t.Context(), true)
		return e, m.MailID
	}
	dropMails := func(t *testing.T, db *sql.DB) {
		if _, err := db.Exec("DROP TABLE character_mails"); err != nil { // let the next check fail
			t.Fatal(err)
		}
	}
	tick := func(t *testing.T, a *Mails) {
		a.u.Signals().RefreshTickerExpired.Emit(t.Context(), struct{}{})
	}

	t.Run("shows percentage while bodies are missing", func(t *testing.T) {
		e, _ := setupHalfDownloaded(t)
		assert.True(t, e.a.NavigationPane.folderDownloaded.Visible())
		assert.Equal(t, "50%", e.a.NavigationPane.folderDownloaded.Text)
		assert.Equal(t, []int{50}, e.missing)
		assert.False(t, e.a.bodiesComplete.Load())
	})
	t.Run("rounds missing percentage up", func(t *testing.T) {
		e := setup(t)
		for range 100 {
			e.factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: e.c.ID})
		}
		e.factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: e.c.ID})

		e.a.NavigationPane.updateDownloaded(t.Context(), true)

		assert.Equal(t, "99%", e.a.NavigationPane.folderDownloaded.Text)
		assert.Equal(t, []int{1}, e.missing)
	})
	t.Run("reports completion", func(t *testing.T) {
		e, mailID := setupHalfDownloaded(t)
		err := e.st.UpdateCharacterMailSetBody(t.Context(), e.c.ID, mailID, optional.New("body"))
		require.NoError(t, err)

		e.a.NavigationPane.updateDownloaded(t.Context(), true)

		assert.False(t, e.a.NavigationPane.folderDownloaded.Visible())
		assert.Equal(t, []int{50, 0}, e.missing)
		assert.True(t, e.a.bodiesComplete.Load())
	})
	t.Run("does not call OnUpdate when unchanged", func(t *testing.T) {
		e, _ := setupHalfDownloaded(t)
		e.a.NavigationPane.updateDownloaded(t.Context(), true)
		assert.Equal(t, []int{50}, e.missing)
	})
	t.Run("tick skips check when complete", func(t *testing.T) {
		e := setup(t)
		e.factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: e.c.ID})
		e.a.NavigationPane.updateDownloaded(t.Context(), true)
		require.True(t, e.a.bodiesComplete.Load())
		dropMails(t, e.db)

		tick(t, e.a)

		assert.Equal(t, "1 total", e.a.NavigationPane.folderTotal.Text)
	})
	t.Run("tick checks while bodies are missing", func(t *testing.T) {
		e, _ := setupHalfDownloaded(t)
		dropMails(t, e.db)

		tick(t, e.a)

		assert.Equal(t, "ERROR", e.a.NavigationPane.folderTotal.Text)
	})
	t.Run("tick retries after error", func(t *testing.T) {
		e := setup(t)
		dropMails(t, e.db)
		e.a.NavigationPane.updateDownloaded(t.Context(), true)
		require.Equal(t, "ERROR", e.a.NavigationPane.folderTotal.Text)
		e.a.NavigationPane.folderTotal.SetText("")

		tick(t, e.a)

		assert.Equal(t, "ERROR", e.a.NavigationPane.folderTotal.Text)
		assert.False(t, e.a.bodiesComplete.Load())
	})
	t.Run("update resets completion", func(t *testing.T) {
		e := setup(t)
		e.factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: e.c.ID,
			Section:     app.SectionCharacterMailHeaders,
			CompletedAt: time.Now().UTC(),
		})
		e.factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: e.c.ID})
		e.a.NavigationPane.updateDownloaded(t.Context(), true)
		require.True(t, e.a.bodiesComplete.Load())
		e.factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: e.c.ID})

		e.a.update(t.Context())

		assert.False(t, e.a.bodiesComplete.Load())
		assert.True(t, e.a.NavigationPane.folderDownloaded.Visible())
	})
	t.Run("update resets completion on early return", func(t *testing.T) {
		e := setup(t)
		e.factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: e.c.ID})
		e.a.NavigationPane.updateDownloaded(t.Context(), true)
		require.True(t, e.a.bodiesComplete.Load())
		e.a.character.Store(e.factory.CreateCharacterFull()) // mail data not loaded yet

		e.a.update(t.Context())

		assert.False(t, e.a.bodiesComplete.Load())
		require.Equal(t, "?", e.a.NavigationPane.folderTotal.Text)
		tick(t, e.a)
		assert.Equal(t, "0 total", e.a.NavigationPane.folderTotal.Text)
	})
	t.Run("discards outdated check", func(t *testing.T) {
		e := setup(t)
		p := e.a.NavigationPane
		older := p.downloadedRun.start()
		newer := p.downloadedRun.start()
		p.applyDownloaded(newer, mailsDownloaded{total: "2 total", downloaded: "50%", missingPercent: 50}, true)
		require.Equal(t, []int{50}, e.missing)

		p.applyDownloaded(older, mailsDownloaded{total: "1 total", complete: true}, true)

		assert.False(t, e.a.bodiesComplete.Load())
		assert.Equal(t, "2 total", p.folderTotal.Text)
		assert.Equal(t, "50%", p.folderDownloaded.Text)
		assert.Equal(t, []int{50}, e.missing)
	})
	t.Run("mail change refreshes total when complete", func(t *testing.T) {
		e := setup(t)
		e.factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: e.c.ID})
		m := e.factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: e.c.ID})
		e.a.NavigationPane.updateDownloaded(t.Context(), true)
		require.True(t, e.a.bodiesComplete.Load())
		require.Equal(t, "2 total", e.a.NavigationPane.folderTotal.Text)
		require.NoError(t, e.st.DeleteCharacterMail(t.Context(), e.c.ID, m.MailID))

		e.a.u.Signals().DataUpdated.Emit(t.Context(), mailsUpdated)

		assert.Equal(t, "1 total", e.a.NavigationPane.folderTotal.Text)
	})
	t.Run("early return does not report previous character", func(t *testing.T) {
		e := setup(t)
		e.factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: e.c.ID})
		e.factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: e.c.ID})
		e.a.NavigationPane.updateDownloaded(t.Context(), true)
		e.a.unreadCount.Store(5)
		e.a.character.Store(e.factory.CreateCharacterFull()) // mail data not loaded yet

		e.a.update(t.Context())
		tick(t, e.a)

		require.NotEmpty(t, e.unread)
		assert.Equal(t, 0, e.unread[len(e.unread)-1])
		assert.Equal(t, 0, e.missing[len(e.missing)-1])
		assert.Zero(t, e.a.unreadCount.Load())
	})
	t.Run("OnUpdate once per update", func(t *testing.T) {
		e := setup(t)
		e.factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: e.c.ID,
			Section:     app.SectionCharacterMailHeaders,
			CompletedAt: time.Now().UTC(),
		})
		e.factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: e.c.ID})
		e.factory.CreateCharacterMail(storage.CreateCharacterMailParams{CharacterID: e.c.ID})

		e.a.update(t.Context())

		assert.Equal(t, []int{50}, e.missing)
	})
}

func TestMailsReadingPane_OnCleared(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	c := factory.CreateCharacterFull()
	m1 := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: c.ID})
	m2 := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{CharacterID: c.ID})
	setup := func(t *testing.T) (*mailsReadingPane, *int) {
		a := NewMailsForCharacter(testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}))
		var cleared int
		a.ReadingPane.OnCleared = func() { cleared++ }
		return a.ReadingPane, &cleared
	}
	t.Run("fires when a shown mail is cleared", func(t *testing.T) {
		p, cleared := setup(t)
		p.showMail(mailRow{characterID: c.ID, mailID: m1.MailID})
		p.clear()
		assert.Equal(t, 1, *cleared)
	})
	t.Run("does not fire when already empty", func(t *testing.T) {
		p, cleared := setup(t)
		p.clear()
		assert.Zero(t, *cleared)
	})
	t.Run("fires when the shown mail is deleted", func(t *testing.T) {
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
		m := factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
			CharacterID: c.ID,
			LabelIDs:    []int64{app.MailLabelInbox},
		})
		a := NewMailsForCharacter(testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}))
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		require.True(t, slices.ContainsFunc(a.MessagePane.rowsFiltered, func(r mailRow) bool {
			return r.mailID == m.MailID
		}), "mail must be listed, else the pane is cleared regardless of the delete")
		var cleared int
		a.ReadingPane.OnCleared = func() { cleared++ }
		a.ReadingPane.showMail(mailRow{characterID: c.ID, mailID: m.MailID})
		a.MessagePane.update(t.Context())
		require.Zero(t, cleared, "listed mail must stay shown")
		require.NoError(t, st.DeleteCharacterMail(t.Context(), c.ID, m.MailID))

		a.mailsChanged(t.Context(), c.ID) // what the delete action does on success

		assert.Equal(t, 1, cleared)
	})
	t.Run("does not fire when another mail is shown", func(t *testing.T) {
		p, cleared := setup(t)
		p.showMail(mailRow{characterID: c.ID, mailID: m1.MailID})
		p.showMail(mailRow{characterID: c.ID, mailID: m2.MailID})
		assert.Zero(t, *cleared)
		assert.Equal(t, m2.MailID, p.requested.mailID)
	})
}

func TestMails_Status(t *testing.T) {
	t.Run("shows status and disables controls when data is not loaded", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		c := factory.CreateCharacterFull()
		a := NewMailsForCharacter(testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}))

		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)

		assert.Equal(t, "Waiting for mail data to be loaded...", a.MessagePane.footerLabel.Text)
		assert.Equal(t, "?", a.MessagePane.topLabel.Text)
		assert.True(t, a.MessagePane.searchEntry.Disabled())
		assert.True(t, a.MessagePane.filterChip.Disabled())
		assert.True(t, a.MessagePane.sortButton.Disabled())
		assert.True(t, a.NavigationPane.compose.Disabled())
	})
	t.Run("shows folder and enables controls once data is loaded", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		c := factory.CreateCharacterFull()
		a := NewMailsForCharacter(testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}))
		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterMailHeaders,
			CompletedAt: time.Now().UTC(),
		})

		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c.ID,
			Section:     app.SectionCharacterMailHeaders,
		})

		assert.Equal(t, "Inbox", a.MessagePane.topLabel.Text)
		assert.Equal(t, "Showing 0 / 0 messages", a.MessagePane.footerLabel.Text)
		assert.Equal(t, widget.MediumImportance, a.MessagePane.footerLabel.Importance)
		assert.False(t, a.MessagePane.searchEntry.Disabled())
		assert.False(t, a.MessagePane.filterChip.Disabled())
		assert.False(t, a.MessagePane.sortButton.Disabled())
		assert.False(t, a.NavigationPane.compose.Disabled())
	})
	t.Run("disables controls when there is no character", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		a := NewMailsForCharacter(testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}))

		a.u.Signals().CurrentCharacterExchanged.Emit(t.Context(), nil)

		assert.Equal(t, "No character", a.MessagePane.footerLabel.Text)
		assert.True(t, a.MessagePane.searchEntry.Disabled())
	})
	t.Run("clears search when disabling controls", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		a := NewMailsForCharacter(testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}))
		a.MessagePane.setControlsEnabled(true)
		a.MessagePane.searchEntry.SetText("alpha")

		a.MessagePane.setControlsEnabled(false)

		assert.Empty(t, a.MessagePane.searchEntry.Text)
	})
}

func TestUnifiedMails_Status(t *testing.T) {
	t.Run("shows status and disables controls when no character has data", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		factory.CreateCharacterFull()
		factory.CreateCharacterFull()
		a := NewUnifiedMails(testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}))

		a.u.Signals().AppInit.Emit(t.Context(), struct{}{})

		assert.Equal(t, "Waiting for mail data to be loaded...", a.MessagePane.footerLabel.Text)
		assert.False(t, a.MessagePane.loadingLabel.Visible())
		assert.True(t, a.MessagePane.searchEntry.Disabled())
		assert.True(t, a.NavigationPane.compose.Disabled())
	})
	t.Run("enables controls and shows loaded characters when some have data", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		c1 := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c1.ID,
			Section:     app.SectionCharacterMailHeaders,
			CompletedAt: time.Now().UTC(),
		})
		factory.CreateCharacterFull()
		a := NewUnifiedMails(testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}))

		a.u.Signals().AppInit.Emit(t.Context(), struct{}{})

		assert.True(t, a.MessagePane.loadingLabel.Visible())
		assert.Equal(t, "1/2 characters loaded", a.MessagePane.loadingLabel.Text)
		assert.Equal(t, "Showing 0 / 0 messages", a.MessagePane.footerLabel.Text)
		assert.False(t, a.MessagePane.searchEntry.Disabled())
		assert.False(t, a.NavigationPane.compose.Disabled())
	})
	t.Run("hides loaded characters once all have data", func(t *testing.T) {
		db, st, factory := testutil.NewDBOnDisk(t)
		t.Cleanup(func() { db.Close() })
		c1 := factory.CreateCharacterFull()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c1.ID,
			Section:     app.SectionCharacterMailHeaders,
			CompletedAt: time.Now().UTC(),
		})
		c2 := factory.CreateCharacterFull()
		a := NewUnifiedMails(testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}))
		a.u.Signals().AppInit.Emit(t.Context(), struct{}{})
		require.True(t, a.MessagePane.loadingLabel.Visible())
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c2.ID,
			Section:     app.SectionCharacterMailHeaders,
			CompletedAt: time.Now().UTC(),
		})

		a.u.Signals().CharacterSectionChanged.Emit(t.Context(), app.CharacterSectionUpdated{
			CharacterID: c2.ID,
			Section:     app.SectionCharacterMailHeaders,
		})

		assert.False(t, a.MessagePane.loadingLabel.Visible())
	})
}
