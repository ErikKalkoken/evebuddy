package contracts

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
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type contractItem struct {
	ContractID  int64
	IsIncluded  bool
	IsSingleton bool
	Quantity    int64
	RawQuantity optional.Optional[int64]
	RecordID    int64
	Type        *app.EveType
}

// ShowCharacterContract2 shows the details of a character contract in a window.
func ShowCharacterContract2(ctx context.Context, u baseUI, characterID, contractID int64) {
	reportError := func(err error) {
		ui.ShowErrorAndLog("Failed to show contract", err, u.IsDeveloperMode(), u.MainWindow())
	}

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
	ShowCharacterContract(u, newContractRowForCharacter(o, func(_ int64) string {
		return c.NameOrZero()
	}))
}

// ShowCharacterContract shows the details of a character contract in a window.
func ShowCharacterContract(u baseUI, r contractRow) {
	showContractDetails(u, r,
		func(ctx context.Context) (int, float64, error) {
			if r.contractType != app.ContractTypeAuction {
				return 0, 0, nil
			}
			count, err := u.Character().CountContractBids(ctx, r.objectID)
			if err != nil {
				return 0, 0, err
			}
			topBid, err := u.Character().GetContractTopBid(ctx, r.objectID)
			if err != nil {
				return 0, 0, err
			}
			return count, topBid.Amount, nil
		},
		func(ctx context.Context) ([]contractItem, error) {
			if r.contractType != app.ContractTypeAuction && r.contractType != app.ContractTypeItemExchange {
				return nil, nil
			}
			oo, err := u.Character().ListContractItems(ctx, r.objectID)
			if err != nil {
				return nil, err
			}
			items := xslices.Map(oo, func(x *app.CharacterContractItem) contractItem {
				return contractItem{
					ContractID:  x.ContractID,
					IsIncluded:  x.IsIncluded,
					IsSingleton: x.IsIncluded,
					Quantity:    x.Quantity,
					RawQuantity: x.RawQuantity,
					RecordID:    x.RecordID,
					Type:        x.Type,
				}
			})
			return items, nil
		},
	)
}

// ShowCorporationContract shows the details of a corporation contract in a window.
func ShowCorporationContract(u baseUI, r contractRow) {
	showContractDetails(u, r,
		func(ctx context.Context) (int, float64, error) {
			if r.contractType != app.ContractTypeAuction {
				return 0, 0, nil
			}
			count, err := u.Corporation().CountContractBids(ctx, r.objectID)
			if err != nil {
				return 0, 0, err
			}
			topBid, err := u.Corporation().GetContractTopBid(ctx, r.objectID)
			if err != nil {
				return 0, 0, err
			}
			return count, topBid.Amount, nil
		},
		func(ctx context.Context) ([]contractItem, error) {
			if r.contractType != app.ContractTypeAuction && r.contractType != app.ContractTypeItemExchange {
				return nil, nil
			}
			oo, err := u.Corporation().ListContractItems(ctx, r.objectID)
			if err != nil {
				return nil, err
			}
			items := xslices.Map(oo, func(x *app.CorporationContractItem) contractItem {
				return contractItem{
					ContractID:  x.ContractID,
					IsIncluded:  x.IsIncluded,
					IsSingleton: x.IsIncluded,
					Quantity:    x.Quantity,
					RawQuantity: x.RawQuantity,
					RecordID:    x.RecordID,
					Type:        x.Type,
				}
			})
			return items, nil
		},
	)
}

func showContractDetails(u baseUI, r contractRow, fetchBids func(context.Context) (int, float64, error), fetchItems func(context.Context) ([]contractItem, error)) {
	title := fmt.Sprintf("Contract #%d", r.contractID)
	w, created := u.GetOrCreateWindow(
		fmt.Sprintf("contract-%d-%d", r.ownerID, r.contractID),
		title,
		r.ownerName,
	)
	if !created {
		w.Show()
		return
	}
	go func() {
		reportError := func(err error) {
			ui.ShowErrorAndLog("Failed to show contract", err, u.IsDeveloperMode(), u.MainWindow())
		}
		ctx := context.Background()
		totalBids, topBidAmount, err := fetchBids(ctx)
		if err != nil {
			reportError(err)
		}
		items, err := fetchItems(ctx)
		if err != nil {
			reportError(err)
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
				widget.NewFormItem("Info by issuer", widget.NewLabel(r.title)),
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
			fi = append(fi, widget.NewFormItem("Status", xwidget.NewRichText(r.status.DisplayRichText()...)))
			fi = append(fi, widget.NewFormItem("Location", makeLocationLabel2(r.startLocation, u.InfoViewer().ShowLocation)))

			if r.contractType == app.ContractTypeCourier || r.contractType == app.ContractTypeItemExchange {
				fi = append(fi, widget.NewFormItem("Date Issued", widget.NewLabel(r.dateIssued.Format(app.DateTimeFormat))))
				fi = append(fi, widget.NewFormItem("Date Accepted", widget.NewLabel(r.dateAccepted.StringFunc("", func(v time.Time) string {
					return v.Format(app.DateTimeFormat)
				}))))
				fi = append(fi, widget.NewFormItem("Date Expired", widget.NewLabel(makeContractExpiresString(r.dateExpired, r.isExpired))))
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
				if r.price.ValueOrZero() > 0 {
					x := widget.NewLabel(r.price.StringFunc("?", ui.FormatISKAmount))
					x.Importance = widget.DangerImportance
					fi = append(fi, widget.NewFormItem("Buyer Will Pay", x))
				} else {
					x := widget.NewLabel(r.reward.StringFunc("?", ui.FormatISKAmount))
					x.Importance = widget.SuccessImportance
					fi = append(fi, widget.NewFormItem("Buyer Will Get", x))
				}
			case app.ContractTypeAuction:
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
					{Text: "Expires", Widget: widget.NewLabel(makeContractExpiresString(r.dateExpired, r.isExpired))},
				})
			}

			makeItemsInfo := func() (fyne.CanvasObject, error) {
				vb := container.NewVBox()
				var itemsIncluded, itemsRequested []contractItem
				for _, it := range items {
					if it.IsIncluded {
						itemsIncluded = append(itemsIncluded, it)
					} else {
						itemsRequested = append(itemsRequested, it)
					}
				}
				makeItem := func(it contractItem) fyne.CanvasObject {
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
					ui.ShowErrorAndLog("Failed to show contract items", err, u.IsDeveloperMode(), u.MainWindow())
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

func makeContractExpiresString(dateExpired time.Time, isExpired bool) string {
	ts := dateExpired.Format(app.DateTimeFormat)
	var ds string
	if isExpired {
		ds = "EXPIRED"
	} else {
		ds = ihumanize.RelTime(dateExpired)
	}
	return fmt.Sprintf("%s (%s)", ts, ds)
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
