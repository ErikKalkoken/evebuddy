package screens

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// ShowCharacterWalletTransactionWindowAsync shows the detail of a character wallet transaction in a window.
func ShowCharacterWalletTransactionWindowAsync(u baseUI, characterID int64, characterName string, transactionID int64) {
	windowID := fmt.Sprintf("wallettransaction-%d-%d", characterID, transactionID)
	title := fmt.Sprintf("Character Market Transaction #%d", transactionID)
	w, created := u.GetOrCreateWindow(windowID, title, characterName)
	if !created {
		w.Show()
		return
	}

	runAsync(func() {
		o, err := u.Character().GetWalletTransactions(context.Background(), characterID, transactionID)
		if err != nil {
			fyne.Do(func() {
				u.DestroyWindow(windowID)
				ui.ShowErrorAndLog("Failed to show market transaction", err, u.IsDeveloperMode(), u.MainWindow())
			})
			return
		}
		fyne.Do(func() {
			var activity string
			total := widget.NewLabel(ui.FormatISKAmount(o.Total()))
			if o.IsBuy {
				total.Importance = widget.DangerImportance
				activity = marketTransactionActivityBuy
			} else {
				total.Importance = widget.SuccessImportance
				activity = marketTransactionActivitySell
			}
			items := []*widget.FormItem{
				widget.NewFormItem("Owner", ui.MakeCharacterActionLabel(
					characterID,
					characterName,
					u.InfoViewer().Show,
				)),
				widget.NewFormItem("Date", widget.NewLabel(o.Date.Format(app.DateTimeFormatWithSeconds))),
				widget.NewFormItem("Activity", widget.NewLabel(activity)),
				widget.NewFormItem("Quantity", widget.NewLabel(humanize.Comma(int64(o.Quantity)))),
				widget.NewFormItem("Type", ui.MakeLinkLabelWithWrap(o.Type.Name, func() {
					u.InfoViewer().Show(o.Type.ToEveEntity())
				})),
				widget.NewFormItem("Unit price", widget.NewLabel(ui.FormatISKAmount(o.UnitPrice))),
				widget.NewFormItem("Total", total),
				widget.NewFormItem("Client", ui.MakeEveEntityActionLabel(o.Client, u.InfoViewer().Show)),
				widget.NewFormItem("Location", ui.MakeLocationLabel(o.Location, u.InfoViewer().ShowLocation)),
				// widget.NewFormItem("Related Journal Entry", ui.MakeLinkLabelWithWrap(
				// 	fmt.Sprintf("#%d", o.JournalRefID), func() {
				// 		showCharacterWalletJournalEntryWindow(u, characterID, o.JournalRefID)
				// 	},
				// )),
			}

			if u.IsDeveloperMode() {
				items = append(items, widget.NewFormItem(
					"Transaction ID",
					xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(transactionID)),
				))
			}
			f := widget.NewForm(items...)
			f.Orientation = widget.Adaptive
			ui.MakeDetailWindow(ui.MakeDetailWindowParams{
				Content: f,
				ImageAction: func() {
					u.InfoViewer().ShowType(o.Type.ID, 0)
				},
				ImageLoader: func(setter func(r fyne.Resource)) {
					u.EVEImage().InventoryTypeIconAsync(o.Type.ID, 256, setter)
				},
				Title:  title,
				Window: w,
			})
			w.Show()
		})
	})
}

// ShowCorporationWalletTransactionWindowAsync shows the detail of a corporation wallet transaction in a window.
func ShowCorporationWalletTransactionWindowAsync(u baseUI, corporationID int64, corporationName string, division app.Division, transactionID int64) {
	windowID := fmt.Sprintf("wallettransaction-%d-%d", corporationID, transactionID)
	title := fmt.Sprintf("Corporation Market Transaction #%d", transactionID)
	w, created := u.GetOrCreateWindow(windowID, title, corporationName)
	if !created {
		w.Show()
		return
	}

	runAsync(func() {
		o, err := u.Corporation().GetWalletTransaction(context.Background(), corporationID, division, transactionID)
		if err != nil {
			fyne.Do(func() {
				u.DestroyWindow(windowID)
				ui.ShowErrorAndLog("Failed to show market transaction", err, u.IsDeveloperMode(), u.MainWindow())
			})
			return
		}
		fyne.Do(func() {
			var activity string
			totalLabel := widget.NewLabel(ui.FormatISKAmount(o.Total()))
			if o.IsBuy {
				totalLabel.Importance = widget.DangerImportance
				activity = marketTransactionActivityBuy
			} else {
				totalLabel.Importance = widget.SuccessImportance
				activity = marketTransactionActivitySell
			}
			items := []*widget.FormItem{
				widget.NewFormItem("Owner", ui.MakeEveEntityActionLabel(
					&app.EveEntity{ID: corporationID, Name: corporationName, Category: app.EveEntityCorporation},
					u.InfoViewer().Show,
				)),
				widget.NewFormItem("Date", widget.NewLabel(o.Date.Format(app.DateTimeFormatWithSeconds))),
				widget.NewFormItem("Activity", widget.NewLabel(activity)),
				widget.NewFormItem("Quantity", widget.NewLabel(humanize.Comma(int64(o.Quantity)))),
				widget.NewFormItem("Type", ui.MakeLinkLabelWithWrap(o.Type.Name, func() {
					u.InfoViewer().Show(o.Type.ToEveEntity())
				})),
				widget.NewFormItem("Unit price", widget.NewLabel(ui.FormatISKAmount(o.UnitPrice))),
				widget.NewFormItem("Total", totalLabel),
				widget.NewFormItem("Client", ui.MakeEveEntityActionLabel(o.Client, u.InfoViewer().Show)),
				widget.NewFormItem("Location", ui.MakeLocationLabel(o.Location, u.InfoViewer().ShowLocation)),
				widget.NewFormItem("Related Journal Entry", ui.MakeLinkLabelWithWrap(
					fmt.Sprintf("#%d", o.JournalRefID), func() {
						ShowCorporationWalletJournalEntryWindowAsync(u, corporationID, corporationName, division, o.JournalRefID)
					},
				)),
			}

			if u.IsDeveloperMode() {
				items = append(items, widget.NewFormItem(
					"Transaction ID",
					xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(transactionID)),
				))
			}
			f := widget.NewForm(items...)
			f.Orientation = widget.Adaptive
			ui.MakeDetailWindow(ui.MakeDetailWindowParams{
				Content: f,
				ImageAction: func() {
					u.InfoViewer().ShowType(o.Type.ID, 0)
				},
				ImageLoader: func(setter func(r fyne.Resource)) {
					u.EVEImage().InventoryTypeIconAsync(o.Type.ID, 256, setter)
				},
				Title:  title,
				Window: w,
			})
			w.Show()
		})
	})
}
