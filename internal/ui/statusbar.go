package ui

import (
	"fmt"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/evebuddy/internal/model"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	characterUpdateStatusTicker = 5 * time.Second
	clockUpdateTicker           = 1 * time.Second
	esiStatusUpdateTicker       = 60 * time.Second
)

// statusBarArea is the UI area showing the current status aka status bar.
type statusBarArea struct {
	content     *fyne.Container
	statusItems binding.UntypedList
	grid        *widget.GridWrap
	infoText    *widget.Label
	ui          *ui
}

const (
	indexCharacterUpdateStatus = iota
	indexEveTime
	indexEveStatus
)

type widgetImportance uint

const (
	mediumImportance widgetImportance = iota
	dangerImportance
	disabledImportance
	successImportance
	warningImportance
)

type statusItem struct {
	details        string
	icon           fyne.Resource
	iconImportance widgetImportance
	importance     widgetImportance
	title          string
}

// type maxWidthLayout struct {
// }

// func (d *maxWidthLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
// 	w, h := float32(0), float32(0)
// 	for _, o := range objects {
// 		childSize := o.MinSize()

// 		w += childSize.Width
// 		h += childSize.Height
// 	}
// 	return fyne.NewSize(w, h)
// }

// func (d *maxWidthLayout) Layout(objects []fyne.CanvasObject, containerSize fyne.Size) {
// 	pos := fyne.NewPos(0, containerSize.Height-d.MinSize(objects).Height)
// 	for _, o := range objects {
// 		size := o.MinSize()
// 		o.Resize(size)
// 		o.Move(pos)

// 		pos = pos.Add(fyne.NewPos(size.Width, size.Height))
// 	}
// }

func (u *ui) newStatusBarArea() *statusBarArea {
	a := &statusBarArea{
		infoText:    widget.NewLabel(""),
		statusItems: binding.NewUntypedList(),
		ui:          u,
	}

	items := make([]statusItem, 3)
	items[indexCharacterUpdateStatus] = statusItem{}
	items[indexEveTime] = statusItem{}
	items[indexEveStatus] = statusItem{
		icon:           theme.MediaRecordIcon(),
		iconImportance: disabledImportance,
	}
	a.statusItems.Set(copyToUntypedSlice(items))

	a.grid = widget.NewGridWrapWithData(
		a.statusItems,
		func() fyne.CanvasObject {
			return container.NewHBox(
				layout.NewSpacer(),
				widget.NewIcon(theme.QuestionIcon()),
				widget.NewLabel("999.999 players"),
				layout.NewSpacer(),
			)
		},
		func(di binding.DataItem, co fyne.CanvasObject) {
			row := co.(*fyne.Container)
			s, err := convertDataItem[statusItem](di)
			if err != nil {
				panic(err)
			}
			icon := row.Objects[1].(*widget.Icon)
			label := row.Objects[2].(*widget.Label)
			var t string
			if s.title == "" {
				t = "..."
			} else {
				t = s.title
			}
			var i widget.Importance
			switch s.importance {
			case dangerImportance:
				i = widget.DangerImportance
			case successImportance:
				i = widget.SuccessImportance
			case warningImportance:
				i = widget.WarningImportance
			default:
				i = widget.MediumImportance
			}
			label.Text = t
			label.Importance = i
			label.Refresh()

			if s.icon == nil {
				icon.Hide()
				return
			}

			var r fyne.Resource
			switch s.iconImportance {
			case successImportance:
				r = theme.NewSuccessThemedResource(s.icon)
			case dangerImportance:
				r = theme.NewErrorThemedResource(s.icon)
			case warningImportance:
				r = theme.NewWarningThemedResource(s.icon)
			case disabledImportance:
				r = theme.NewDisabledResource(s.icon)
			default:
				r = s.icon
			}
			icon.SetResource(r)
			icon.Show()
		},
	)
	a.grid.OnSelected = func(id widget.GridWrapItemID) {
		s, err := getFromBoundUntypedList[statusItem](a.statusItems, id)
		if err != nil {
			panic(err)
		}
		switch id {
		case indexCharacterUpdateStatus:
			c := u.CurrentChar()
			if c != nil {
				a.ui.showStatusDialog(model.CharacterShort{ID: c.ID, Name: c.EveCharacter.Name})
			}
			a.grid.UnselectAll()
		case indexEveStatus:
			var text string
			if s.details == "" {
				text = "No error detected"
			} else {
				text = s.details
			}
			d := dialog.NewInformation("ESI status", text, a.ui.window)
			d.SetOnClosed(func() {
				a.grid.UnselectAll()
			})
			d.Show()
		default:
			a.grid.UnselectAll()
		}
	}

	// c := container.NewHBox(a.infoText, layout.NewSpacer(), container.New(&maxWidthLayout{}, a.grid))
	a.content = container.NewBorder(nil, nil, a.infoText, nil, a.grid)
	return a
}

func (a *statusBarArea) updateIfNeeded(index int, newItem statusItem) {
	x, err := a.statusItems.GetValue(indexEveTime)
	if err != nil {
		panic(err)
	}
	currentItem := x.(statusItem)
	if currentItem != newItem {
		a.statusItems.SetValue(index, newItem)
		a.grid.Refresh()
	}
}

func (a *statusBarArea) StartUpdateTicker() {
	updateTicker := time.NewTicker(characterUpdateStatusTicker)
	go func() {
		for {
			a.RefreshCharacterUpdateStatus()
			<-updateTicker.C
		}
	}()
	clockTicker := time.NewTicker(clockUpdateTicker)
	go func() {
		for {
			t := time.Now().UTC()
			s := statusItem{title: t.Format("15:04")}
			a.updateIfNeeded(indexEveTime, s)
			<-clockTicker.C
		}
	}()
	esiStatusTicker := time.NewTicker(esiStatusUpdateTicker)
	go func() {
		for {
			s := statusItem{icon: theme.MediaRecordIcon()}
			x, err := a.ui.service.FetchESIStatus()
			if err != nil {
				slog.Error("Failed to fetch ESI status", "err", err)
				s.title = "ERROR"
				s.details = err.Error()
				s.iconImportance = dangerImportance
			} else if !x.IsOK() {
				s.title = "OFFLINE"
				s.details = x.ErrorMessage
				s.iconImportance = warningImportance
			} else {
				arg := message.NewPrinter(language.English)
				s.title = arg.Sprintf("%d players", x.PlayerCount)
				s.iconImportance = successImportance
			}
			a.updateIfNeeded(indexEveStatus, s)
			<-esiStatusTicker.C
		}
	}()
}

func (a *statusBarArea) RefreshCharacterUpdateStatus() {
	characterID := a.ui.CurrentCharID()
	var title string
	i := mediumImportance
	if characterID == 0 {
		title = "No character"
		i = disabledImportance
	} else {
		data := newUpdateStatusList(a.ui.service, characterID)
		completed := 0
		hasError := false
		for _, d := range data {
			if !d.IsOK() {
				hasError = true
				break
			}
			if d.IsCurrent() {
				completed++
			}
		}
		if hasError {
			title = "ERROR"
			i = dangerImportance
		} else {
			p := float32(completed) / float32(len(data)) * 100
			if p == 100 {
				title = "OK"
				i = successImportance
			} else {
				title = fmt.Sprintf("Updating %.0f%%...", p)
			}
		}
	}
	s := statusItem{title: title, importance: i}
	a.updateIfNeeded(indexCharacterUpdateStatus, s)
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
