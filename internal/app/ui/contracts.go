package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type contractEntry struct {
	from     string
	info     string
	issued   time.Time
	accepted optional.Optional[time.Time]
	name     string
	expired  time.Time
	status   string
	to       string
	type_    string
}

// func (e contractEntry) refTypeOutput() string {
// 	s := strings.ReplaceAll(e.refType, "_", " ")
// 	c := cases.Title(language.English)
// 	s = c.String(s)
// 	return s
// }

// contractsArea is the UI area that shows the skillqueue
type contractsArea struct {
	content   *fyne.Container
	contracts []*app.CharacterContract
	table     *widget.Table
	top       *widget.Label
	u         *UI
}

func (u *UI) newContractsArea() *contractsArea {
	a := contractsArea{
		contracts: make([]*app.CharacterContract, 0),
		top:       widget.NewLabel(""),
		u:         u,
	}

	a.top.TextStyle.Bold = true
	a.table = a.makeTable()
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.content = container.NewBorder(top, nil, nil, nil, a.table)
	return &a
}

func (a *contractsArea) makeTable() *widget.Table {
	var headers = []struct {
		text  string
		width float32
	}{
		{"Contract", 200},
		{"Type", 120},
		{"From", 150},
		{"To", 150},
		{"Status", 100},
		{"Date Issued", 150},
		{"Date Accepted", 150},
		{"Time Left", 150},
	}
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.contracts), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Template")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			l.Importance = widget.MediumImportance
			l.Alignment = fyne.TextAlignLeading
			l.Truncation = fyne.TextTruncateOff
			if tci.Row >= len(a.contracts) || tci.Row < 0 {
				return
			}
			o := a.contracts[tci.Row]
			switch tci.Col {
			case 0:
				l.Text = o.NameDisplay()
			case 1:
				l.Text = o.TypeDisplay()
			case 2:
				l.Text = o.Issuer.Name
			case 3:
				l.Text = o.Assignee.Name
			case 4:
				l.Text = o.StatusDisplay()
			case 5:
				l.Text = o.DateIssued.Format(app.TimeDefaultFormat)
			case 6:
				if o.DateAccepted.IsEmpty() {
					l.Text = ""
				} else {
					l.Text = o.DateAccepted.MustValue().Format(app.TimeDefaultFormat)
				}
			case 7:
				x := o.DateExpiredEffective()
				if x.Before(time.Now()) {
					l.Text = "EXPIRED"
					l.Importance = widget.DangerImportance
				} else {
					l.Text = humanize.RelTime(x, time.Now(), "", "")
				}
			}
			l.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		co.(*widget.Label).SetText(s.text)
	}
	for i, h := range headers {
		t.SetColumnWidth(i, h.width)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
		if tci.Row >= len(a.contracts) || tci.Row < 0 {
			return
		}
		o := a.contracts[tci.Row]
		a.showContract(o)
	}
	return t
}

func (a *contractsArea) showContract(o *app.CharacterContract) {
	w := a.u.fyneApp.NewWindow("Contract")
	t := widget.NewLabel(o.NameDisplay())
	t.Importance = widget.HighImportance
	expiredAt := o.DateExpiredEffective()
	expirationDate := fmt.Sprintf(
		"%s (%s)",
		expiredAt.Format(app.TimeDefaultFormat),
		strings.Trim(humanize.RelTime(expiredAt, time.Now(), "", ""), " "),
	)
	main := container.NewVBox(
		&widget.Form{
			Items: []*widget.FormItem{
				{Text: "Info by issuer", Widget: widget.NewLabel(o.TitleDisplay())},
				{Text: "Type", Widget: widget.NewLabel(o.TypeDisplay())},
				{Text: "Issued By", Widget: widget.NewLabel(o.Issuer.Name)},
				{Text: "Availability", Widget: widget.NewLabel(o.AvailabilityDisplay())},
				{Text: "Status", Widget: widget.NewLabel(o.StatusDisplay())},
				{Text: "Location", Widget: widget.NewLabel(o.StartLocation.Name)},
				{Text: "Date Issued", Widget: widget.NewLabel(o.DateIssued.Format(app.TimeDefaultFormat))},
				{Text: "Expiration Date", Widget: widget.NewLabel(expirationDate)},
			},
		},
		widget.NewSeparator(),
	)
	switch o.Type {
	case app.ContractTypeCourier:
		var collateral string
		if o.Collateral == 0 {
			collateral = "(None)"
		} else {
			collateral = fmt.Sprintf("%s ISK", humanize.Commaf(o.Collateral))
		}
		main.Add(&widget.Form{
			Items: []*widget.FormItem{
				{Text: "Complete In", Widget: widget.NewLabel(fmt.Sprintf("%d days", o.DaysToComplete))},
				{Text: "Volume", Widget: widget.NewLabel(fmt.Sprintf("%f m3", o.Volume))},
				{Text: "Reward", Widget: widget.NewLabel(fmt.Sprintf("%s ISK", humanize.Commaf(o.Reward)))},
				{Text: "Collateral", Widget: widget.NewLabel(collateral)},
				{Text: "Destination", Widget: widget.NewLabel(o.EndLocation.Name)},
			},
		})
		main.Add(widget.NewSeparator())
	case app.ContractTypeItemExchange, app.ContractTypeAuction:
		items, err := a.u.CharacterService.ListCharacterContractItems(context.TODO(), o.ID)
		if err != nil {
			d := NewErrorDialog("Failed to fetch contract items", err, w)
			d.SetOnClosed(w.Hide)
			d.Show()
		}
		var itemsIncluded, itemsRequested []*app.CharacterContractItem
		for _, it := range items {
			if it.IsIncluded {
				itemsIncluded = append(itemsIncluded, it)
			} else {
				itemsRequested = append(itemsRequested, it)
			}
		}
		// payment
		f := widget.NewForm()
		if o.Price > 0 {
			x := widget.NewLabel(humanize.Commaf(o.Price) + " ISK")
			x.Importance = widget.DangerImportance
			f.Append("Buyer Will Pay", x)
		} else {
			x := widget.NewLabel(humanize.Commaf(o.Reward) + " ISK")
			x.Importance = widget.SuccessImportance
			f.Append("Buyer Will Get", x)
		}
		main.Add(f)
		main.Add(widget.NewSeparator())
		makeItemLabel := func(it *app.CharacterContractItem) *kxwidget.TappableLabel {
			t := fmt.Sprintf("%s (%s) x %d ", it.Type.Name, it.Type.Group.Name, it.Quantity)
			return kxwidget.NewTappableLabel(t, func() {
				a.u.showTypeInfoWindow(it.Type.ID, o.CharacterID, 0)
			})
		}
		// included items
		if len(itemsIncluded) > 0 {
			t := widget.NewLabel("Buyer Will Get")
			t.Importance = widget.SuccessImportance
			c := container.NewVBox(t)
			for _, it := range itemsIncluded {
				c.Add(makeItemLabel(it))
			}
			main.Add(c)
			main.Add(widget.NewSeparator())
		}
		// requested items
		if len(itemsRequested) > 0 {
			t := widget.NewLabel("Buyer Will Provide")
			t.Importance = widget.DangerImportance
			c := container.NewVBox(t)
			for _, it := range itemsRequested {
				c.Add(makeItemLabel(it))
			}
			main.Add(c)
			main.Add(widget.NewSeparator())
		}
	}
	b := widget.NewButton("Close", func() {
		w.Hide()
	})
	w.SetContent(container.NewBorder(
		container.NewVBox(t, widget.NewSeparator()),
		container.NewCenter(b),
		nil,
		nil,
		main,
	))
	w.Resize(fyne.NewSize(600, 400))
	w.Show()
}

func (a *contractsArea) refresh() {
	var t string
	var i widget.Importance
	if err := a.updateEntries(); err != nil {
		slog.Error("Failed to refresh contracts UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeTopText()
	}
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.table.Refresh()
}

func (a *contractsArea) makeTopText() (string, widget.Importance) {
	if !a.u.hasCharacter() {
		return "No character", widget.LowImportance
	}
	c := a.u.currentCharacter()
	hasData := a.u.StatusCacheService.CharacterSectionExists(c.ID, app.SectionContracts)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	t := humanize.Comma(int64(len(a.contracts)))
	s := fmt.Sprintf("Entries: %s", t)
	return s, widget.MediumImportance
}

func (a *contractsArea) updateEntries() error {
	if !a.u.hasCharacter() {
		a.contracts = make([]*app.CharacterContract, 0)
		return nil
	}
	characterID := a.u.characterID()
	var err error
	a.contracts, err = a.u.CharacterService.ListCharacterContracts(context.TODO(), characterID)
	if err != nil {
		return err
	}
	return nil
}
