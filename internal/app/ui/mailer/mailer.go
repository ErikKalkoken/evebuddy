// Package mailer provides the functionality to create a new window
// for composing and sending mail.
package mailer

import (
	"context"
	"fmt"
	"log/slog"

	"slices"
	"strings"

	"sync/atomic"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"

	"fyne.io/fyne/v2/container"

	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

//go:generate go tool stringer -type=Mode

// Mode represents the mode in which mails are send.
type Mode uint

const (
	Undefined Mode = iota
	New
	Reply
	ReplyAll
	Forward
)

const (
	labelWith       = 45
	minSearchLength = 3
)

// Hooks that tests can replace.
var (
	runAsync = func(f func()) { go f() }
	// waitSearchDelay waits briefly, so typing triggers only one search.
	// Reports false when the search was canceled while waiting.
	waitSearchDelay = func(ctx context.Context) bool {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(300 * time.Millisecond):
			return true
		}
	}
	// searchTimeout limits the ESI requests of a search, as a dead connection can otherwise hang for minutes.
	searchTimeout = 15 * time.Second
)

type baseUI interface {
	Character() *characterservice.CharacterService
	ErrorDisplay(err error) string
	EVEImage() ui.EVEImageService
	EVEUniverse() *eveuniverseservice.EVEUniverseService
	Go(job func(ctx context.Context)) bool
	InfoViewer() ui.InfoViewer
	IsDeveloperMode() bool
	IsMobile() bool
	IsOffline() bool
	MainWindow() fyne.Window
	MakeWindowTitle(parts ...string) string
	Settings() *settings.Settings
	DisplaySnackbar(text string)
	Signals() *app.Signals
}

// NewWindow creates and returns a new Fyne window for composing and sending mail.
func NewWindow(u baseUI, c *app.Character, mode Mode, mail *app.CharacterMail) (fyne.Window, error) {
	if c == nil {
		return nil, fmt.Errorf("missing character: %w", app.ErrInvalid)
	}
	if mode == Undefined {
		return nil, fmt.Errorf("undefined mode: %w", app.ErrInvalid)
	}
	if mail == nil && mode != New {
		return nil, fmt.Errorf("missing mail for mode %s: %w", mode, app.ErrInvalid)
	}

	title := fmt.Sprintf("New message [%s]", c.EveCharacter.Name)
	w := fyne.CurrentApp().NewWindow(u.MakeWindowTitle(title))
	a := newMailer(u, c, mode, mail, w)
	w.SetOnClosed(a.cancel)
	w.SetContent(a)
	w.Resize(fyne.NewSize(600, 500))
	return w, nil
}

type mailer struct {
	widget.BaseWidget

	bodyEntry    *widget.Entry
	cancel       context.CancelFunc
	character    atomic.Pointer[app.Character]
	ctx          context.Context // canceled when the window closes
	fromEntry    *eveEntityEntry
	sendButton   *kxwidget.LoadingButton
	subjectEntry *widget.Entry
	toButton     *widget.Button
	toEntry      *eveEntityEntry
	u            baseUI
	w            fyne.Window
}

func newMailer(u baseUI, c *app.Character, mode Mode, mail *app.CharacterMail, w fyne.Window) *mailer {
	a := &mailer{
		u: u,
		w: w,
	}
	a.ctx, a.cancel = context.WithCancel(context.Background())
	a.character.Store(c)
	a.ExtendBaseWidget(a)

	a.fromEntry = newEveEntityEntry(widget.NewLabel("From"), labelWith, u.EVEImage().EveEntityLogoAsync)
	a.fromEntry.showInfo = u.InfoViewer().Show
	a.fromEntry.Set([]*app.EveEntity{{
		ID:       c.ID,
		Name:     c.EveCharacter.Name,
		Category: app.EveEntityCharacter,
	}})
	a.fromEntry.Disable()

	a.toButton = widget.NewButton("To", func() {
		if a.u.IsOffline() {
			ui.ShowInformation("OFFLINE", "Search not available while offline", a.w)
			return
		}
		showAddDialog(a.ctx, u, c.ID, func(ee *app.EveEntity) {
			a.toEntry.Add(ee)
		}, a.w)
	})
	a.toEntry = newEveEntityEntry(a.toButton, labelWith, u.EVEImage().EveEntityLogoAsync)
	a.toEntry.showInfo = u.InfoViewer().Show
	a.toEntry.placeholderText = "Tap To-Button to add recipients..."

	a.subjectEntry = widget.NewEntry()
	a.subjectEntry.PlaceHolder = "Subject"

	a.bodyEntry = widget.NewEntry()
	a.bodyEntry.MultiLine = true
	a.bodyEntry.Wrapping = fyne.TextWrapWord
	a.bodyEntry.SetMinRowsVisible(14)
	a.bodyEntry.PlaceHolder = "Compose message"

	const sep = "\n\n--------------------------------\n"
	switch mode {
	case New:
		// nothing to do
	case Reply:
		a.toEntry.Set([]*app.EveEntity{mail.From})
		a.subjectEntry.SetText(fmt.Sprintf("Re: %s", mail.Subject))
		a.bodyEntry.SetText(sep + mail.String())
	case ReplyAll:
		oo := slices.Concat([]*app.EveEntity{mail.From}, mail.Recipients)
		oo = slices.DeleteFunc(oo, func(o *app.EveEntity) bool {
			return o.ID == c.EveCharacter.ID
		})
		a.toEntry.Set(oo)
		a.subjectEntry.SetText(fmt.Sprintf("Re: %s", mail.Subject))
		a.bodyEntry.SetText(sep + mail.String())
	case Forward:
		a.subjectEntry.SetText(fmt.Sprintf("Fw: %s", mail.Subject))
		a.bodyEntry.SetText(sep + mail.String())
	default:
		panic(fmt.Errorf("unexpected mailer mode: %v", mode))
	}

	a.sendButton = kxwidget.NewLoadingButton("Send", theme.MailSendIcon(), func(done func()) {
		// TODO: Convert to dynamic enable/disable of send button
		var issue string
		if a.toEntry.IsEmpty() {
			issue = "Needs to have at least one recipient."
		}
		if a.subjectEntry.Text == "" {
			issue = "Subject can not be empty"
		}
		if a.bodyEntry.Text == "" {
			issue = "Message can not be empty"
		}
		if issue != "" {
			ui.ShowInformation("Incomplete mail", issue, w)
			done()
			return
		}
		characterID := a.character.Load().ID
		subject := a.subjectEntry.Text
		body := a.bodyEntry.Text
		recipients := slices.Clone(a.toEntry.Items())
		to := a.toEntry.String()
		ok := a.u.Go(func(ctx context.Context) {
			err := a.send(ctx, characterID, subject, recipients, body)
			if ctx.Err() != nil {
				return // UI is gone, so skip done()
			}
			defer done()
			if err != nil {
				fyne.Do(func() {
					ui.ShowErrorAndLog("Failed to send mail", err, a.u.IsDeveloperMode(), w)
				})
				return
			}
			fyne.Do(func() {
				w.Close()
			})
			a.u.DisplaySnackbar(fmt.Sprintf("Your mail to %s has been sent.", to))
		})
		if !ok {
			done()
		}
	})
	a.sendButton.SetImportance(widget.HighImportance)
	return a
}

func (a *mailer) CreateRenderer() fyne.WidgetRenderer {
	inner := container.NewBorder(
		container.NewVBox(a.fromEntry, a.toEntry, a.subjectEntry),
		nil,
		nil,
		nil,
		ui.NewWideScrollBarOverride(a.bodyEntry),
	)
	p := theme.Padding()
	c := container.NewBorder(
		nil,
		container.NewCenter(container.New(layout.NewCustomPaddedLayout(p, p, 0, 0),
			a.sendButton,
		)),
		nil,
		nil,
		inner,
	)

	return widget.NewSimpleRenderer(c)
}

// send sends a mail and reports any errors.
func (a *mailer) send(ctx context.Context, characterID int64, subject string, recipients []*app.EveEntity, body string) error {
	_, err := a.u.Character().SendMail(ctx, characterID, subject, recipients, body)
	if err != nil {
		return err
	}
	runAsync(func() {
		a.u.Signals().CharacterSectionChanged.Emit(ctx, app.CharacterSectionUpdated{
			CharacterID: characterID,
			Section:     app.SectionCharacterMailHeaders,
		})
	})
	return nil
}

func showAddDialog(ctx context.Context, u baseUI, characterID int64, onSelected func(ee *app.EveEntity), w fyne.Window) {
	d := newAddDialog(ctx, u, characterID, onSelected)
	modal := widget.NewModalPopUp(d, w.Canvas())
	d.onClose = modal.Hide
	_, s := w.Canvas().InteractiveArea()
	modal.Resize(fyne.NewSize(s.Width, s.Height))
	modal.Show()
	w.Canvas().Focus(d.searchEntry)
}

// addDialog is a widget for searching and adding a recipient.
type addDialog struct {
	widget.BaseWidget

	onClose func()

	cancelButton *widget.Button
	cancelSearch context.CancelFunc
	characterID  int64
	ctx          context.Context // parent of all searches
	list         *widget.List
	results      []*app.EveEntity
	searchEntry  *xwidget.SearchEntry
	spinner      *kxwidget.Spinner
	statusLabel  *widget.Label
	u            baseUI
}

func newAddDialog(ctx context.Context, u baseUI, characterID int64, onSelected func(ee *app.EveEntity)) *addDialog {
	d := &addDialog{
		characterID: characterID,
		ctx:         ctx,
		spinner:     kxwidget.NewSpinner(),
		statusLabel: widget.NewLabel(""),
		u:           u,
	}
	d.ExtendBaseWidget(d)
	d.spinner.Hide()
	d.statusLabel.Truncation = fyne.TextTruncateEllipsis
	d.list = widget.NewList(
		func() int {
			return len(d.results)
		},
		func() fyne.CanvasObject {
			return newEntityItem(u.EVEImage().EveEntityLogoAsync)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(d.results) {
				return
			}
			co.(*entityItem).set(d.results[id])
		},
	)
	d.list.HideSeparators = true
	d.list.OnSelected = func(id widget.ListItemID) {
		if id >= len(d.results) {
			d.list.UnselectAll()
			return
		}
		onSelected(d.results[id])
		d.close()
	}
	d.searchEntry = xwidget.NewSearchEntry("Type to start searching...", d.startSearch)
	d.cancelButton = widget.NewButton("Cancel", func() {
		d.close()
	})
	return d
}

func (d *addDialog) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewBorder(
			widget.NewLabel("Add Recipient"),
			nil,
			nil,
			d.cancelButton,
			d.searchEntry,
		),
		d.statusLabel,
		nil,
		nil,
		container.NewStack(
			d.list,
			container.NewCenter(container.NewStack(
				xwidget.NewSpacer(fyne.NewSquareSize(2*theme.IconInlineSize())),
				d.spinner,
			)),
		),
	)
	return widget.NewSimpleRenderer(c)
}

func (d *addDialog) close() {
	d.stopSearch()
	d.spinner.Stop()
	if d.onClose != nil {
		d.onClose()
	}
}

// startSearch starts a new search and cancels the previous one.
func (d *addDialog) startSearch(search string) {
	d.stopSearch()
	d.results = d.results[:0]
	d.list.Refresh()
	search = strings.TrimSpace(search)
	if utf8.RuneCountInString(search) < minSearchLength {
		d.spinner.Stop()
		d.spinner.Hide()
		d.setStatus("", widget.MediumImportance)
		return
	}
	d.setStatus("Searching...", widget.MediumImportance)
	d.spinner.Show()
	d.spinner.Start()
	ctx, cancel := context.WithCancel(d.ctx)
	d.cancelSearch = cancel
	runAsync(func() {
		d.doSearch(ctx, search)
	})
}

func (d *addDialog) stopSearch() {
	if d.cancelSearch == nil {
		return
	}
	d.cancelSearch()
	d.cancelSearch = nil
}

// doSearch runs a search. Results of a canceled search are discarded,
// so a slow search can't overwrite the results of a newer one.
func (d *addDialog) doSearch(ctx context.Context, search string) {
	if !waitSearchDelay(ctx) {
		return
	}
	esiCtx, cancel := context.WithTimeout(ctx, searchTimeout) // separate, so ctx only reports newer searches
	defer cancel()

	showError := func(timedOut bool, err error) {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Recipient search failed", "search", search, "error", err)
		var msg string
		if timedOut {
			msg = "search timed out"
		} else {
			msg = d.u.ErrorDisplay(err)
		}
		fyne.Do(func() {
			if ctx.Err() != nil {
				return
			}
			d.spinner.Stop()
			d.spinner.Hide()
			d.setStatus("ERROR: "+msg, widget.DangerImportance)
		})
	}

	err := d.u.Character().AddEveEntitiesFromSearchESI(esiCtx, d.characterID, search)
	if err != nil {
		timedOut := esiCtx.Err() != nil
		showError(timedOut, err)
		return
	}
	r, err := d.u.Character().ListMailRecipientsByPartialName(ctx, d.characterID, search)
	if err != nil {
		showError(false, err)
		return
	}
	fyne.Do(func() {
		if ctx.Err() != nil {
			return
		}
		d.spinner.Stop()
		d.spinner.Hide()
		d.results = r
		d.list.Refresh()
		if len(r) == 0 {
			d.setStatus("No results", widget.MediumImportance)
		} else {
			d.setStatus(ihumanize.Plural(len(r), "result", "results"), widget.MediumImportance)
		}
	})
}

func (d *addDialog) setStatus(text string, importance widget.Importance) {
	d.statusLabel.Importance = importance
	d.statusLabel.Text = text
	d.statusLabel.Refresh()
}
