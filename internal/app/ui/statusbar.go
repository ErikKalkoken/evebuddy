package ui

import (
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscache"
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	characterUpdateStatusTicker = 2 * time.Second
	clockUpdateTicker           = 2 * time.Second
	esiStatusUpdateTicker       = 60 * time.Second
)

// statusBarArea is the UI area showing the current status aka status bar.
type statusBarArea struct {
	content                   *fyne.Container
	characterUpdateStatusArea *characterUpdateStatusArea
	eveClock                  binding.String
	eveStatusArea             *eveStatusArea
	infoText                  *widget.Label
	ui                        *ui
}

func (u *ui) newStatusBarArea() *statusBarArea {
	a := &statusBarArea{
		infoText:                  widget.NewLabel(""),
		eveClock:                  binding.NewString(),
		eveStatusArea:             newEveStatusArea(u),
		characterUpdateStatusArea: newCharacterUpdateStatusArea(u),
		ui:                        u,
	}
	a.eveClock.Set(" 00:00 ")
	clock := widget.NewLabelWithData(a.eveClock)
	a.content = container.NewVBox(widget.NewSeparator(), container.NewHBox(
		a.infoText,
		layout.NewSpacer(),
		widget.NewSeparator(),
		a.characterUpdateStatusArea.content,
		widget.NewSeparator(),
		clock,
		widget.NewSeparator(),
		a.eveStatusArea.content,
	))
	return a
}

func (a *statusBarArea) StartUpdateTicker() {
	updateTicker := time.NewTicker(characterUpdateStatusTicker)
	go func() {
		for {
			a.characterUpdateStatusArea.refresh()
			<-updateTicker.C
		}
	}()
	clockTicker := time.NewTicker(clockUpdateTicker)
	go func() {
		for {
			t := time.Now().UTC().Format("15:04")
			a.eveClock.Set(t)
			<-clockTicker.C
		}
	}()
	esiStatusTicker := time.NewTicker(esiStatusUpdateTicker)
	go func() {
		for {
			x, err := a.ui.sv.ESIStatus.Fetch()
			var t, errorMessage string
			var s eveStatus
			if err != nil {
				slog.Error("Failed to fetch ESI status", "err", err)
				errorMessage = humanize.Error(err)
				s = eveStatusError
				t = "ERROR"
			} else if !x.IsOK() {
				errorMessage = x.ErrorMessage
				s = eveStatusOffline
				t = "OFFLINE"
			} else {
				arg := message.NewPrinter(language.English)
				t = arg.Sprintf("%d players", x.PlayerCount)
				s = eveStatusOnline
			}
			a.eveStatusArea.setStatus(s, t, errorMessage)
			<-esiStatusTicker.C
		}
	}()
}

func (s *statusBarArea) SetInfo(text string) {
	s.setInfo(text, widget.MediumImportance)
}

func (s *statusBarArea) SetError(text string) {
	s.setInfo(text, widget.DangerImportance)
}

func (s *statusBarArea) ClearInfo() {
	s.SetInfo("")
}

func (s *statusBarArea) setInfo(text string, importance widget.Importance) {
	s.infoText.Text = text
	s.infoText.Importance = importance
	s.infoText.Refresh()
}

type eveStatus uint

const (
	eveStatusUnknown eveStatus = iota
	eveStatusOnline
	eveStatusOffline
	eveStatusError
)

type eveStatusArea struct {
	content      *widget.GridWrap
	status       eveStatus
	title        string
	errorMessage string
	ui           *ui
}

func newEveStatusArea(u *ui) *eveStatusArea {
	a := &eveStatusArea{ui: u}
	a.content = widget.NewGridWrap(
		func() int {
			return 1
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.MediaRecordIcon()),
				widget.NewLabel("999.999 players"))
		},
		func(_ widget.GridWrapItemID, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			icon := row.Objects[0].(*widget.Icon)
			label := row.Objects[1].(*widget.Label)
			r1 := theme.MediaRecordIcon()
			var r2 fyne.Resource
			switch a.status {
			case eveStatusOnline:
				r2 = theme.NewSuccessThemedResource(r1)
			case eveStatusError:
				r2 = theme.NewErrorThemedResource(r1)
			case eveStatusOffline:
				r2 = theme.NewWarningThemedResource(r1)
			case eveStatusUnknown:
				r2 = theme.NewDisabledResource(r1)
			}
			icon.SetResource(r2)
			label.SetText(a.title)
		},
	)
	a.content.OnSelected = func(_ widget.GridWrapItemID) {
		var i widget.Importance
		var text string
		if a.errorMessage == "" {
			text = "No error detected"
			i = widget.MediumImportance
		} else {
			text = a.errorMessage
			i = widget.DangerImportance
		}
		lb := widget.NewLabel(text)
		lb.Wrapping = fyne.TextWrapWord
		lb.Importance = i
		d := dialog.NewCustom("ESI status", "OK", container.NewVScroll(lb), a.ui.window)
		d.SetOnClosed(func() {
			a.content.UnselectAll()
		})
		d.Show()
		d.Resize(fyne.Size{Width: 400, Height: 200})
	}
	return a
}

func (a *eveStatusArea) setStatus(status eveStatus, title, errorMessage string) {
	a.status = status
	a.title = title
	a.errorMessage = errorMessage
	a.content.Refresh()
}

type updateStatusOutput struct {
	status statuscache.Status
	title  string
}

type characterUpdateStatusArea struct {
	content *widget.GridWrap
	data    updateStatusOutput
	ui      *ui
}

func newCharacterUpdateStatusArea(u *ui) *characterUpdateStatusArea {
	a := &characterUpdateStatusArea{ui: u}
	a.content = widget.NewGridWrap(
		func() int {
			return 1
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				layout.NewSpacer(),
				widget.NewLabel("100% Fresh"),
				layout.NewSpacer(),
			)
		},
		func(_ widget.GridWrapItemID, co fyne.CanvasObject) {
			label := co.(*fyne.Container).Objects[1].(*widget.Label)
			label.Text = a.data.title
			label.Importance = status2widgetImportance(a.data.status)
			label.Refresh()
		},
	)
	a.content.OnSelected = func(_ widget.GridWrapItemID) {
		a.ui.showStatusWindow()
		a.content.UnselectAll()
	}
	return a
}

func (a *characterUpdateStatusArea) refresh() {
	ss := a.ui.sv.StatusCache.Summary()
	a.data = updateStatusOutput{status: ss.Status(), title: ss.Display()}
	a.content.Refresh()
}
