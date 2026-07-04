package characters

import (
	"context"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

// fakeInfoViewer records Show calls and implements ui.InfoViewer.
type fakeInfoViewer struct {
	called bool
	entity *app.EveEntity
}

func (f *fakeInfoViewer) Show(o *app.EveEntity) { f.called = true; f.entity = o }
func (f *fakeInfoViewer) ShowLocation(_ int64)  {}
func (f *fakeInfoViewer) ShowRace(_ int64)      {}
func (f *fakeInfoViewer) ShowType(_, _ int64)   {}

var _ ui.InfoViewer = (*fakeInfoViewer)(nil)

// fakeCharactersUI is a minimal baseUI implementation for tests in this package.
type fakeCharactersUI struct {
	infoViewer ui.InfoViewer
	eveImage   ui.EVEImageService
	signals    *app.Signals
	window     fyne.Window
}

func newFakeCharactersUI(a fyne.App) *fakeCharactersUI {
	return &fakeCharactersUI{
		eveImage:   testutil.NewEveImageServiceStub(),
		infoViewer: &fakeInfoViewer{},
		signals:    app.NewSignals(),
		window:     a.NewWindow(""),
	}
}

func (u *fakeCharactersUI) Character() *characterservice.CharacterService       { return nil }
func (u *fakeCharactersUI) Corporation() *corporationservice.CorporationService { return nil }
func (u *fakeCharactersUI) ErrorDisplay(err error) string                       { return err.Error() }
func (u *fakeCharactersUI) EVEImage() ui.EVEImageService                        { return u.eveImage }
func (u *fakeCharactersUI) EVEUniverse() *eveuniverseservice.EVEUniverseService { return nil }
func (u *fakeCharactersUI) GetOrCreateWindow(id string, titles ...string) (fyne.Window, bool) {
	return u.window, true
}
func (u *fakeCharactersUI) InfoViewer() ui.InfoViewer { return u.infoViewer }
func (u *fakeCharactersUI) IsDeveloperMode() bool     { return false }
func (u *fakeCharactersUI) IsMobile() bool            { return false }
func (u *fakeCharactersUI) IsOffline() bool           { return false }
func (u *fakeCharactersUI) IsUpdateDisabled() bool    { return false }
func (u *fakeCharactersUI) MainWindow() fyne.Window   { return u.window }
func (u *fakeCharactersUI) MakeWindowTitle(parts ...string) string {
	return strings.Join(parts, " - ")
}
func (u *fakeCharactersUI) Settings() *settings.Settings             { return nil }
func (u *fakeCharactersUI) ShowCharacter(_ context.Context, _ int64) {}
func (u *fakeCharactersUI) ShowSnackbar(_ string)                    {}
func (u *fakeCharactersUI) Signals() *app.Signals                    { return u.signals }
func (u *fakeCharactersUI) UpdateMailIndicator(_ context.Context)    {}

var _ baseUI = (*fakeCharactersUI)(nil)

// Note: these tests must live in package characters (not package infoviewer) because they
// access unexported fields that are only visible within this package:
//   - communicationDetail.header  (*MailHeader, unexported field of unexported type)
//   - mailDetail.header           (*MailHeader, unexported field of unexported type)
//   - MailHeader.showInfo         (unexported field)
// Moving them to another package would make those fields invisible and the tests
// would need to be rewritten using a weaker approach that cannot verify the captured
// method value directly.

// TestNewCommunications_ShowInfoCapturesInfoViewer verifies that the Show method bound
// from u.InfoViewer() at construction time is valid and callable. This is a regression
// test for the bug where u.iw in newBaseUI was nil when NewCommunications was called:
// the captured showInfo had a nil receiver and panicked the first time a notification
// sender or recipient icon was tapped.
func TestNewCommunications_ShowInfoCapturesInfoViewer(t *testing.T) {
	a := test.NewTempApp(t)
	u := newFakeCharactersUI(a)
	comms := NewCommunications(u)
	fakeIV := u.infoViewer.(*fakeInfoViewer)

	entity := &app.EveEntity{ID: 1, Name: "Tester", Category: app.EveEntityCharacter}
	// If u.InfoViewer() had returned nil at construction time, showInfo would be a
	// nil-receiver method value. Calling it here would panic.
	assert.NotPanics(t, func() {
		comms.Detail.header.showInfo(entity)
	})
	assert.True(t, fakeIV.called, "showInfo must invoke InfoViewer.Show")
	assert.Equal(t, entity, fakeIV.entity)
}

// TestNewMails_ShowInfoCapturesInfoViewer is the same regression test for NewMails.
func TestNewMails_ShowInfoCapturesInfoViewer(t *testing.T) {
	a := test.NewTempApp(t)
	u := newFakeCharactersUI(a)
	mails := NewMails(u)
	fakeIV := u.infoViewer.(*fakeInfoViewer)

	entity := &app.EveEntity{ID: 1, Name: "Tester", Category: app.EveEntityCharacter}
	assert.NotPanics(t, func() {
		mails.Detail.header.showInfo(entity)
	})
	assert.True(t, fakeIV.called, "showInfo must invoke InfoViewer.Show")
	assert.Equal(t, entity, fakeIV.entity)
}
