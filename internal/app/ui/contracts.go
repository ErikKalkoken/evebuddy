package ui

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

type Contracts struct {
	widget.BaseWidget

	OnUpdate func(count int)

	contracts      []*app.CharacterContract
	showActiveOnly atomic.Bool
	body           fyne.CanvasObject
	top            *widget.Label
	u              *BaseUI
}

func NewContracts(u *BaseUI, showActiveOnly bool) *Contracts {
	a := &Contracts{
		contracts: make([]*app.CharacterContract, 0),
		top:       makeTopLabel(),
		u:         u,
	}
	a.ExtendBaseWidget(a)
	a.showActiveOnly.Store(showActiveOnly)
	headers := []headerDef{
		{Text: "Contract", Width: 300},
		{Text: "Type", Width: 120},
		{Text: "From", Width: 150},
		{Text: "To", Width: 150},
		{Text: "Status", Width: 100},
		{Text: "Date Issued", Width: 150},
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
	if a.u.isDesktop {
		a.body = makeDataTableForDesktop(headers, &a.contracts, makeCell, func(column int, r *app.CharacterContract) {
			a.showContract(r)
		})
	} else {
		a.body = makeDataTableForMobile(headers, &a.contracts, makeCell, a.showContract)
	}
	return a
}

func (a *Contracts) update() {
	contracts, err := a.updateEntries()
	if err != nil {
		slog.Error("Failed to refresh contracts UI", "err", err)
		fyne.Do(func() {
			a.top.Text = fmt.Sprintf("ERROR: %s", a.u.humanizeError(err))
			a.top.Importance = widget.DangerImportance
			a.top.Refresh()
			a.top.Show()
		})
		return
	}
	fyne.Do(func() {
		a.top.Hide()
		a.contracts = contracts
		a.body.Refresh()
	})
	if a.OnUpdate != nil {
		a.OnUpdate(len(contracts))
	}
}

func (a *Contracts) updateEntries() ([]*app.CharacterContract, error) {
	var contracts []*app.CharacterContract
	oo, err := a.u.cs.ListAllContracts(context.Background())
	if err != nil {
		return nil, err
	}
	if a.showActiveOnly.Load() {
		contracts = xslices.Filter(oo, func(o *app.CharacterContract) bool {
			return o.IsActive()
		})
	} else {
		contracts = oo
	}
	return contracts, nil
}

func (a *Contracts) showContract(c *app.CharacterContract) {
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
	makeLocation := func(l *app.EveLocationShort) *iwidget.TappableRichText {
		if l == nil {
			return iwidget.NewTappableRichTextWithText("?", nil)
		}
		x := iwidget.NewTappableRichText(func() {
			a.u.ShowLocationInfoWindow(l.ID)
		},
			l.DisplayRichText()...,
		)
		return x
	}
	makeISKString := func(v float64) string {
		t := humanize.Commaf(v) + " ISK"
		if math.Abs(v) > 999 {
			t += fmt.Sprintf(" (%s)", ihumanize.Number(v, 1))
		}
		return t
	}

	availability := container.NewHBox(widget.NewLabel(c.AvailabilityDisplay()))
	if c.Assignee != nil {
		availability.Add(makeEntity(c.Assignee))
	}
	fi := []*widget.FormItem{
		widget.NewFormItem("Info by issuer", widget.NewLabel(c.TitleDisplay())),
		widget.NewFormItem("Type", widget.NewLabel(c.TypeDisplay())),
		widget.NewFormItem("Issued By", makeEntity(c.IssuerEffective())),
		widget.NewFormItem("Availability", availability),
	}
	if a.u.IsDeveloperMode() {
		fi = append(fi, widget.NewFormItem("Contract ID", a.u.makeCopyToClipboardLabel(fmt.Sprint(c.ContractID))))
	}
	if c.Type == app.ContractTypeCourier {
		fi = append(fi, widget.NewFormItem("Contractor", widget.NewLabel(c.ContractorDisplay())))
	}
	fi = append(fi, widget.NewFormItem("Status", widget.NewRichText(c.StatusDisplayRichText()...)))
	fi = append(fi, widget.NewFormItem("Location", makeLocation(c.StartLocation)))

	if c.Type == app.ContractTypeCourier || c.Type == app.ContractTypeItemExchange {
		fi = append(fi, widget.NewFormItem("Date Issued", widget.NewLabel(c.DateIssued.Format(app.DateTimeFormat))))
		fi = append(fi, widget.NewFormItem("Date Accepted", widget.NewLabel(c.DateAccepted.StringFunc("", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))))
		fi = append(fi, widget.NewFormItem("Date Expired", widget.NewLabel(makeExpiresString(c))))
		fi = append(fi, widget.NewFormItem("Date Completed", widget.NewLabel(c.DateCompleted.StringFunc("", func(v time.Time) string {
			return v.Format(app.DateTimeFormat)
		}))))
	}

	switch c.Type {
	case app.ContractTypeCourier:
		var collateral string
		if c.Collateral == 0 {
			collateral = "(None)"
		} else {
			collateral = makeISKString(c.Collateral)
		}
		fi = slices.Concat(fi, []*widget.FormItem{
			{Text: "Complete In", Widget: widget.NewLabel(fmt.Sprintf("%d days", c.DaysToComplete))},
			{Text: "Volume", Widget: widget.NewLabel(fmt.Sprintf("%f m3", c.Volume))},
			{Text: "Reward", Widget: widget.NewLabel(makeISKString(c.Reward))},
			{Text: "Collateral", Widget: widget.NewLabel(collateral)},
			{Text: "Destination", Widget: makeLocation(c.EndLocation)},
		})
	case app.ContractTypeItemExchange:
		if c.Price > 0 {
			x := widget.NewLabel(makeISKString(c.Price))
			x.Importance = widget.DangerImportance
			fi = append(fi, widget.NewFormItem("Buyer Will Pay", x))
		} else {
			x := widget.NewLabel(makeISKString(c.Reward))
			x.Importance = widget.SuccessImportance
			fi = append(fi, widget.NewFormItem("Buyer Will Get", x))
		}
	case app.ContractTypeAuction:
		ctx := context.TODO()
		total, err := a.u.cs.CountContractBids(ctx, c.ID)
		if err != nil {
			d := a.u.NewErrorDialog("Failed to count contract bids", err, w)
			d.SetOnClosed(w.Hide)
			d.Show()
		}
		var currentBid string
		if total == 0 {
			currentBid = "(None)"
		} else {
			top, err := a.u.cs.GetContractTopBid(ctx, c.ID)
			if err != nil {
				d := a.u.NewErrorDialog("Failed to get top bid", err, w)
				d.SetOnClosed(w.Hide)
				d.Show()
			}
			currentBid = fmt.Sprintf("%s (%d bids so far)", makeISKString(float64(top.Amount)), total)
		}
		fi = slices.Concat(fi, []*widget.FormItem{
			{Text: "Starting Bid", Widget: widget.NewLabel(makeISKString(c.Price))},
			{Text: "Buyout Price", Widget: widget.NewLabel(makeISKString(c.Buyout))},
			{Text: "Current Bid", Widget: widget.NewLabel(currentBid)},
			{Text: "Expires", Widget: widget.NewLabel(makeExpiresString(c))},
		})
	}

	makeItemsInfo := func(c *app.CharacterContract) fyne.CanvasObject {
		vb := container.NewVBox()
		items, err := a.u.cs.ListContractItems(context.TODO(), c.ID)
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

	subTitle := fmt.Sprintf("%s (%s)", c.NameDisplay(), c.TypeDisplay())
	f := widget.NewForm(fi...)
	f.Orientation = widget.Adaptive
	main := container.NewVBox(f)
	if c.Type == app.ContractTypeItemExchange || c.Type == app.ContractTypeAuction {
		main.Add(widget.NewSeparator())
		main.Add(makeItemsInfo(c))
	}
	w = a.u.makeDetailWindow("Contract", subTitle, main)
	w.Show()
}

func (a *Contracts) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.body)
	return widget.NewSimpleRenderer(c)
}
