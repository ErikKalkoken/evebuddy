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
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	clockUpdateTicker     = 1 * time.Second
	esiStatusUpdateTicker = 60 * time.Second
)

type eveStatusLevel uint

const (
	eveStatusLevelUnknown eveStatusLevel = iota
	eveStatusLevelOK
	eveStatusLevelError
	eveStatusLevelWarn
)

type eveStatus struct {
	title string
	error string
	level eveStatusLevel
}

// statusArea is the UI area showing the current status aka status bar.
type statusArea struct {
	content                  *fyne.Container
	eveClock                 *widget.Label
	eveStatusTrafficResource fyne.Resource
	eveStatus                binding.Untyped
	eveStatusErrorMessage    string
	eveStatusGrid            *widget.GridWrap
	infoText                 *widget.Label
	infoPB                   *widget.ProgressBarInfinite
	ui                       *ui
}

func (u *ui) newStatusArea() *statusArea {
	a := &statusArea{
		eveClock:                 widget.NewLabel(""),
		eveStatus:                binding.NewUntyped(),
		eveStatusTrafficResource: theme.MediaRecordIcon(),
		eveStatusErrorMessage:    "Connecting...",
		infoText:                 widget.NewLabel(""),
		infoPB:                   widget.NewProgressBarInfinite(),
		ui:                       u,
	}
	a.infoPB.Hide()

	a.eveStatusGrid = widget.NewGridWrap(
		func() int {
			return 1
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.NewDisabledResource(a.eveStatusTrafficResource)),
				widget.NewLabel("999.999 players"),
			)
		},
		func(id widget.GridWrapItemID, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			x, err := a.eveStatus.Get()
			if err != nil {
				panic(err)
			}
			s := x.(eveStatus)

			label := row.Objects[1].(*widget.Label)
			label.SetText(s.title)

			var r fyne.Resource
			switch s.level {
			case eveStatusLevelOK:
				r = theme.NewSuccessThemedResource(a.eveStatusTrafficResource)
			case eveStatusLevelError:
				r = theme.NewErrorThemedResource(a.eveStatusTrafficResource)
			case eveStatusLevelWarn:
				r = theme.NewWarningThemedResource(a.eveStatusTrafficResource)
			default:
				r = theme.NewDisabledResource(a.eveStatusTrafficResource)
			}
			icon := row.Objects[0].(*widget.Icon)
			icon.SetResource(r)

		},
	)
	a.eveStatusGrid.OnSelected = func(id widget.GridWrapItemID) {
		x, err := a.eveStatus.Get()
		if err != nil {
			panic(err)
		}
		s := x.(eveStatus)

		var text string
		if s.error == "" {
			text = "No error detected"
		} else {
			text = s.error
		}
		d := dialog.NewInformation("ESI status", text, a.ui.window)
		d.SetOnClosed(func() {
			a.eveStatusGrid.UnselectAll()
		})
		d.Show()
	}

	a.eveStatus.Set(eveStatus{
		error: "Connecting...",
		title: "?",
	})

	c := container.NewHBox(
		container.NewHBox(a.infoText, a.infoPB),
		layout.NewSpacer(),
		widget.NewSeparator(),
		container.NewHBox(
			a.eveClock, layout.NewSpacer(), widget.NewSeparator(), a.eveStatusGrid))
	a.content = container.NewVBox(widget.NewSeparator(), c)
	return a
}

func (a *statusArea) StartUpdateTicker() {
	clockTicker := time.NewTicker(clockUpdateTicker)
	go func() {
		for {
			t := time.Now().UTC()
			a.eveClock.SetText(t.Format("15:04"))
			<-clockTicker.C
		}
	}()
	esiStatusTicker := time.NewTicker(esiStatusUpdateTicker)
	go func() {
		for {
			var s eveStatus
			x, err := a.ui.service.FetchESIStatus()
			if err != nil {
				slog.Error("Failed to fetch ESI status", "err", err)
				s.title = "ERROR"
				s.error = err.Error()
				s.level = eveStatusLevelError
			} else if !x.IsOK() {
				s.title = "OFFLINE"
				s.error = x.ErrorMessage
				s.level = eveStatusLevelWarn
			} else {
				arg := message.NewPrinter(language.English)
				s.title = arg.Sprintf("%d players", x.PlayerCount)
				s.error = ""
				s.level = eveStatusLevelOK
			}
			err = a.eveStatus.Set(s)
			if err != nil {
				panic(err)
			}
			a.eveStatusGrid.Refresh()
			<-esiStatusTicker.C
		}
	}()
}

func (s *statusArea) SetInfo(text string) {
	s.setInfo(text, widget.MediumImportance)
	s.infoPB.Stop()
	s.infoPB.Hide()
}

func (s *statusArea) SetInfoWithProgress(text string) {
	s.setInfo(text, widget.MediumImportance)
	s.infoPB.Start()
	s.infoPB.Show()
}

func (s *statusArea) SetError(text string) {
	s.setInfo(text, widget.DangerImportance)
	s.infoPB.Stop()
	s.infoPB.Hide()
}

func (s *statusArea) ClearInfo() {
	s.SetInfo("")
}

func (s *statusArea) setInfo(text string, importance widget.Importance) {
	s.infoText.Text = text
	s.infoText.Importance = importance
	s.infoText.Refresh()
}
