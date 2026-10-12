package mailer

import (
	"fmt"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

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

	type fixture struct {
		a         *mailer
		snackbars chan string
		url       string
	}
	setup := func(t *testing.T) fixture {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		snackbars := make(chan string, 1)
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:                 test.NewTempApp(t),
			Storage:             st,
			DisplaySnackbarFunc: func(s string) { snackbars <- s },
		})
		c := factory.CreateCharacterFull()
		factory.CreateCharacterToken(storage.UpdateOrCreateCharacterTokenParams{CharacterID: c.ID})
		factory.CreateEveEntityCharacter(app.EveEntity{ID: c.ID})
		a := newMailer(u, c, New, nil, test.NewWindow(nil))
		return fixture{
			a:         a,
			snackbars: snackbars,
			url:       fmt.Sprintf("https://esi.evetech.net/characters/%d/mail", c.ID),
		}
	}
	fill := func(a *mailer) *app.EveEntity {
		r := factory.CreateEveEntityCharacter(app.EveEntity{Name: "Bruce Wayne"})
		a.toEntry.Set([]*app.EveEntity{r})
		a.subjectEntry.SetText("subject")
		a.bodyEntry.SetText("body")
		return r
	}
	// tapSend runs the send action and returns a channel which is closed when done is called.
	tapSend := func(a *mailer) chan struct{} {
		done := make(chan struct{})
		a.sendButton.OnAction(func() { close(done) })
		return done
	}
	waitDone := func(t *testing.T, done chan struct{}) {
		t.Helper()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("send did not finish")
		}
	}

	t.Run("can send mail", func(t *testing.T) {
		f := setup(t)
		httpmock.RegisterResponder("POST", f.url, httpmock.NewJsonResponderOrPanic(201, 123))
		fill(f.a)
		// when
		waitDone(t, tapSend(f.a))
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		select {
		case s := <-f.snackbars:
			assert.Equal(t, "Your mail to Bruce Wayne has been sent.", s)
		case <-time.After(5 * time.Second):
			t.Fatal("no snackbar shown")
		}
	})
	t.Run("does not send incomplete mail", func(t *testing.T) {
		f := setup(t)
		httpmock.RegisterResponder("POST", f.url, httpmock.NewJsonResponderOrPanic(201, 123))
		// when
		waitDone(t, tapSend(f.a))
		// then
		assert.Zero(t, httpmock.GetTotalCallCount())
	})
	t.Run("does not report success when sending fails", func(t *testing.T) {
		f := setup(t)
		httpmock.RegisterResponder("POST", f.url, httpmock.NewStringResponder(400, `{"error":"bad request"}`))
		fill(f.a)
		// when
		waitDone(t, tapSend(f.a))
		// then
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		assert.Empty(t, f.snackbars)
	})
}
