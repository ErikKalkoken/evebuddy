package characters

// FIXME

// func TestCharacterMails_updateUnreadCounts(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip(ui.SkipUIReason)
// 	}

// 	db, st, factory := testutil.NewDBOnDisk(t)
// 	defer db.Close()
// 	test.ApplyTheme(t, test.Theme())

// 	t.Run("can update counts from zero", func(t *testing.T) {
// 		// given
// 		testutil.MustTruncateTables(db)
// 		c := factory.CreateCharacter()
// 		factory.CreateCharacterMailLabel(app.CharacterMailLabel{
// 			CharacterID: c.ID,
// 			LabelID:     app.MailLabelInbox,
// 		})
// 		a := NewMails(testdouble.NewUIFake(testdouble.UIParams{
// 			App:     test.NewTempApp(t),
// 			Storage: st,
// 		}))

// 		factory.CreateCharacterMailWithBody(storage.CreateCharacterMailParams{
// 			LabelIDs:    []int64{app.MailLabelInbox},
// 			CharacterID: c.ID,
// 		})
// 		// when
// 		a.update(t.Context())
// 		// then
// 	})

// 	t.Run("can reset counts to zero", func(t *testing.T) {
// 		// given
// 		testutil.MustTruncateTables(db)
// 		c := factory.CreateCharacter()
// 		factory.CreateCharacterMailLabel(app.CharacterMailLabel{
// 			CharacterID: c.ID,
// 			LabelID:     app.MailLabelInbox,
// 		})
// 		ui := MakeFakeBaseUI(st, test.NewTempApp(t), true)
// 		ui.setCharacter(c)
// 		td, _, err := ui.characterMails.fetchFolders(ui.services(), c.ID)
// 		if err != nil {
// 			panic(err)
// 		}
// 		inbox, _ := td.Node(makeMailNodeUID(c.ID, folderNodeInbox, app.MailLabelInbox))
// 		inbox.UnreadCount = 1
// 		if err := td.Replace(inbox); err != nil {
// 			panic(err)
// 		}
// 		// when
// 		x, err := ui.characterMails.updateCountsInTree(ui.services(), c.ID, td)
// 		// then
// 		if assert.NoError(t, err) {
// 			// td.Print("")
// 			xassert.Equal(t, 0, x)
// 			inbox, _ = td.Node(makeMailNodeUID(c.ID, folderNodeInbox, app.MailLabelInbox))
// 			xassert.Equal(t, 0, inbox.UnreadCount)
// 		}
// 	})
// }
