package charactermanager

import (
	"context"
	"errors"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	goesi "github.com/fnt-eve/goesi-openapi"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

func TestAdmin_Update(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("shows characters sorted by name with their missing scopes", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		corporation := factory.CreateCorporation()
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Bravo", CorporationID: corporation.ID})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c1.ID, Scopes: app.Scopes()})
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Alpha", CorporationID: corporation.ID})
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		scopes := app.Scopes()
		scopes.Delete(goesi.ScopeMailSendMailV1)
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c2.ID, Scopes: scopes})
		// when
		cw.characterAdmin.update(context.Background())
		// then
		a := cw.characterAdmin
		assert.Equal(t, "Characters (2)", a.ab.Title())
		want := []adminRow{
			{
				characterID:   c2.ID,
				corporationID: corporation.ID,
				characterName: "Alpha",
				missingScopes: set.Of(goesi.ScopeMailSendMailV1),
			},
			{
				characterID:   c1.ID,
				corporationID: corporation.ID,
				characterName: "Bravo",
				missingScopes: set.Of[string](),
			},
		}
		if assert.Len(t, a.rows, 2) {
			for i, w := range want {
				got := a.rows[i]
				assert.Equal(t, w.characterID, got.characterID)
				assert.Equal(t, w.corporationID, got.corporationID)
				assert.Equal(t, w.characterName, got.characterName)
				assert.True(t, w.missingScopes.Equal(got.missingScopes), "%s: missing scopes: %s", w.characterName, got.missingScopes)
			}
		}
	})
	t.Run("reports error and keeps rows when loading fails", func(t *testing.T) {
		// given
		db2, st2, _ := testutil.NewDBInMemory()
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st2})}
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		var messages []string
		cw.displaySnackbar = func(s string) {
			messages = append(messages, s)
		}
		a := cw.characterAdmin
		a.rows = []adminRow{{characterID: 42, characterName: "Alpha"}}
		db2.Close()
		// when
		a.update(context.Background())
		// then
		assert.Equal(t, []adminRow{{characterID: 42, characterName: "Alpha"}}, a.rows)
		if assert.Len(t, messages, 1) {
			assert.Contains(t, messages[0], "ERROR: Failed to update characters")
		}
	})
}

func TestAdmin_AddButton(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()

	t.Run("is enabled when online", func(t *testing.T) {
		// given
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		// when
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		// then
		assert.False(t, cw.characterAdmin.addButton.Disabled())
	})
	t.Run("is disabled when offline", func(t *testing.T) {
		// given
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st}), isOffline: true}
		// when
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		// then
		assert.True(t, cw.characterAdmin.addButton.Disabled())
	})
}

func TestAdmin_DeleteCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	t.Run("deletes character and updates the list", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		t.Cleanup(u.Corporation().Stop)
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		var messages []string
		cw.displaySnackbar = func(s string) {
			messages = append(messages, s)
		}
		corporation := factory.CreateCorporation()
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Alpha", CorporationID: corporation.ID})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Bravo", CorporationID: corporation.ID})
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		a := cw.characterAdmin
		a.update(context.Background())
		r := adminRow{characterID: c1.ID, characterName: "Alpha", corporationID: corporation.ID}
		// when
		a.deleteCharacter(context.Background(), r)
		// then
		_, err := st.GetCharacter(context.Background(), c1.ID)
		assert.ErrorIs(t, err, app.ErrNotFound)
		if assert.Len(t, a.rows, 1) {
			assert.Equal(t, c2.ID, a.rows[0].characterID)
		}
		assert.Equal(t, "Characters (1)", a.ab.Title())
		assert.Equal(t, []string{"Character Alpha deleted"}, messages)
	})
	t.Run("emits CharacterRemoved", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		t.Cleanup(u.Corporation().Stop)
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		cw.displaySnackbar = func(string) {}
		corporation := factory.CreateCorporation()
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{Name: "Alpha", CorporationID: corporation.ID})
		c := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
		removed := make(chan *app.EntityShort, 1)
		u.Signals().CharacterRemoved.AddListener(func(_ context.Context, x *app.EntityShort) {
			removed <- x
		})
		r := adminRow{characterID: c.ID, characterName: "Alpha", corporationID: corporation.ID}
		// when
		cw.characterAdmin.deleteCharacter(context.Background(), r)
		// then
		select {
		case got := <-removed:
			assert.Equal(t, &app.EntityShort{ID: c.ID, Name: "Alpha"}, got)
		case <-time.After(5 * time.Second):
			t.Fatal("CharacterRemoved not emitted")
		}
	})
	t.Run("sets another current character when the current one was deleted", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		t.Cleanup(u.Corporation().Stop)
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		cw.displaySnackbar = func(string) {}
		corporation := factory.CreateCorporation()
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corporation.ID})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corporation.ID})
		factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		u.currentCharacter = c1
		r := adminRow{characterID: c1.ID, characterName: ec1.Name, corporationID: corporation.ID}
		// when
		cw.characterAdmin.deleteCharacter(context.Background(), r)
		// then
		assert.Equal(t, 1, u.setAnyCharacterCalls)
	})
	t.Run("keeps current character when another one was deleted", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		t.Cleanup(u.Corporation().Stop)
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		cw.displaySnackbar = func(string) {}
		corporation := factory.CreateCorporation()
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corporation.ID})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corporation.ID})
		c2 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		u.currentCharacter = c2
		r := adminRow{characterID: c1.ID, characterName: ec1.Name, corporationID: corporation.ID}
		// when
		cw.characterAdmin.deleteCharacter(context.Background(), r)
		// then
		assert.Equal(t, 0, u.setAnyCharacterCalls)
	})
	t.Run("deletes corporation when it has no characters left", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		t.Cleanup(u.Corporation().Stop)
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		cw.displaySnackbar = func(string) {}
		corporation := factory.CreateCorporation()
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corporation.ID})
		c := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
		r := adminRow{characterID: c.ID, characterName: ec.Name, corporationID: corporation.ID}
		// when
		cw.characterAdmin.deleteCharacter(context.Background(), r)
		// then
		_, err := st.GetCorporation(context.Background(), corporation.ID)
		require.ErrorIs(t, err, app.ErrNotFound)
	})
	t.Run("keeps corporation when it still has characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		t.Cleanup(u.Corporation().Stop)
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		cw.displaySnackbar = func(string) {}
		corporation := factory.CreateCorporation()
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corporation.ID})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corporation.ID})
		factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		r := adminRow{characterID: c1.ID, characterName: ec1.Name, corporationID: corporation.ID}
		// when
		cw.characterAdmin.deleteCharacter(context.Background(), r)
		// then
		_, err := st.GetCorporation(context.Background(), corporation.ID)
		require.NoError(t, err)
	})
	t.Run("removes corporation data before CharacterRemoved", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		t.Cleanup(u.Corporation().Stop)
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		cw.displaySnackbar = func(string) {}
		// A second character keeps the corporation, but has no permission for its industry jobs.
		corporation := factory.CreateCorporation()
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corporation.ID})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corporation.ID})
		factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		j := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: corporation.ID,
		})
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: corporation.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		removed := make(chan bool, 1)
		u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			_, err := st.GetCorporationIndustryJob(ctx, j.CorporationID, j.JobID)
			removed <- errors.Is(err, app.ErrNotFound)
		})
		r := adminRow{characterID: c1.ID, characterName: ec1.Name, corporationID: corporation.ID}
		// when
		cw.characterAdmin.deleteCharacter(context.Background(), r)
		// then
		select {
		case got := <-removed:
			assert.True(t, got, "corporation data still there when CharacterRemoved was emitted")
		case <-time.After(5 * time.Second):
			t.Fatal("CharacterRemoved not emitted")
		}
	})
	t.Run("reports error when character does not exist", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		t.Cleanup(u.Corporation().Stop)
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		var messages []string
		cw.displaySnackbar = func(s string) {
			messages = append(messages, s)
		}
		r := adminRow{characterID: 42, characterName: "Alpha"}
		// when
		cw.characterAdmin.deleteCharacter(context.Background(), r)
		// then
		if assert.Len(t, messages, 1) {
			assert.Contains(t, messages[0], "ERROR: Failed to delete character Alpha")
		}
	})
	t.Run("skips UI updates when canceled", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := &uiFake{UIFake: testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		t.Cleanup(u.Corporation().Stop)
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		defer cw.stop()
		var messages []string
		cw.displaySnackbar = func(s string) {
			messages = append(messages, s)
		}
		corporation := factory.CreateCorporation()
		ec := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corporation.ID})
		c := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec.ID})
		u.currentCharacter = c
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		r := adminRow{characterID: c.ID, characterName: ec.Name, corporationID: corporation.ID}
		// when
		cw.characterAdmin.deleteCharacter(ctx, r)
		// then
		assert.Empty(t, messages)
		assert.Equal(t, 0, u.setAnyCharacterCalls)
	})
}

func TestAdminListItem_Set(t *testing.T) {
	loadIcon := func(*app.EveEntity, int, func(fyne.Resource)) {}

	t.Run("shows sorted missing scopes", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newAdminListItem(func(adminRow) {}, loadIcon)
		r := adminRow{characterID: 42, characterName: "Alpha", missingScopes: set.Of("b-scope", "a-scope")}
		// when
		w.set(r)
		// then
		assert.True(t, w.issue.Visible())
		want := "Please re-add to approve missing scopes: a-scope, b-scope"
		assert.Equal(t, want, w.issueLabel.ToolTip())
		assert.Equal(t, want, w.issueIcon.ToolTip())
	})
	t.Run("hides issue when no scopes are missing", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		w := newAdminListItem(func(adminRow) {}, loadIcon)
		w.set(adminRow{characterID: 42, characterName: "Alpha", missingScopes: set.Of("a-scope")})
		// when
		w.set(adminRow{characterID: 42, characterName: "Alpha"})
		// then
		assert.False(t, w.issue.Visible())
	})
	t.Run("delete button shows delete dialog for its row", func(t *testing.T) {
		// given
		test.NewTempApp(t)
		var got adminRow
		w := newAdminListItem(func(r adminRow) {
			got = r
		}, loadIcon)
		r := adminRow{characterID: 42, characterName: "Alpha"}
		w.set(r)
		// when
		test.Tap(w.delete)
		// then
		assert.Equal(t, r, got)
	})
}
