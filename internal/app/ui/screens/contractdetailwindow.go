package screens

import (
	"context"
	"fmt"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// ShowCharacterContractByID shows the details of a character contract in a window.
// Use ShowContract when the contract row is already available.
func ShowCharacterContractByID(u baseUI, characterID, contractID int64) {
	reportError := func(err error) {
		fyne.Do(func() {
			ui.ShowErrorAndLog("Failed to show contract", err, u.IsDeveloperMode(), u.MainWindow())
		})
	}
	go func() {
		ctx := context.Background()
		o, err := u.Character().GetContract(ctx, characterID, contractID)
		if err != nil {
			reportError(err)
			return
		}
		c, err := u.Character().GetCharacter(ctx, characterID)
		if err != nil {
			reportError(err)
			return
		}
		myCharacterIDs, err := u.Character().ListCharacterIDs(ctx)
		if err != nil {
			reportError(err)
			return
		}
		myCorporationIDs, err := u.Character().ListCharacterCorporationIDs(ctx)
		if err != nil {
			reportError(err)
			return
		}
		fyne.Do(func() {
			ShowContract(u, newContractRow(&o.Contract, o.CharacterID, c.NameOrZero(), false, myCharacterIDs, myCorporationIDs))
		})
	}()
}

// ShowContract shows the details of a character or corporation contract in a window.
func ShowContract(u baseUI, r contractRow) {
	title := fmt.Sprintf("Contract #%d", r.contractID)
	windowID := fmt.Sprintf("contract-%d-%d", r.ownerID, r.contractID)
	w, created := u.GetOrCreateWindow(windowID, title, r.ownerName)
	if !created {
		w.Show()
		return
	}
	go func() {
		reportError := func(err error) {
			fyne.Do(func() {
				u.DestroyWindow(windowID)
				ui.ShowErrorAndLog("Failed to show contract", err, u.IsDeveloperMode(), u.MainWindow())
			})
		}
		ctx := context.Background()
		totalBids, topBidAmount, err := fetchContractBids(ctx, u, r)
		if err != nil {
			reportError(err)
			return
		}
		items, err := fetchContractItems(ctx, u, r)
		if err != nil {
			reportError(err)
			return
		}

		fyne.Do(func() {
			var availability fyne.CanvasObject
			availabilityLabel := widget.NewLabel(r.availability.Display())
			if v, ok := r.assignee.Value(); ok {
				availability = container.NewBorder(
					nil,
					nil,
					availabilityLabel,
					nil,
					ui.MakeEveEntityActionLabel(v, u.InfoViewer().Show),
				)
			} else {
				availability = availabilityLabel
			}
			fi := []*widget.FormItem{
				widget.NewFormItem("Owner", ui.MakeCharacterActionLabel(r.ownerID, r.ownerName, u.InfoViewer().Show)),
				widget.NewFormItem("Description", widget.NewLabel(r.title)),
				widget.NewFormItem("Type", widget.NewLabel(r.contractType.Display())),
				widget.NewFormItem("Issued By", ui.MakeEveEntityActionLabel(r.issuer, u.InfoViewer().Show)),
				widget.NewFormItem("Availability", availability),
			}
			if u.IsDeveloperMode() {
				fi = append(fi, widget.NewFormItem("Contract ID", xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(r.contractID))))
			}
			if r.contractType == app.ContractTypeCourier {
				fi = append(fi, widget.NewFormItem("Contractor", makeEveEntityActionLabel2(r.acceptor, u.InfoViewer().Show)))
			}
			fi = append(fi, widget.NewFormItem("Status", xwidget.NewRichText(r.statusDisplay...)))
			fi = append(fi, widget.NewFormItem("Location", makeLocationLabel2(r.startLocation, u.InfoViewer().ShowLocation)))

			if r.contractType == app.ContractTypeCourier || r.contractType == app.ContractTypeItemExchange {
				fi = append(fi, widget.NewFormItem("Date Issued", widget.NewLabel(r.dateIssued.Format(app.DateTimeFormat))))
				fi = append(fi, widget.NewFormItem("Date Accepted", widget.NewLabel(r.dateAccepted.StringFunc("", func(v time.Time) string {
					return v.Format(app.DateTimeFormat)
				}))))
				var expiredNote string
				if r.deadlineKind == contractDeadlineExpiry {
					expiredNote = makeContractDeadlineNote(r.dateExpired, r.isExpired, "EXPIRED")
				}
				fi = append(fi, widget.NewFormItem("Date Expired", widget.NewLabel(makeContractDateString(r.dateExpired, expiredNote))))
				if deadline, ok := app.ContractDeliveryDeadline(r.dateAccepted, r.daysToComplete); ok {
					var note string
					if r.status == app.ContractStatusInProgress {
						note = makeContractDeadlineNote(deadline, r.isOverdue, "OVERDUE")
					}
					fi = append(fi, widget.NewFormItem("Deliver By", widget.NewLabel(makeContractDateString(deadline, note))))
				}
				fi = append(fi, widget.NewFormItem("Date Completed", widget.NewLabel(r.dateCompleted.StringFunc("", func(v time.Time) string {
					return v.Format(app.DateTimeFormat)
				}))))
			}

			switch r.contractType {
			case app.ContractTypeCourier:
				fi = slices.Concat(fi, []*widget.FormItem{
					{Text: "Complete In", Widget: widget.NewLabel(fmt.Sprintf("%s days", r.daysToComplete.StringFunc("?", func(v int64) string {
						return fmt.Sprint(v)
					})))},
					{Text: "Volume", Widget: widget.NewLabel(fmt.Sprintf("%s m3", r.volume.StringFunc("?", func(v float64) string {
						return fmt.Sprint(v)
					})))},
					{Text: "Reward", Widget: widget.NewLabel(r.reward.StringFunc("-", ui.FormatISKAmount))},
					{Text: "Collateral", Widget: widget.NewLabel(r.collateral.StringFunc("-", ui.FormatISKAmount))},
					{Text: "Destination", Widget: makeLocationLabel2(r.endLocation, u.InfoViewer().ShowLocation)},
				})
			case app.ContractTypeItemExchange:
				if v, ok := r.price.Value(); ok {
					x := widget.NewLabel(ui.FormatISKAmount(v))
					x.Importance = widget.DangerImportance
					fi = append(fi, widget.NewFormItem("Buyer Will Pay", x))
				}
				if v, ok := r.reward.Value(); ok {
					x := widget.NewLabel(ui.FormatISKAmount(v))
					x.Importance = widget.SuccessImportance
					fi = append(fi, widget.NewFormItem("Buyer Will Get", x))
				}
			case app.ContractTypeAuction:
				var auctionExpiresNote string
				if r.status == app.ContractStatusOutstanding {
					auctionExpiresNote = makeContractDeadlineNote(r.dateExpired, r.isExpired, "EXPIRED")
				}
				var currentBid string
				if totalBids == 0 {
					currentBid = "(None)"
				} else {
					currentBid = fmt.Sprintf("%s (%d bids so far)", ui.FormatISKAmount(topBidAmount), totalBids)
				}
				fi = slices.Concat(fi, []*widget.FormItem{
					{Text: "Starting Bid", Widget: widget.NewLabel(r.price.StringFunc("?", ui.FormatISKAmount))},
					{Text: "Buyout Price", Widget: widget.NewLabel(r.buyout.StringFunc("?", ui.FormatISKAmount))},
					{Text: "Current Bid", Widget: widget.NewLabel(currentBid)},
					{Text: "Expires", Widget: widget.NewLabel(makeContractDateString(r.dateExpired, auctionExpiresNote))},
				})
			}

			makeItemsInfo := func() (fyne.CanvasObject, error) {
				vb := container.NewVBox()
				var itemsIncluded, itemsRequested []*app.ContractItem
				for _, it := range items {
					if it.IsIncluded {
						itemsIncluded = append(itemsIncluded, it)
					} else {
						itemsRequested = append(itemsRequested, it)
					}
				}
				makeItem := func(it *app.ContractItem) fyne.CanvasObject {
					c := container.NewHBox(
						ui.MakeLinkLabel(it.Type.Name, func() {
							u.InfoViewer().ShowType(it.Type.ID, r.ownerID)
						}),
						widget.NewLabel(fmt.Sprintf("(%s)", it.Type.Group.Name)),
						widget.NewLabel(fmt.Sprintf("x %s ", humanize.Comma(int64(it.Quantity)))),
					)
					return c
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
				return vb, nil
			}

			subTitle := fmt.Sprintf("%s (%s)", r.name, r.contractType.Display())
			f := widget.NewForm(fi...)
			f.Orientation = widget.Adaptive
			main := container.NewVBox(f)
			if r.contractType == app.ContractTypeItemExchange || r.contractType == app.ContractTypeAuction {
				main.Add(widget.NewSeparator())
				x, err := makeItemsInfo()
				if err != nil {
					reportError(err)
					return
				}
				main.Add(x)
			}
			ui.MakeDetailWindow(ui.MakeDetailWindowParams{
				Title:   subTitle,
				Content: main,
				Window:  w,
			})
			w.Show()
		})
	}()
}

// makeContractDateString returns a formatted date with an optional note.
func makeContractDateString(t time.Time, note string) string {
	ts := t.Format(app.DateTimeFormat)
	if note == "" {
		return ts
	}
	return fmt.Sprintf("%s (%s)", ts, note)
}

// makeContractDeadlineNote returns the time left until a deadline or the passed text when it is over.
func makeContractDeadlineNote(deadline time.Time, isPassed bool, passed string) string {
	if isPassed {
		return passed
	}
	return ihumanize.RelTime(deadline)
}

// ui.MakeEveEntityActionLabel returns a Hyperlink for existing entities or a placeholder label otherwise.
func makeEveEntityActionLabel2(o optional.Optional[*app.EveEntity], action func(o *app.EveEntity)) fyne.CanvasObject {
	v, ok := o.Value()
	if !ok {
		return widget.NewLabel("-")
	}
	return ui.MakeLinkLabelWithWrap(v.Name, func() {
		action(v)
	})
}

func makeLocationLabel2(o optional.Optional[*app.EveLocationShort], show func(int64)) fyne.CanvasObject {
	el, ok := o.Value()
	if !ok {
		return widget.NewLabel("?")
	}
	x := ui.MakeLinkLabelWithWrap(el.DisplayName(), func() {
		show(el.ID)
	})
	x.Wrapping = fyne.TextWrapWord
	return x
}

// fetchContractBids returns the number of bids and the amount of the top bid for an auction.
func fetchContractBids(ctx context.Context, u baseUI, r contractRow) (int, float64, error) {
	if r.contractType != app.ContractTypeAuction {
		return 0, 0, nil
	}
	f := u.Character().GetContractBidSummary
	if r.isCorporation {
		f = u.Corporation().GetContractBidSummary
	}
	count, top, err := f(ctx, r.objectID)
	if err != nil {
		return 0, 0, err
	}
	return count, optional.Map(top, 0, func(x *app.ContractBid) float64 { return x.Amount }), nil
}

// fetchContractItems returns the items of a contract, if its type has items.
func fetchContractItems(ctx context.Context, u baseUI, r contractRow) ([]*app.ContractItem, error) {
	if r.contractType != app.ContractTypeAuction && r.contractType != app.ContractTypeItemExchange {
		return nil, nil
	}
	if r.isCorporation {
		return u.Corporation().ListContractItems(ctx, r.objectID)
	}
	return u.Character().ListContractItems(ctx, r.objectID)
}
