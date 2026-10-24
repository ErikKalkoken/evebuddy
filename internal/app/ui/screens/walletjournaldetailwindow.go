package screens

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// ShowCharacterWalletJournalEntryWindowAsync shows a wallet journal entry for a character in a new window.
func ShowCharacterWalletJournalEntryWindowAsync(u baseUI, characterID int64, characterName string, refID int64) {
	windowID := fmt.Sprintf("walletjournalentry-%d-%d", characterID, refID)
	title := fmt.Sprintf("Character Wallet Transaction #%d", refID)
	w, created := u.GetOrCreateWindow(windowID, title, characterName)
	if !created {
		w.Show()
		return
	}

	runAsync(func() {
		o, err := u.Character().GetWalletJournalEntry(context.Background(), characterID, refID)
		if err != nil {
			fyne.Do(func() {
				u.DestroyWindow(windowID)
				ui.ShowErrorAndLog("Failed to show wallet transaction", err, u.IsDeveloperMode(), u.MainWindow())
			})
			return
		}

		fyne.Do(func() {
			f := widget.NewForm()
			f.Orientation = widget.Adaptive

			amount := widget.NewLabel(o.Amount.StringFunc("?", ui.FormatISKAmount))
			amount.Importance = importanceISKAmount(o.Amount)
			reason := o.Reason.ValueOrFallback("-")

			contextDefaultWidget := widget.NewLabel("?")
			contextItem := widget.NewFormItem("Related item", contextDefaultWidget)
			reportError := func(o *app.CharacterWalletJournalEntry, err error) {
				slog.Error(
					"Failed to fetch related context",
					slog.Any("contextIDType", o.ContextIDType),
					slog.Any("contextID", o.ContextID),
					slog.Any("error", err),
				)
				fyne.Do(func() {
					contextDefaultWidget.SetText("Failed to load related item: " + u.ErrorDisplay(err))
				})
			}
			// TODO: Add support for industry jobs
			var hasContext bool
			if v, ok := o.ContextIDType.Value(); ok {
				hasContext = true
				switch v {
				case "alliance_id", "character_id", "corporation_id", "system_id", "type_id":
					runAsync(func() {
						ee, err := u.EVEUniverse().GetOrCreateEntityESI(context.Background(), o.ContextID.ValueOrZero())
						if err != nil {
							reportError(o, err)
							return
						}
						fyne.Do(func() {
							contextItem.Text = "Related " + ee.CategoryDisplay()
							contextItem.Widget = ui.MakeEveEntityActionLabel(ee, u.InfoViewer().Show)
							f.Refresh()
						})
					})
				case "contract_id":
					runAsync(func() {
						c, err := u.Character().GetContract(context.Background(), characterID, o.ContextID.ValueOrZero())
						if err != nil {
							reportError(o, err)
							return
						}
						fyne.Do(func() {
							contextItem.Text = "Related contract"
							contextItem.Widget = ui.MakeLinkLabelWithWrap(c.NameDisplay(), func() {
								ShowCharacterContractByID(u, c.CharacterID, c.ContractID)
							})
							f.Refresh()
						})
					})
				case "market_transaction_id":
					contextItem.Text = "Related market transaction"
					contextItem.Widget = ui.MakeLinkLabelWithWrap(fmt.Sprintf("#%d", o.ContextID.ValueOrZero()), func() {
						ShowCharacterWalletTransactionWindowAsync(u, characterID, characterName, o.ContextID.ValueOrZero())
					})
					f.Refresh()
				case "station_id", "structure_id":
					contextItem.Text = "Related location"
					runAsync(func() {
						ctx := context.Background()
						ts, err := u.Character().TokenSource(ctx, characterID, set.Of(goesi.ScopeUniverseReadStructuresV1))
						if err != nil {
							reportError(o, err)
							return
						}
						ctx = xgoesi.NewContextWithAuth(ctx, characterID, ts)
						el, err := u.EVEUniverse().GetOrCreateLocationESI(ctx, o.ContextID.ValueOrZero())
						if err != nil {
							reportError(o, err)
							return
						}
						fyne.Do(func() {
							contextItem.Widget = ui.MakeLocationLabel(el.ToEveLocationShort(), u.InfoViewer().ShowLocation)
							f.Refresh()
						})
					})
				default:
					hasContext = false // unsupported types would show "?" forever
				}
			}
			items := []*widget.FormItem{
				widget.NewFormItem("Owner", ui.MakeCharacterActionLabel(
					characterID,
					characterName,
					u.InfoViewer().Show,
				)),
				widget.NewFormItem("Date", widget.NewLabel(o.Date.Format(app.DateTimeFormatWithSeconds))),
				widget.NewFormItem("Type", makeLabelWithWrap(o.RefTypeDisplay())),
				widget.NewFormItem("Amount", amount),
				widget.NewFormItem("Balance", widget.NewLabel(o.Balance.StringFunc("?", ui.FormatISKAmount))),
				widget.NewFormItem("Description", makeLabelWithWrap(o.Description)),
				widget.NewFormItem("Reason", makeLabelWithWrap(reason)),
			}
			if v, ok := o.FirstParty.Value(); ok {
				items = append(items, widget.NewFormItem(
					"First Party",
					ui.MakeEveEntityActionLabel(v, u.InfoViewer().Show),
				))
			}
			if v, ok := o.SecondParty.Value(); ok {
				items = append(items, widget.NewFormItem(
					"Second Party",
					ui.MakeEveEntityActionLabel(v, u.InfoViewer().Show),
				))
			}
			if v, ok := o.TaxReceiver.Value(); ok {
				items = append(items, widget.NewFormItem(
					"Tax",
					widget.NewLabel(o.Tax.StringFunc("?", ui.FormatISKAmount)),
				))
				items = append(items, widget.NewFormItem(
					"Tax Receiver",
					ui.MakeEveEntityActionLabel(v, u.InfoViewer().Show)),
				)
			}
			if hasContext {
				items = append(items, contextItem)
			}
			if u.IsDeveloperMode() {
				items = append(items, widget.NewFormItem("Ref ID", xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(refID))))
			}

			for _, it := range items {
				f.AppendItem(it)
			}
			ui.MakeDetailWindow(ui.MakeDetailWindowParams{
				Title:   title,
				Content: f,
				Window:  w,
			})
			w.Show()
		})
	})
}

// ShowCorporationWalletJournalEntryWindowAsync shows a wallet journal entry for a corporation in a new window.
func ShowCorporationWalletJournalEntryWindowAsync(u baseUI, corporationID int64, corporationName string, division app.Division, refID int64) {
	windowID := fmt.Sprintf("walletjournalentry-%d-%d", corporationID, refID)
	title := fmt.Sprintf("Corporation Wallet Transaction #%d", refID)
	w, created := u.GetOrCreateWindow(windowID, title, corporationName)
	if !created {
		w.Show()
		return
	}

	runAsync(func() {
		o, err := u.Corporation().GetWalletJournalEntry(context.Background(), corporationID, division, refID)
		if err != nil {
			fyne.Do(func() {
				u.DestroyWindow(windowID)
				ui.ShowErrorAndLog("Failed to show wallet transaction", err, u.IsDeveloperMode(), u.MainWindow())
			})
			return
		}

		fyne.Do(func() {
			f := widget.NewForm()
			f.Orientation = widget.Adaptive

			amount := widget.NewLabel(o.Amount.StringFunc("?", ui.FormatISKAmount))
			amount.Importance = importanceISKAmount(o.Amount)
			reason := o.Reason.ValueOrFallback("-")

			// contextDefaultWidget := widget.NewLabel("?")
			// contextItem := widget.NewFormItem("Related item", contextDefaultWidget)
			// ctx := context.Background()
			// reportError := func(o *app.CorporationWalletJournalEntry, err error) {
			// 	slog.Error("Failed to fetch related context", "contextIDType", o.ContextIDType, "contextID", o.ContextID, "error", err)
			// 	contextDefaultWidget.SetText("Failed to load related item: " + a.u.ErrorDisplay(err))
			// }
			// TODO: Add support for industry jobs
			// switch o.ContextIDType {
			// case "alliance_id", "character_id", "corporation_id", "system_id", "type_id":
			// 	go func() {
			// 		ee, err := u.EVEUniverse().GetOrCreateEntityESI(ctx, int64(o.ContextID))
			// 		if err != nil {
			// 			reportError(o, err)
			// 			return
			// 		}
			// 		contextItem.Text = "Related " + ee.CategoryDisplay()
			// 		contextItem.Widget = ui.MakeEveEntityActionLabel(ee, u.InfoViewer().ShowEntity)
			// 		f.Refresh()
			// 	}()
			// case "contract_id":
			// 	c, err := u.Character().GetContract(ctx, corporationID, int64(o.ContextID))
			// 	if err != nil {
			// 		reportError(o, err)
			// 		break
			// 	}
			// 	contextItem.Text = "Related contract"
			// 	contextItem.Widget = ui.MakeLinkLabelWithWrap(c.NameDisplay(), func() {
			// 		showContract(u, c.CorporationID, c.ContractID)
			// 	})
			// case "market_transaction_id":
			// 	contextItem.Text = "Related market transaction"
			// 	contextItem.Widget = ui.MakeLinkLabelWithWrap(fmt.Sprintf("#%d", o.ContextID), func() {
			// 		showCorporationWalletTransaction(u, o.CorporationID, o.ContextID)
			// 	})
			// case "station_id", "structure_id":
			// 	contextItem.Text = "Related location"
			// 	go func() {
			// 		token, err := u.Character().GetValidCorporationToken(ctx, corporationID)
			// 		if err != nil {
			// 			reportError(o, err)
			// 			return
			// 		}
			// 		ctx = xesi.NewContextWithAuth(ctx, token.CharacterID, token.AccessToken)
			// 		el, err := u.EVEUniverse().GetOrCreateLocationESI(ctx, o.ContextID)
			// 		if err != nil {
			// 			reportError(o, err)
			// 			return
			// 		}
			// 		contextItem.Widget = ui.MakeLocationLabel(el.ToShort(), u.InfoViewer().ShowLocation)
			// 		f.Refresh()
			// 	}()
			// }
			items := []*widget.FormItem{
				widget.NewFormItem("Owner", ui.MakeEveEntityActionLabel(
					&app.EveEntity{ID: corporationID, Name: corporationName, Category: app.EveEntityCorporation},
					u.InfoViewer().Show,
				)),
				widget.NewFormItem("Date", widget.NewLabel(o.Date.Format(app.DateTimeFormatWithSeconds))),
				widget.NewFormItem("Type", makeLabelWithWrap(o.RefTypeDisplay())),
				widget.NewFormItem("Amount", amount),
				widget.NewFormItem("Balance", widget.NewLabel(o.Balance.StringFunc("?", ui.FormatISKAmount))),
				widget.NewFormItem("Description", makeLabelWithWrap(o.Description)),
				widget.NewFormItem("Reason", makeLabelWithWrap(reason)),
			}
			if v, ok := o.FirstParty.Value(); ok {
				items = append(items, widget.NewFormItem(
					"First Party",
					ui.MakeEveEntityActionLabel(v, u.InfoViewer().Show),
				))
			}
			if v, ok := o.SecondParty.Value(); ok {
				items = append(items, widget.NewFormItem(
					"Second Party",
					ui.MakeEveEntityActionLabel(v, u.InfoViewer().Show),
				))
			}
			if v, ok := o.TaxReceiver.Value(); ok {
				items = append(items, widget.NewFormItem(
					"Tax",
					widget.NewLabel(o.Tax.StringFunc("?", ui.FormatISKAmount)),
				))
				items = append(items, widget.NewFormItem(
					"Tax Receiver",
					ui.MakeEveEntityActionLabel(v, u.InfoViewer().Show)),
				)
			}
			// items = append(items, contextItem)
			if u.IsDeveloperMode() {
				items = append(items, widget.NewFormItem("Ref ID", xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(refID))))
			}

			for _, it := range items {
				f.AppendItem(it)
			}
			ui.MakeDetailWindow(ui.MakeDetailWindowParams{
				Title:   title,
				Content: f,
				Window:  w,
			})
			w.Show()
		})
	})
}

func makeLabelWithWrap(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Wrapping = fyne.TextWrapWord
	return l
}
