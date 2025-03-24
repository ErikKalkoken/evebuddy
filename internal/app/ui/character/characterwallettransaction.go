package character

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type CharacterWalletTransaction struct {
	widget.BaseWidget

	rows []*app.CharacterWalletTransaction
	body fyne.CanvasObject
	top  *widget.Label
	u    app.UI
}

func NewCharacterWalletTransaction(u app.UI) *CharacterWalletTransaction {
	a := &CharacterWalletTransaction{
		top:  appwidget.MakeTopLabel(),
		rows: make([]*app.CharacterWalletTransaction, 0),
		u:    u,
	}
	a.ExtendBaseWidget(a)
	makeDataLabel := func(col int, r *app.CharacterWalletTransaction) (string, fyne.TextAlign, widget.Importance) {
		var align fyne.TextAlign
		var importance widget.Importance
		var text string
		switch col {
		case 0:
			text = r.Date.Format(app.DateTimeFormat)
		case 1:
			align = fyne.TextAlignTrailing
			text = humanize.Comma(int64(r.Quantity))
		case 2:
			text = r.EveType.Name
		case 3:
			align = fyne.TextAlignTrailing
			text = humanize.FormatFloat(app.FloatFormat, r.UnitPrice)
		case 4:
			total := r.UnitPrice * float64(r.Quantity)
			align = fyne.TextAlignTrailing
			text = humanize.FormatFloat(app.FloatFormat, total)
			switch {
			case total < 0:
				importance = widget.DangerImportance
			case total > 0:
				importance = widget.SuccessImportance
			default:
				importance = widget.MediumImportance
			}
		case 5:
			text = r.Client.Name
		case 6:
			text = r.Location.Name
		}
		return text, align, importance
	}
	headers := []iwidget.HeaderDef{
		{Text: "Date", Width: 150},
		{Text: "Quantity", Width: 130},
		{Text: "Type", Width: 200},
		{Text: "Unit Price", Width: 200},
		{Text: "Total", Width: 200},
		{Text: "Client", Width: 250},
		{Text: "Where", Width: 250},
	}
	if a.u.IsDesktop() {
		a.body = iwidget.MakeDataTableForDesktop(headers, &a.rows, makeDataLabel, func(column int, r *app.CharacterWalletTransaction) {
			switch column {
			case 2:
				a.u.ShowTypeInfoWindow(r.EveType.ID)
			case 5:
				a.u.ShowEveEntityInfoWindow(r.Client)
			case 6:
				a.u.ShowLocationInfoWindow(r.Location.ID)
			}
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile(headers, &a.rows, makeDataLabel, func(r *app.CharacterWalletTransaction) {
			a.u.ShowTypeInfoWindow(r.EveType.ID)
		})
	}
	return a
}

func (a *CharacterWalletTransaction) CreateRenderer() fyne.WidgetRenderer {
	top := container.NewVBox(a.top, widget.NewSeparator())
	c := container.NewBorder(top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterWalletTransaction) Update() {
	var t string
	var i widget.Importance
	if err := a.updateEntries(); err != nil {
		slog.Error("Failed to refresh wallet transaction UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeTopText()
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.body.Refresh()
}

func (a *CharacterWalletTransaction) makeTopText() (string, widget.Importance) {
	if !a.u.HasCharacter() {
		return "No character", widget.LowImportance
	}
	characterID := a.u.CurrentCharacterID()
	hasData := a.u.StatusCacheService().CharacterSectionExists(characterID, app.SectionWalletTransactions)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	t := humanize.Comma(int64(len(a.rows)))
	s := fmt.Sprintf("Entries: %s", t)
	return s, widget.MediumImportance
}

func (a *CharacterWalletTransaction) updateEntries() error {
	if !a.u.HasCharacter() {
		a.rows = make([]*app.CharacterWalletTransaction, 0)
		return nil
	}
	characterID := a.u.CurrentCharacterID()
	ww, err := a.u.CharacterService().ListCharacterWalletTransactions(context.TODO(), characterID)
	if err != nil {
		return err
	}
	a.rows = ww
	return nil
}
