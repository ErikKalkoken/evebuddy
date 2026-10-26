package mailer

import (
	"context"
	"fmt"
	"os"
	"testing"

	"fyne.io/fyne/v2"
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
)

func TestMain(m *testing.M) {
	// Leftover goroutines from one test can render text concurrently with the next,
	// which is not safe in Fyne's shared text shaper.
	runAsync = func(f func()) { f() }
	os.Exit(m.Run())
}

const searchURL = `=~^https://esi\.evetech\.net/characters/\d+/search`

// dialogTexts returns the label texts of the dialog shown in w.
func dialogTexts(t *testing.T, w fyne.Window) []string {
	t.Helper()
	top := w.Canvas().Overlays().Top()
	require.NotNil(t, top, "no dialog shown")
	var s []string
	for _, o := range test.LaidOutObjects(top) {
		if x, ok := o.(*widget.Label); ok {
			s = append(s, x.Text)
		}
	}
	return s
}

func TestNewWindow(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
	c := factory.CreateCharacterFull()

	t.Run("should return error when character is missing", func(t *testing.T) {
		_, err := NewWindow(u, nil, New, nil)
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
	t.Run("should return error when mode is undefined", func(t *testing.T) {
		_, err := NewWindow(u, c, Undefined, nil)
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
	t.Run("should return error when mail is missing for reply", func(t *testing.T) {
		_, err := NewWindow(u, c, Reply, nil)
		assert.ErrorIs(t, err, app.ErrInvalid)
	})
	t.Run("can render window", func(t *testing.T) {
		// given
		other := factory.CreateEveEntityCharacter()
		m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			RecipientIDs: []int64{other.ID},
			Subject:      optional.New("Hello"),
		})
		// when
		w, err := NewWindow(u, c, ReplyAll, m)
		// then
		require.NoError(t, err)
		defer w.Close()
		w.Show()
		a := w.Content().(*mailer)
		objs := test.LaidOutObjects(w.Content())
		var badges int
		for _, o := range objs {
			if _, ok := o.(*eveEntityBadge); ok {
				badges++
			}
		}
		assert.Equal(t, 3, badges) // from + sender + other recipient
		assert.Contains(t, objs, fyne.CanvasObject(a.subjectEntry))
		assert.Contains(t, objs, fyne.CanvasObject(a.bodyEntry))
		assert.Contains(t, objs, fyne.CanvasObject(a.sendButton))
	})
}

func TestMailer_Modes(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
	c := factory.CreateCharacterFull()
	self := factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
	other := factory.CreateEveEntityCharacter()
	m := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
		CharacterID:  c.ID,
		RecipientIDs: []int64{self.ID, other.ID},
		Subject:      optional.New("Hello"),
	})

	t.Run("new mail starts empty", func(t *testing.T) {
		a := newMailer(u, c, New, nil, test.NewWindow(nil))
		assert.True(t, a.toEntry.IsEmpty())
		assert.Empty(t, a.subjectEntry.Text)
		assert.Empty(t, a.bodyEntry.Text)
	})
	t.Run("from is the character and can not be changed", func(t *testing.T) {
		a := newMailer(u, c, New, nil, test.NewWindow(nil))
		assert.Equal(t, []*app.EveEntity{{ID: c.ID, Name: c.EveCharacter.Name, Category: app.EveEntityCharacter}}, a.fromEntry.Items())
		assert.True(t, a.fromEntry.Disabled())
	})
	t.Run("reply is addressed to sender", func(t *testing.T) {
		a := newMailer(u, c, Reply, m, test.NewWindow(nil))
		assert.Equal(t, []*app.EveEntity{m.From}, a.toEntry.Items())
		assert.Equal(t, "Re: Hello", a.subjectEntry.Text)
		assert.Contains(t, a.bodyEntry.Text, m.String())
	})
	t.Run("reply all is addressed to sender and other recipients", func(t *testing.T) {
		a := newMailer(u, c, ReplyAll, m, test.NewWindow(nil))
		var ids []int64
		for _, o := range a.toEntry.Items() {
			ids = append(ids, o.ID)
		}
		assert.ElementsMatch(t, []int64{m.From.ID, other.ID}, ids)
		assert.Equal(t, "Re: Hello", a.subjectEntry.Text)
		assert.Contains(t, a.bodyEntry.Text, m.String())
	})
	t.Run("reply all to own mail is addressed to other recipients only", func(t *testing.T) {
		m2 := factory.CreateCharacterMail(storage.CreateCharacterMailParams{
			CharacterID:  c.ID,
			FromID:       self.ID,
			RecipientIDs: []int64{other.ID},
		})
		a := newMailer(u, c, ReplyAll, m2, test.NewWindow(nil))
		assert.Equal(t, []*app.EveEntity{other}, a.toEntry.Items())
	})
	t.Run("forward has no recipients", func(t *testing.T) {
		a := newMailer(u, c, Forward, m, test.NewWindow(nil))
		assert.True(t, a.toEntry.IsEmpty())
		assert.Equal(t, "Fw: Hello", a.subjectEntry.Text)
		assert.Contains(t, a.bodyEntry.Text, m.String())
	})
}

func TestMailer_Send(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	t.Run("can send mail", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		var snackbars []string
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:                 test.NewTempApp(t),
			Storage:             st,
			Go:                  func(job func(ctx context.Context)) { job(t.Context()) },
			DisplaySnackbarFunc: func(s string) { snackbars = append(snackbars, s) },
		})
		var sections []app.CharacterSectionUpdated
		u.Signals().CharacterSectionChanged.AddListener(func(_ context.Context, arg app.CharacterSectionUpdated) {
			sections = append(sections, arg)
		})
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail", c.ID),
			httpmock.NewJsonResponderOrPanic(201, 123),
		)
		w := test.NewWindow(nil)
		var isClosed bool
		w.SetOnClosed(func() { isClosed = true })
		a := newMailer(u, c, New, nil, w)
		a.toEntry.Set([]*app.EveEntity{factory.CreateEveEntityCharacter(app.EveEntity{Name: "Bruce Wayne"})})
		a.subjectEntry.SetText("subject")
		a.bodyEntry.SetText("body")
		var isDone bool
		// when
		a.sendButton.OnAction(func() { isDone = true })
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		assert.True(t, isDone)
		assert.True(t, isClosed)
		assert.Equal(t, []string{"Your mail to Bruce Wayne has been sent."}, snackbars)
		assert.Equal(t, []app.CharacterSectionUpdated{{
			CharacterID: c.ID,
			Section:     app.SectionCharacterMailHeaders,
		}}, sections)
	})
	t.Run("does not send incomplete mail", func(t *testing.T) {
		cases := []struct {
			name       string
			recipients bool
			subject    string
			body       string
			want       string
		}{
			{"no recipient", false, "subject", "body", "Needs to have at least one recipient."},
			{"no subject", true, "", "body", "Subject can not be empty"},
			{"no body", true, "subject", "", "Message can not be empty"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// given
				testutil.MustTruncateTables(db)
				httpmock.Reset()
				u := testdouble.NewUIFake(testdouble.UIParams{
					App:     test.NewTempApp(t),
					Storage: st,
					Go:      func(job func(ctx context.Context)) { job(t.Context()) },
				})
				c := factory.CreateCharacterFull()
				factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
				httpmock.RegisterResponder(
					"POST",
					fmt.Sprintf("https://esi.evetech.net/characters/%d/mail", c.ID),
					httpmock.NewJsonResponderOrPanic(201, 123),
				)
				w := test.NewWindow(nil)
				defer w.Close()
				a := newMailer(u, c, New, nil, w)
				if tc.recipients {
					a.toEntry.Set([]*app.EveEntity{factory.CreateEveEntityCharacter()})
				}
				a.subjectEntry.SetText(tc.subject)
				a.bodyEntry.SetText(tc.body)
				var isDone bool
				// when
				a.sendButton.OnAction(func() { isDone = true })
				// then
				assert.Zero(t, httpmock.GetTotalCallCount())
				assert.True(t, isDone)
				assert.Contains(t, dialogTexts(t, w), tc.want)
			})
		}
	})
	t.Run("shows error when sending fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		var snackbars []string
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:                 test.NewTempApp(t),
			Storage:             st,
			Go:                  func(job func(ctx context.Context)) { job(t.Context()) },
			DisplaySnackbarFunc: func(s string) { snackbars = append(snackbars, s) },
		})
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		httpmock.RegisterResponder(
			"POST",
			fmt.Sprintf("https://esi.evetech.net/characters/%d/mail", c.ID),
			httpmock.NewStringResponder(400, `{"error":"bad request"}`),
		)
		w := test.NewWindow(nil)
		defer w.Close()
		var isClosed bool
		w.SetOnClosed(func() { isClosed = true })
		a := newMailer(u, c, New, nil, w)
		a.toEntry.Set([]*app.EveEntity{factory.CreateEveEntityCharacter()})
		a.subjectEntry.SetText("subject")
		a.bodyEntry.SetText("body")
		var isDone bool
		// when
		a.sendButton.OnAction(func() { isDone = true })
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		assert.True(t, isDone)
		assert.False(t, isClosed)
		assert.Empty(t, snackbars)
		assert.Contains(t, dialogTexts(t, w), "Failed to send mail")
	})
	t.Run("does nothing more when UI is gone", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		var snackbars []string
		ctx, cancel := context.WithCancel(t.Context())
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
			Go: func(job func(ctx context.Context)) {
				cancel()
				job(ctx)
			},
			DisplaySnackbarFunc: func(s string) { snackbars = append(snackbars, s) },
		})
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		w := test.NewWindow(nil)
		defer w.Close()
		var isClosed bool
		w.SetOnClosed(func() { isClosed = true })
		a := newMailer(u, c, New, nil, w)
		a.toEntry.Set([]*app.EveEntity{factory.CreateEveEntityCharacter()})
		a.subjectEntry.SetText("subject")
		a.bodyEntry.SetText("body")
		var isDone bool
		// when
		a.sendButton.OnAction(func() { isDone = true })
		// then
		assert.False(t, isDone)
		assert.False(t, isClosed)
		assert.Empty(t, snackbars)
	})
}

func TestMailer_ToButton(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()

	t.Run("shows information when offline", func(t *testing.T) {
		// given
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})
		c := factory.CreateCharacterFull()
		w := test.NewWindow(nil)
		defer w.Close()
		a := newMailer(u, c, New, nil, w)
		// when
		test.Tap(a.toButton)
		// then
		assert.Contains(t, dialogTexts(t, w), "Search not available while offline")
	})
	t.Run("opens add dialog when online", func(t *testing.T) {
		// given
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacterFull()
		w := test.NewWindow(nil)
		defer w.Close()
		a := newMailer(u, c, New, nil, w)
		// when
		test.Tap(a.toButton)
		// then
		assert.Contains(t, dialogTexts(t, w), "Add Recipient")
	})
}

func TestAddDialog(t *testing.T) {
	db, st, factory := testutil.NewDBInMemory()
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	t.Run("does not search with less than 3 characters", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{Name: "Bruce Wayne"})
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{}))
		w := test.NewWindow(nil)
		defer w.Close()
		d := newAddDialog(u, c.ID, func(_ *app.EveEntity) {}, w)
		// when
		d.searchEntry.SetText("Br")
		// then
		assert.Empty(t, d.results)
		assert.Zero(t, httpmock.GetTotalCallCount())
	})
	t.Run("clears results when search becomes too short", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{Name: "Bruce Wayne"})
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{}))
		w := test.NewWindow(nil)
		defer w.Close()
		d := newAddDialog(u, c.ID, func(_ *app.EveEntity) {}, w)
		d.searchEntry.SetText("Bru")
		require.Len(t, d.results, 1)
		// when
		d.searchEntry.SetText("Br")
		// then
		assert.Empty(t, d.results)
	})
	t.Run("shows known entities matching the search", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		e := factory.CreateEveEntityCharacter(app.EveEntity{Name: "Bruce Wayne"})
		factory.CreateEveEntityCharacter(app.EveEntity{Name: "Clark Kent"})
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{}))
		w := test.NewWindow(nil)
		defer w.Close()
		d := newAddDialog(u, c.ID, func(_ *app.EveEntity) {}, w)
		// when
		d.searchEntry.SetText("Bru")
		// then
		assert.Equal(t, []*app.EveEntity{e}, d.results)
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
	})
	t.Run("shows entities newly found on ESI", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{
			"character": {3007},
		}))
		httpmock.RegisterResponder(
			"POST",
			"https://esi.evetech.net/universe/names",
			httpmock.NewJsonResponderOrPanic(200, []map[string]any{{
				"id":       3007,
				"name":     "Bruce Wayne",
				"category": "character",
			}}),
		)
		w := test.NewWindow(nil)
		defer w.Close()
		d := newAddDialog(u, c.ID, func(_ *app.EveEntity) {}, w)
		// when
		d.searchEntry.SetText("Bru")
		// then
		require.Len(t, d.results, 1)
		assert.EqualValues(t, 3007, d.results[0].ID)
		assert.Equal(t, "Bruce Wayne", d.results[0].Name)
	})
	t.Run("shows error when ESI search fails", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewStringResponder(400, `{"error":"bad request"}`))
		w := test.NewWindow(nil)
		defer w.Close()
		d := newAddDialog(u, c.ID, func(_ *app.EveEntity) {}, w)
		// when
		d.searchEntry.SetText("Bru")
		// then
		assert.Contains(t, dialogTexts(t, w), "Search for 'Bru' failed")
	})
	t.Run("selecting a result reports it and closes the dialog", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		e := factory.CreateEveEntityCharacter(app.EveEntity{Name: "Bruce Wayne"})
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{}))
		w := test.NewWindow(nil)
		defer w.Close()
		var selected []*app.EveEntity
		d := newAddDialog(u, c.ID, func(ee *app.EveEntity) {
			selected = append(selected, ee)
		}, w)
		var isClosed bool
		d.onClose = func() { isClosed = true }
		d.searchEntry.SetText("Bru")
		// when
		d.list.Select(0)
		// then
		assert.Equal(t, []*app.EveEntity{e}, selected)
		assert.True(t, isClosed)
	})
	t.Run("cancel closes the dialog", func(t *testing.T) {
		// given
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		w := test.NewWindow(nil)
		defer w.Close()
		var selected []*app.EveEntity
		d := newAddDialog(u, 42, func(ee *app.EveEntity) {
			selected = append(selected, ee)
		}, w)
		var isClosed bool
		d.onClose = func() { isClosed = true }
		// when
		test.Tap(d.cancelButton)
		// then
		assert.True(t, isClosed)
		assert.Empty(t, selected)
	})
	t.Run("can render dialog with results", func(t *testing.T) {
		// given
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), IsOnline: true, Storage: st})
		c := factory.CreateCharacter()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{Name: "Bruce Wayne"})
		httpmock.RegisterResponder("GET", searchURL, httpmock.NewJsonResponderOrPanic(200, map[string][]int64{}))
		d := newAddDialog(u, c.ID, func(_ *app.EveEntity) {}, test.NewWindow(nil))
		w := test.NewWindow(d)
		defer w.Close()
		w.Resize(fyne.NewSize(400, 300))
		// when
		d.searchEntry.SetText("Bru")
		// then
		var names []string
		for _, o := range test.LaidOutObjects(d) {
			if x, ok := o.(*entityItem); ok && x.name.Text != "" {
				names = append(names, x.name.Text)
			}
		}
		assert.Equal(t, []string{"Bruce Wayne"}, names)
	})
}
