// Package mailer provides the functionality to create a new window
// for composing and sending mail.
package mailer

import (
	"context"
	"fmt"

	"slices"

	"sync/atomic"

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

const labelWith = 45

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
	w.SetContent(a)
	w.Resize(fyne.NewSize(600, 500))
	return w, nil
}

type mailer struct {
	widget.BaseWidget

	bodyEntry    *widget.Entry
	character    atomic.Pointer[app.Character]
	fromEntry    *eveEntityEntry
	sendButton   *kxwidget.LoadingButton
	subjectEntry *widget.Entry
	toEntry      *eveEntityEntry
	u            baseUI
	w            fyne.Window
}

func newMailer(u baseUI, c *app.Character, mode Mode, mail *app.CharacterMail, w fyne.Window) *mailer {
	a := &mailer{
		u: u,
		w: w,
	}
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

	toButton := widget.NewButton("To", func() {
		if a.u.IsOffline() {
			ui.ShowInformation("OFFLINE", "Search not available while offline", a.w)
			return
		}
		showAddDialog(u, c.ID, func(ee *app.EveEntity) {
			a.toEntry.Add(ee)
		}, a.w)
	})
	a.toEntry = newEveEntityEntry(toButton, labelWith, u.EVEImage().EveEntityLogoAsync)
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
	go a.u.Signals().CharacterSectionChanged.Emit(ctx, app.CharacterSectionUpdated{
		CharacterID: characterID,
		Section:     app.SectionCharacterMailHeaders,
	})
	return nil
}

func showAddDialog(u baseUI, characterID int64, onSelected func(ee *app.EveEntity), w fyne.Window) {
	var modal *widget.PopUp
	var results []*app.EveEntity
	list := widget.NewList(
		func() int {
			return len(results)
		},
		func() fyne.CanvasObject {
			return newEntityItem(u.EVEImage().EveEntityLogoAsync)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(results) {
				return
			}
			co.(*entityItem).set(results[id])
		},
	)
	list.HideSeparators = true
	list.OnSelected = func(id widget.ListItemID) {
		if id >= len(results) {
			list.UnselectAll()
			return
		}
		onSelected(results[id])
		modal.Hide()
	}
	showErrorDialog := func(search string, err error) {
		fyne.Do(func() {
			ui.ShowErrorAndLog("Search for '"+search+"' failed", err, u.IsDeveloperMode(), w)
		})
	}
	entry := xwidget.NewSearchEntry("Type to start searching...", func(search string) {
		if len(search) < 3 {
			results = results[:0]
			list.Refresh()
			return
		}
		ctx := context.Background()
		go func() {
			r, err := u.EVEUniverse().ListEntitiesByPartialName(ctx, search)
			if err != nil {
				showErrorDialog(search, err)
				return
			}
			fyne.Do(func() {
				results = r
				list.Refresh()
			})
		}()
		go func() {
			missingIDs, err := u.Character().AddEveEntitiesFromSearchESI(
				ctx,
				characterID,
				search,
			)
			if err != nil {
				showErrorDialog(search, err)
				return
			}
			if missingIDs.Size() == 0 {
				return // no need to update when not changed
			}
			r, err := u.EVEUniverse().ListEntitiesByPartialName(ctx, search)
			if err != nil {
				showErrorDialog(search, err)
				return
			}
			fyne.Do(func() {
				results = r
				list.Refresh()
			})
		}()
	})
	c := container.NewBorder(
		container.NewBorder(
			widget.NewLabel("Add Recipient"),
			nil,
			nil,
			widget.NewButton("Cancel", func() {
				modal.Hide()
			}),
			entry,
		),
		nil,
		nil,
		nil,
		list,
	)
	modal = widget.NewModalPopUp(c, w.Canvas())
	_, s := w.Canvas().InteractiveArea()
	modal.Resize(fyne.NewSize(s.Width, s.Height))
	modal.Show()
	w.Canvas().Focus(entry)
}
