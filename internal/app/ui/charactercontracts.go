package ui

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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

type CharacterContracts struct {
	widget.BaseWidget

	ShowActiveOnly bool
	OnUpdate       func(count int)

	contracts []*app.CharacterContract
	body      fyne.CanvasObject
	top       *widget.Label
	u         *BaseUI
}

func NewCharacterContracts(u *BaseUI) *CharacterContracts {
	a := &CharacterContracts{
		contracts: make([]*app.CharacterContract, 0),
		top:       appwidget.MakeTopLabel(),
		u:         u,
	}
	a.ExtendBaseWidget(a)
	headers := []iwidget.HeaderDef{
		{Text: "Contract", Width: 300},
		{Text: "Type", Width: 120},
		{Text: "From", Width: 150},
		{Text: "To", Width: 150},
		{Text: "Status", Width: 100},
		{Text: "Date Issued", Width: 150},
		{Text: "Date Accepted", Width: 150},
		{Text: "Time Left", Width: 100},
	}
	makeCell := func(col int, o *app.CharacterContract) []widget.RichTextSegment {
		switch col {
		case 0:
			return iwidget.NewRichTextSegmentFromText(o.NameDisplay())
		case 1:
			return iwidget.NewRichTextSegmentFromText(o.TypeDisplay())
		case 2:
			return iwidget.NewRichTextSegmentFromText(o.IssuerEffective().Name)
		case 3:
			var s string
			if o.Assignee == nil {
				s = ""
			} else {
				s = o.Assignee.Name
			}
			return iwidget.NewRichTextSegmentFromText(s)
		case 4:
			return o.StatusDisplayRichText()
		case 5:
			return iwidget.NewRichTextSegmentFromText(o.DateIssued.Format(app.DateTimeFormat))
		case 6:
			var s string
			if o.DateAccepted.IsEmpty() {
				s = ""
			} else {
				s = o.DateAccepted.MustValue().Format(app.DateTimeFormat)
			}
			return iwidget.NewRichTextSegmentFromText(s)
		case 7:
			var text string
			var color fyne.ThemeColorName
			if o.IsExpired() {
				text = "EXPIRED"
				color = theme.ColorNameError
			} else {
				text = ihumanize.RelTime(o.DateExpired)
				color = theme.ColorNameForeground
			}
			return iwidget.NewRichTextSegmentFromText(text, widget.RichTextStyle{
				ColorName: color,
			})
		}
		return iwidget.NewRichTextSegmentFromText("?")
	}
	if a.u.IsDesktop() {
		a.body = iwidget.MakeDataTableForDesktop2(headers, &a.contracts, makeCell, func(column int, r *app.CharacterContract) {
			switch column {
			case 0:
				a.showContract(r)
			case 2:
				a.u.ShowEveEntityInfoWindow(r.IssuerEffective())
			case 3:
				if r.Assignee != nil {
					a.u.ShowEveEntityInfoWindow(r.Assignee)
				}
			}
		})
	} else {
		a.body = iwidget.MakeDataTableForMobile2(headers, &a.contracts, makeCell, a.showContract)
	}
	return a
}

func (a *CharacterContracts) Update() {
	var t string
	var i widget.Importance
	if err := a.updateEntries(); err != nil {
		slog.Error("Failed to refresh contracts UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeTopText()
	}
	if t != "" {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
		a.top.Show()
	} else {
		a.top.Hide()
	}
	a.body.Refresh()
}

func (a *CharacterContracts) makeTopText() (string, widget.Importance) {
	if !a.u.HasCharacter() {
		return "No character", widget.LowImportance
	}
	c := a.u.CurrentCharacter()
	hasData := a.u.StatusCacheService().CharacterSectionExists(c.ID, app.SectionContracts)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	return "", widget.MediumImportance
}

func (a *CharacterContracts) updateEntries() error {
	if !a.u.HasCharacter() {
		a.contracts = make([]*app.CharacterContract, 0)
		return nil
	}
	characterID := a.u.CurrentCharacterID()
	var err error
	oo, err := a.u.CharacterService().ListContracts(context.Background(), characterID)
	if err != nil {
		return err
	}
	if a.ShowActiveOnly {
		a.contracts = xslices.Filter(oo, func(o *app.CharacterContract) bool {
			return o.IsActive()
		})

	} else {
		a.contracts = oo
	}
	if a.OnUpdate != nil {
		a.OnUpdate(len(a.contracts))
	}
	return nil
}

func (a *CharacterContracts) showContract(c *app.CharacterContract) {
	var w fyne.Window
	makeExpiresString := func(c *app.CharacterContract) string {
		ts := c.DateExpired.Format(app.DateTimeFormat)
		var ds string
		if c.IsExpired() {
			ds = "EXPIRED"
		} else {
			ds = ihumanize.RelTime(c.DateExpired)
		}
		return fmt.Sprintf("%s (%s)", ts, ds)
	}
	makeEntity := func(ee *app.EveEntity) *kxwidget.TappableLabel {
		return kxwidget.NewTappableLabel(ee.Name, func() {
			a.u.ShowEveEntityInfoWindow(ee)
		})
	}
	makeLocation := func(l *app.EntityShort[int64]) *kxwidget.TappableLabel {
		return kxwidget.NewTappableLabel(l.Name, func() {
			a.u.ShowLocationInfoWindow(l.ID)
		})
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
		if a.u.IsMobile() {
			f.Orientation = widget.Vertical
		}
		f.Append("Info by issuer", widget.NewLabel(c.TitleDisplay()))
		f.Append("Type", widget.NewLabel(c.TypeDisplay()))
		f.Append("Issued By", makeEntity(c.IssuerEffective()))
		availability := container.NewHBox(widget.NewLabel(c.AvailabilityDisplay()))
		if c.Assignee != nil {
			availability.Add(makeEntity(c.Assignee))
		}
		f.Append("Availability", availability)
		if c.Type == app.ContractTypeCourier {
			f.Append("Contractor", widget.NewLabel(c.ContractorDisplay()))
		}
		f.Append("Status", widget.NewRichText(c.StatusDisplayRichText()...))
		f.Append("Location", makeLocation(c.StartLocation))
		if c.Type == app.ContractTypeCourier || c.Type == app.ContractTypeItemExchange {
			f.Append("Date Issued", widget.NewLabel(c.DateIssued.Format(app.DateTimeFormat)))
			f.Append("Expiration Date", widget.NewLabel(makeExpiresString(c)))
		}
		return f
	}
	makePaymentInfo := func(c *app.CharacterContract) fyne.CanvasObject {
		f := widget.NewForm()
		if a.u.IsMobile() {
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
		total, err := a.u.CharacterService().CountContractBids(ctx, c.ID)
		if err != nil {
			d := a.u.NewErrorDialog("Failed to count contract bids", err, w)
			d.SetOnClosed(w.Hide)
			d.Show()
		}
		var currentBid string
		if total == 0 {
			currentBid = "(None)"
		} else {
			top, err := a.u.CharacterService().GetContractTopBid(ctx, c.ID)
			if err != nil {
				d := a.u.NewErrorDialog("Failed to get top bid", err, w)
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
		items, err := a.u.CharacterService().ListContractItems(context.TODO(), c.ID)
		if err != nil {
			d := a.u.NewErrorDialog("Failed to fetch contract items", err, w)
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
			x := kxwidget.NewTappableLabel(it.Type.Name, func() {
				a.u.ShowTypeInfoWindow(it.Type.ID)
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
	if a.u.IsDeveloperMode() {
		main.Add(widget.NewSeparator())
		main.Add(&widget.Form{
			Items: []*widget.FormItem{
				{
					Text:   "Contract ID",
					Widget: a.u.makeCopyToClipbardLabel(fmt.Sprint(c.ContractID)),
				},
			}})
	}
	subTitle := fmt.Sprintf("%s (%s)", c.NameDisplay(), c.TypeDisplay())
	w = a.u.makeDetailWindow("Contract", subTitle, main)
	w.Show()
}

func (a *CharacterContracts) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}
