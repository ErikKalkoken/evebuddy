package ui

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
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

// ContractsArea is the UI area that shows the skillqueue
type ContractsArea struct {
	Content *fyne.Container

	contracts []*app.CharacterContract
	body      fyne.CanvasObject
	top       *widget.Label
	u         *BaseUI
}

func NewContractsArea(u *BaseUI) *ContractsArea {
	a := ContractsArea{
		contracts: make([]*app.CharacterContract, 0),
		top:       makeTopLabel(),
		u:         u,
	}
	headers := []headerDef{
		{"Contract", 300},
		{"Type", 120},
		{"From", 150},
		{"To", 150},
		{"Status", 100},
		{"Date Issued", 150},
		{"Date Accepted", 150},
		{"Time Left", 100},
	}
	makeDataLabel := func(col int, o *app.CharacterContract) (string, fyne.TextAlign, widget.Importance) {
		var align fyne.TextAlign
		var importance widget.Importance
		var text string
		switch col {
		case 0:
			text = o.NameDisplay()
		case 1:
			text = o.TypeDisplay()
		case 2:
			text = o.Issuer.Name
		case 3:
			if o.Assignee == nil {
				text = ""
			} else {
				text = o.Assignee.Name
			}
		case 4:
			text = o.StatusDisplay()
		case 5:
			text = o.DateIssued.Format(app.DateTimeDefaultFormat)
		case 6:
			if o.DateAccepted.IsEmpty() {
				text = ""
			} else {
				text = o.DateAccepted.MustValue().Format(app.DateTimeDefaultFormat)
			}
		case 7:
			if o.IsExpired() {
				text = "EXPIRED"
				importance = widget.DangerImportance
			} else {
				text = ihumanize.RelTime(o.DateExpiredEffective())
			}
		}
		return text, align, importance
	}
	if a.u.IsDesktop() {
		a.body = makeDataTableForDesktop(headers, &a.contracts, makeDataLabel, func(column int, r *app.CharacterContract) {
			switch column {
			case 0:
				a.showContract(r)
			case 2:
				a.u.ShowEveEntityInfoWindow(r.Issuer)
			case 3:
				if r.Assignee != nil {
					a.u.ShowEveEntityInfoWindow(r.Assignee)
				}
			}
		})
	} else {
		a.body = makeDataTableForMobile(headers, &a.contracts, makeDataLabel, a.showContract)
	}
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.Content = container.NewBorder(top, nil, nil, nil, a.body)
	return &a
}

// func (a *ContractsArea) makeTable() *widget.Table {
// 	var headers = []struct {
// 		text  string
// 		width float32
// 	}{
// 		{"Contract", 300},
// 		{"Type", 120},
// 		{"From", 150},
// 		{"To", 150},
// 		{"Status", 100},
// 		{"Date Issued", 150},
// 		{"Date Accepted", 150},
// 		{"Time Left", 100},
// 	}
// 	t := widget.NewTable(
// 		func() (rows int, cols int) {
// 			return len(a.contracts), len(headers)
// 		},
// 		func() fyne.CanvasObject {
// 			return widget.NewLabel("Template Template")
// 		},
// 		func(tci widget.TableCellID, co fyne.CanvasObject) {
// 			if tci.Row >= len(a.contracts) || tci.Row < 0 {
// 				return
// 			}
// 			l := co.(*widget.Label)
// 			w := a.contracts[tci.Row]
// 			l.Text, l.Alignment, l.Importance = makeDataLabel(tci.Col, w)
// 			l.Refresh()
// 		},
// 	)
// 	t.ShowHeaderRow = true
// 	t.CreateHeader = func() fyne.CanvasObject {
// 		return widget.NewLabel("Template")
// 	}
// 	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
// 		s := headers[tci.Col]
// 		co.(*widget.Label).SetText(s.text)
// 	}
// 	for i, h := range headers {
// 		t.SetColumnWidth(i, h.width)
// 	}
// 	t.OnSelected = func(tci widget.TableCellID) {
// 		defer t.UnselectAll()
// 		if tci.Row >= len(a.contracts) || tci.Row < 0 {
// 			return
// 		}
// 		o := a.contracts[tci.Row]
// 		a.showContract(o)
// 	}
// 	return t
// }

func (a *ContractsArea) Refresh() {
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
	a.body.Refresh()
}

func (a *ContractsArea) makeTopText() (string, widget.Importance) {
	if !a.u.HasCharacter() {
		return "No character", widget.LowImportance
	}
	c := a.u.CurrentCharacter()
	hasData := a.u.StatusCacheService.CharacterSectionExists(c.ID, app.SectionContracts)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	t := humanize.Comma(int64(len(a.contracts)))
	s := fmt.Sprintf("Entries: %s", t)
	return s, widget.MediumImportance
}

func (a *ContractsArea) updateEntries() error {
	if !a.u.HasCharacter() {
		a.contracts = make([]*app.CharacterContract, 0)
		return nil
	}
	characterID := a.u.CharacterID()
	var err error
	a.contracts, err = a.u.CharacterService.ListCharacterContracts(context.TODO(), characterID)
	if err != nil {
		return err
	}
	return nil
}

func (a *ContractsArea) showContract(c *app.CharacterContract) {
	w := a.u.FyneApp.NewWindow("Contract")
	makeExpiresString := func(c *app.CharacterContract) string {
		t := c.DateExpiredEffective()
		ts := t.Format(app.DateTimeDefaultFormat)
		var ds string
		if c.IsExpired() {
			ds = "EXPIRED"
		} else {
			ds = ihumanize.RelTime(t)
		}
		return fmt.Sprintf("%s (%s)", ts, ds)
	}
	makeLocation := func(l *app.EntityShort[int64]) fyne.CanvasObject {
		x := NewCustomHyperlink(l.Name, func() {
			a.u.ShowLocationInfoWindow(l.ID)
		})
		return x
	}
	makeISKString := func(v float64) string {
		t := humanize.Commaf(v) + " ISK"
		if math.Abs(v) > 999 {
			t += fmt.Sprintf(" (%s)", ihumanize.Number(v, 1))
		}
		return t
	}
	makeBaseInfo := func(c *app.CharacterContract) fyne.CanvasObject {
		f := widget.NewForm()
		if a.u.isMobile {
			f.Orientation = widget.Vertical
		}
		f.Append("Info by issuer", widget.NewLabel(c.TitleDisplay()))
		f.Append("Type", widget.NewLabel(c.TypeDisplay()))
		f.Append("Issued By", widget.NewLabel(c.Issuer.Name))
		f.Append("Availability", widget.NewLabel(c.AvailabilityDisplay()))
		if c.Type == app.ContractTypeCourier {
			f.Append("Contractor", widget.NewLabel(c.ContractorDisplay()))
		}
		f.Append("Status", widget.NewLabel(c.StatusDisplay()))
		f.Append("Location", makeLocation(c.StartLocation))
		if c.Type == app.ContractTypeCourier || c.Type == app.ContractTypeItemExchange {
			f.Append("Date Issued", widget.NewLabel(c.DateIssued.Format(app.DateTimeDefaultFormat)))
			f.Append("Expiration Date", widget.NewLabel(makeExpiresString(c)))
		}
		return f
	}
	makePaymentInfo := func(c *app.CharacterContract) fyne.CanvasObject {
		f := widget.NewForm()
		if a.u.isMobile {
			f.Orientation = widget.Vertical
		}
		if c.Price > 0 {
			x := widget.NewLabel(makeISKString(c.Price))
			x.Importance = widget.DangerImportance
			f.Append("Buyer Will Pay", x)
		} else {
			x := widget.NewLabel(makeISKString(c.Reward))
			x.Importance = widget.SuccessImportance
			f.Append("Buyer Will Get", x)
		}
		return f
	}
	makeCourierInfo := func(c *app.CharacterContract) fyne.CanvasObject {
		var collateral string
		if c.Collateral == 0 {
			collateral = "(None)"
		} else {
			collateral = makeISKString(c.Collateral)
		}
		f := &widget.Form{
			Items: []*widget.FormItem{
				{Text: "Complete In", Widget: widget.NewLabel(fmt.Sprintf("%d days", c.DaysToComplete))},
				{Text: "Volume", Widget: widget.NewLabel(fmt.Sprintf("%f m3", c.Volume))},
				{Text: "Reward", Widget: widget.NewLabel(makeISKString(c.Reward))},
				{Text: "Collateral", Widget: widget.NewLabel(collateral)},
				{Text: "Destination", Widget: makeLocation(c.EndLocation)},
			},
		}
		return f
	}
	makeBidInfo := func(c *app.CharacterContract) fyne.CanvasObject {
		ctx := context.TODO()
		total, err := a.u.CharacterService.CountCharacterContractBids(ctx, c.ID)
		if err != nil {
			d := iwidget.NewErrorDialog("Failed to count contract bids", err, w)
			d.SetOnClosed(w.Hide)
			d.Show()
		}
		var currentBid string
		if total == 0 {
			currentBid = "(None)"
		} else {
			top, err := a.u.CharacterService.GetCharacterContractTopBid(ctx, c.ID)
			if err != nil {
				d := iwidget.NewErrorDialog("Failed to get top bid", err, w)
				d.SetOnClosed(w.Hide)
				d.Show()
			}
			currentBid = fmt.Sprintf("%s (%d bids so far)", makeISKString(float64(top.Amount)), total)
		}
		f := &widget.Form{
			Items: []*widget.FormItem{
				{Text: "Starting Bid", Widget: widget.NewLabel(makeISKString(c.Price))},
				{Text: "Buyout Price", Widget: widget.NewLabel(makeISKString(c.Buyout))},
				{Text: "Current Bid", Widget: widget.NewLabel(currentBid)},
				{Text: "Expires", Widget: widget.NewLabel(makeExpiresString(c))},
			},
		}
		return f
	}
	makeItemsInfo := func(c *app.CharacterContract) fyne.CanvasObject {
		vb := container.NewVBox()
		items, err := a.u.CharacterService.ListCharacterContractItems(context.TODO(), c.ID)
		if err != nil {
			d := iwidget.NewErrorDialog("Failed to fetch contract items", err, w)
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
		makeItem := func(it *app.CharacterContractItem) fyne.CanvasObject {
			x := NewCustomHyperlink(it.Type.Name, func() {
				a.u.ShowTypeInfoWindow(it.Type.ID, c.CharacterID, 0)
			})
			return container.NewHBox(
				x,
				widget.NewLabel(fmt.Sprintf("(%s)", it.Type.Group.Name)),
				widget.NewLabel(fmt.Sprintf("x %s ", humanize.Comma(int64(it.Quantity)))),
			)
		}
		// included items
		if len(itemsIncluded) > 0 {
			t := widget.NewLabel("Buyer Will Get")
			t.Importance = widget.SuccessImportance
			vb.Add(t)
			for _, it := range itemsIncluded {
				vb.Add(makeItem(it))
			}
		}
		// requested items
		if len(itemsRequested) > 0 {
			t := widget.NewLabel("Buyer Will Provide")
			t.Importance = widget.DangerImportance
			vb.Add(t)
			for _, it := range itemsRequested {
				vb.Add(makeItem(it))
			}
		}
		return vb
	}

	// construct window content
	main := container.NewVBox(makeBaseInfo(c), widget.NewSeparator())
	switch c.Type {
	case app.ContractTypeCourier:
		main.Add(makeCourierInfo(c))
	case app.ContractTypeItemExchange:
		main.Add(makePaymentInfo(c))
		main.Add(widget.NewSeparator())
		main.Add(makeItemsInfo(c))
	case app.ContractTypeAuction:
		main.Add(makeBidInfo(c))
		main.Add(widget.NewSeparator())
		main.Add(makeItemsInfo(c))
	}
	main.Add(widget.NewSeparator())

	t := widget.NewLabel(fmt.Sprintf("%s (%s)", c.NameDisplay(), c.TypeDisplay()))
	t.Importance = widget.HighImportance
	t.TextStyle.Bold = true
	top := container.NewVBox(t, widget.NewSeparator())

	bottom := container.NewCenter(widget.NewButton("Close", func() {
		w.Hide()
	}))

	vs := container.NewVScroll(main)
	vs.SetMinSize(fyne.NewSize(600, 500))

	w.SetContent(container.NewPadded(container.NewBorder(
		top,
		bottom,
		nil,
		nil,
		vs,
	)))
	w.Show()
}
