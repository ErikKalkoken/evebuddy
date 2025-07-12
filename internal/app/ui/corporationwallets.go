package ui

import (
	"context"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/dustin/go-humanize"
)

type corporationWalletRow struct {
	balance          float64
	balanceFormatted string
	corporationID    int32
	isTitle          bool
	isTotal          bool
	name             string
}

// corporationWallets shows the attributes for the current character.
type corporationWallets struct {
	widget.BaseWidget

	rows []corporationWalletRow
	list *widget.List
	top  *widget.Label
	u    *baseUI
}

func newCorporationWallets(u *baseUI) *corporationWallets {
	w := &corporationWallets{
		rows: make([]corporationWalletRow, 0),
		top:  makeTopLabel(),
		u:    u,
	}
	w.list = w.makeAttributeList()
	w.ExtendBaseWidget(w)
	return w
}

func (a *corporationWallets) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.list)
	return widget.NewSimpleRenderer(c)
}

func (a *corporationWallets) makeAttributeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rows)
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("Template")
			balance := widget.NewLabel("Template")
			return container.NewHBox(name, layout.NewSpacer(), balance)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rows) {
				return
			}
			r := a.rows[id]

			hbox := co.(*fyne.Container).Objects

			name := hbox[0].(*widget.Label)
			name.Text = r.name
			name.TextStyle.Bold = r.isTotal
			name.Refresh()

			balance := hbox[2].(*widget.Label)
			balance.Text = r.balanceFormatted
			balance.TextStyle.Bold = r.isTotal
			balance.Refresh()
		})

	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	l.HideSeparators = true
	return l
}

func (a *corporationWallets) update() {
	var err error
	rows := make([]corporationWalletRow, 0)
	corporationID := a.u.currentCorporationID()
	hasData := a.u.scs.HasCorporationSection(corporationID, app.SectionCorporationWalletBalances)
	if hasData {
		rows2, err2 := a.fetchData(a.u.currentCorporationID(), a.u.services())
		if err2 != nil {
			slog.Error("Failed to refresh corporation wallets UI", "err", err)
			err = err2
		} else {
			rows = rows2
		}
	}
	t, i := a.u.makeTopText(corporationID, hasData, err, func() (string, widget.Importance) {
		return "", widget.MediumImportance
	})
	fyne.Do(func() {
		if t != "" {
			a.top.Text, a.top.Importance = t, i
			a.top.Refresh()
			a.top.Show()
		} else {
			a.top.Hide()
		}
	})
	fyne.Do(func() {
		a.rows = rows
		a.list.Refresh()
	})
}

func (*corporationWallets) fetchData(corporationID int32, s services) ([]corporationWalletRow, error) {
	rows := make([]corporationWalletRow, 0)
	if corporationID == 0 {
		return rows, nil
	}
	oo, err := s.rs.ListCorporationWalletBalances(context.Background(), corporationID)
	if err != nil {
		return nil, err
	}
	formatBalance := func(f float64) string {
		return humanize.FormatFloat(app.FloatFormat, f)
	}
	var total float64
	for _, o := range oo {
		rows = append(rows, corporationWalletRow{
			corporationID:    corporationID,
			balance:          o.Balance,
			balanceFormatted: formatBalance(o.Balance),
			name:             o.Name,
		})
		total += o.Balance
	}
	rows = append(rows, corporationWalletRow{
		corporationID:    corporationID,
		balance:          total,
		balanceFormatted: formatBalance(total),
		name:             "Total",
		isTotal:          true,
	})
	return rows, nil
}
