package ui

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCharacterMails_updateUnreadCounts(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	test.ApplyTheme(t, test.Theme())
	t.Run("can update counts from zero", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: c.ID,
			LabelID:     app.MailLabelInbox,
		})
		ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
		ui.setCharacter(c)
		td, _, err := ui.characterMails.fetchFolders(ui.services(), c.ID)
		if err != nil {
			panic(err)
		}
		factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			LabelIDs:    []int32{app.MailLabelInbox},
			CharacterID: c.ID,
			IsRead:      false,
		})
		// when
		x, err := ui.characterMails.updateCountsInTree(ui.services(), c.ID, td)
		// then
		if assert.NoError(t, err) {
			assert.Equal(t, 1, x)
			inbox, _ := td.Node(makeMailNodeUID(c.ID, folderNodeInbox, app.MailLabelInbox))
			assert.Equal(t, 1, inbox.UnreadCount)
			unread, _ := td.Node(makeMailNodeUID(c.ID, folderNodeUnread, app.MailLabelUnread))
			assert.Equal(t, 1, unread.UnreadCount)
			all, _ := td.Node(makeMailNodeUID(c.ID, folderNodeAll, app.MailLabelAll))
			assert.Equal(t, 1, all.UnreadCount)
		}
	})
	t.Run("can reset counts to zero", func(t *testing.T) {
		// given
		testutil.TruncateTables(db)
		c := factory.CreateCharacter()
		factory.CreateCharacterMailLabel(app.CharacterMailLabel{
			CharacterID: c.ID,
			LabelID:     app.MailLabelInbox,
		})
		ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
		ui.setCharacter(c)
		td, _, err := ui.characterMails.fetchFolders(ui.services(), c.ID)
		if err != nil {
			panic(err)
		}
		inbox, _ := td.Node(makeMailNodeUID(c.ID, folderNodeInbox, app.MailLabelInbox))
		inbox.UnreadCount = 1
		if err := td.Replace(inbox); err != nil {
			panic(err)
		}
		// when
		x, err := ui.characterMails.updateCountsInTree(ui.services(), c.ID, td)
		// then
		if assert.NoError(t, err) {
			// td.Print("")
			assert.Equal(t, 0, x)
			inbox, _ = td.Node(makeMailNodeUID(c.ID, folderNodeInbox, app.MailLabelInbox))
			assert.Equal(t, 0, inbox.UnreadCount)
		}
	})
}
