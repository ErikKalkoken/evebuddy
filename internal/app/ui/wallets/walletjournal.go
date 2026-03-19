package wallets

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"
	"github.com/fnt-eve/goesi-openapi"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/contracts"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type walletJournalRow struct {
	amount           optional.Optional[float64]
	amountDisplay    []widget.RichTextSegment
	amountFormatted  string
	balance          optional.Optional[float64]
	balanceFormatted string
	date             time.Time
	dateFormatted    string
	description      string
	division         app.Division
	forCorporation   bool
	ownerID          int64
	ownerName        string
	reason           optional.Optional[string]
	refID            int64
	refType          string
	refTypeDisplay   string
}

func (r walletJournalRow) descriptionWithReason() string {
	if r.reason.IsEmpty() {
		return r.description
	}
	return fmt.Sprintf("[r] %s", r.description)
}

// WalletJournal is a widget for showing wallet journals for both characters and corporations.
type WalletJournal struct {
	widget.BaseWidget

	body         fyne.CanvasObject
	character    atomic.Pointer[app.Character]
	columnSorter *xwidget.ColumnSorter[walletJournalRow]
	corporation  atomic.Pointer[app.Corporation]
	division     app.Division
	footer       *widget.Label
	rows         []walletJournalRow
	rowsFiltered []walletJournalRow
	selectType   *kxwidget.FilterChipSelect
	sortButton   *xwidget.SortButton
	top          *widget.Label
	u            baseUI
}

func NewCharacterWalletJournal(u baseUI) *WalletJournal {
	a := newWalletJournal(u, app.DivisionZero)
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.Update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if a.character.Load().IDOrZero() != arg.CharacterID {
			return
		}
		if arg.Section == app.SectionCharacterWalletJournal {
			a.Update(ctx)
		}
	})
	return a
}

func NewCorporationWalletJournal(u baseUI, d app.Division) *WalletJournal {
	a := newWalletJournal(u, d)
	a.u.Signals().CurrentCorporationExchanged.AddListener(
		func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			a.Update(ctx)
		},
	)
	a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		if a.corporation.Load().IDOrZero() != arg.CorporationID {
			return
		}
		if arg.Section == app.CorporationSectionWalletJournal(d) {
			a.Update(ctx)
		}
	})
	return a
}

const (
	walletJournalColDate = iota + 1
	walletJournalColType
	walletJournalColAmount
	walletJournalColBalance
	walletJournalColDescription
)

func newWalletJournal(u baseUI, division app.Division) *WalletJournal {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[walletJournalRow]{{
		ID:    walletJournalColDate,
		Label: "Date",
		Width: 150,
		Sort: func(a, b walletJournalRow) int {
			return a.date.Compare(b.date)
		},
		Update: func(r walletJournalRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.dateFormatted)
		},
	}, {
		ID:    walletJournalColType,
		Label: "Type",
		Width: 150,
		Sort: func(a, b walletJournalRow) int {
			return strings.Compare(a.refType, b.refType)
		},
		Update: func(r walletJournalRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.refTypeDisplay)
		},
	}, {
		ID:    walletJournalColAmount,
		Label: "Amount",
		Width: 200,
		Sort: func(a, b walletJournalRow) int {
			return optional.Compare(a.amount, b.amount)
		},
		Update: func(r walletJournalRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).Set(r.amountDisplay)
		},
	}, {
		ID:    walletJournalColBalance,
		Label: "Balance",
		Width: 200,
		Update: func(r walletJournalRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.balance.StringFunc("?", func(v float64) string {
				return humanize.FormatFloat(ui.FloatFormat, v)
			}), widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
			},
			)
		},
	}, {
		ID:    walletJournalColDescription,
		Label: "Description",
		Width: 450,
		Update: func(r walletJournalRow, co fyne.CanvasObject) {
			co.(*xwidget.RichText).SetWithText(r.descriptionWithReason())
		},
	}})
	a := &WalletJournal{
		columnSorter: xwidget.NewColumnSorter(columns, walletJournalColDate, xwidget.SortDesc),
		division:     division,
		footer:       ui.NewLabelWithTruncation(""),
		top:          ui.NewLabelWithTruncation(""),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	if a.u.IsMobile() {
		a.body = a.makeDataList()
	} else {
		a.body = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync,
			func(_ int, r walletJournalRow) {
				if a.isCorporation() {
					ShowCorporationWalletJournalEntryWindowAsync(a.u, r.ownerID, r.ownerName, r.division, r.refID)
				} else {
					ShowCharacterWalletJournalEntryWindowAsync(a.u, r.ownerID, r.ownerName, r.refID)
				}
			},
		)
	}
	a.selectType = kxwidget.NewFilterChipSelectWithSearch("Type", []string{}, func(string) {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())
	return a
}

func (a *WalletJournal) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectType)
	if a.u.IsMobile() {
		filter.Add(a.sortButton)
	}
	c := container.NewBorder(
		container.NewHScroll(filter),
		a.footer,
		nil,
		nil,
		a.body,
	)
	return widget.NewSimpleRenderer(c)
}
func (a *WalletJournal) isCorporation() bool {
	return a.division != app.DivisionZero
}

func (a *WalletJournal) makeDataList() *xwidget.StripedList {
	p := theme.Padding()
	l := xwidget.NewStripedList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			date := widget.NewLabel("Template")
			date.Truncation = fyne.TextTruncateClip
			balance := widget.NewLabel("Template")
			balance.Alignment = fyne.TextAlignTrailing
			refType := widget.NewLabel("Template")
			refType.Truncation = fyne.TextTruncateClip
			value := widget.NewLabel("Template")
			value.Alignment = fyne.TextAlignTrailing
			description := widget.NewLabel("Template")
			description.Truncation = fyne.TextTruncateClip
			return container.New(layout.NewCustomPaddedVBoxLayout(-p),
				container.NewBorder(nil, nil, nil, value, date),
				container.NewBorder(nil, nil, nil, balance, refType),
				description,
			)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id < 0 || id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			c := co.(*fyne.Container).Objects

			b0 := c[0].(*fyne.Container).Objects
			b0[0].(*widget.Label).SetText(r.dateFormatted)
			amount := b0[1].(*widget.Label)
			amount.Text = r.amountFormatted
			amount.Importance = importanceISKAmount(r.amount)
			amount.Refresh()

			b1 := c[1].(*fyne.Container).Objects
			b1[0].(*widget.Label).SetText(r.refTypeDisplay)
			b1[1].(*widget.Label).SetText(r.balanceFormatted)

			c[2].(*widget.Label).SetText(r.description)
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		defer l.UnselectAll()
		if id < 0 || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		if a.isCorporation() {
			ShowCorporationWalletJournalEntryWindowAsync(a.u, r.ownerID, r.ownerName, r.division, r.refID)
		} else {
			ShowCharacterWalletJournalEntryWindowAsync(a.u, r.ownerID, r.ownerName, r.refID)
		}
	}
	l.HideSeparators = true
	return l
}

func (a *WalletJournal) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	et := a.selectType.Selected
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		if et != "" {
			rows = slices.DeleteFunc(rows, func(r walletJournalRow) bool {
				return r.refTypeDisplay != et
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		typeOptions := xslices.Map(rows, func(r walletJournalRow) string {
			return r.refTypeDisplay
		})
		footer := fmt.Sprintf("Showing %s / %s entries", ihumanize.Comma(len(rows)), ihumanize.Comma(totalRows))

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectType.SetOptions(typeOptions)
			a.rowsFiltered = rows
			a.body.Refresh()
		})
	}()
}

func (a *WalletJournal) Update(ctx context.Context) {
	if a.isCorporation() {
		a.updateCorporation(ctx)
	} else {
		a.updateCharacter(ctx)
	}
}

func (a *WalletJournal) updateCharacter(ctx context.Context) {
	setInfo := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.footer.Text, a.footer.Importance = s, i
			a.footer.Refresh()
		})
	}
	reset := func() {
		a.rows = xslices.Reset(a.rows)
		a.filterRowsAsync(-1)
	}
	character := a.character.Load()
	if character == nil {
		reset()
		setInfo("No character", widget.LowImportance)
		return
	}
	hasData, err := a.u.Character().HasSection(ctx, character.ID, app.SectionCharacterWalletJournal)
	if err != nil {
		reset()
		setInfo("Error: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	if !hasData {
		reset()
		setInfo("No data", widget.WarningImportance)
		return
	}
	rows, err := a.fetchCharacterRows(ctx, character)
	if err != nil {
		reset()
		setInfo("Error: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync(-1)
	})
}

func (a *WalletJournal) fetchCharacterRows(ctx context.Context, character *app.Character) ([]walletJournalRow, error) {
	oo, err := a.u.Character().ListWalletJournalEntries(ctx, character.ID)
	if err != nil {
		return nil, err
	}
	var rows []walletJournalRow
	for _, o := range oo {
		r := walletJournalRow{
			amount: o.Amount,
			amountFormatted: o.Amount.StringFunc("?", func(v float64) string {
				return humanize.FormatFloat(ui.FloatFormat, v)
			}),
			balance: o.Balance,
			balanceFormatted: o.Balance.StringFunc("?", func(v float64) string {
				return humanize.FormatFloat(ui.FloatFormat, v)
			}),
			ownerID:        character.ID,
			ownerName:      character.NameOrZero(),
			date:           o.Date,
			dateFormatted:  o.Date.Format(app.DateTimeFormat),
			description:    o.Description,
			reason:         o.Reason,
			refID:          o.RefID,
			refType:        o.RefType,
			refTypeDisplay: o.RefTypeDisplay(),
		}
		r.amountDisplay = xwidget.RichTextSegmentsFromText(
			r.amountFormatted,
			widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				ColorName: colorISKAmount(o.Amount),
			},
		)
		rows = append(rows, r)
	}
	return rows, nil
}

func (a *WalletJournal) updateCorporation(ctx context.Context) {
	setInfo := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.footer.Text, a.footer.Importance = s, i
			a.footer.Refresh()
		})
	}
	reset := func() {
		a.rows = xslices.Reset(a.rows)
		a.filterRowsAsync(-1)
	}
	corporation := a.corporation.Load()
	if corporation == nil {
		reset()
		setInfo("No corporation", widget.LowImportance)
		return
	}
	hasData, err := a.u.Corporation().HasSection(ctx, corporation.ID, app.CorporationSectionWalletJournal(a.division))
	if err != nil {
		reset()
		setInfo("Error: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	if !hasData {
		reset()
		setInfo("No data", widget.WarningImportance)
		return
	}
	rows, err := a.fetchCorporationRows(ctx, corporation, a.division)
	if err != nil {
		reset()
		setInfo("Error: "+a.u.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync(-1)
	})
}

func (a *WalletJournal) fetchCorporationRows(ctx context.Context, corporation *app.Corporation, division app.Division) ([]walletJournalRow, error) {
	entries, err := a.u.Corporation().ListWalletJournalEntries(ctx, corporation.ID, division)
	if err != nil {
		return nil, err
	}
	var rows []walletJournalRow
	for _, o := range entries {
		r := walletJournalRow{
			amount: o.Amount,
			amountFormatted: o.Amount.StringFunc("?", func(v float64) string {
				return humanize.FormatFloat(ui.FloatFormat, v)
			}),
			balance: o.Balance,
			balanceFormatted: o.Balance.StringFunc("?", func(v float64) string {
				return humanize.FormatFloat(ui.FloatFormat, v)
			}),
			ownerID:        corporation.ID,
			ownerName:      corporation.NameOrZero(),
			forCorporation: true,
			division:       division,
			date:           o.Date,
			dateFormatted:  o.Date.Format(app.DateTimeFormat),
			description:    o.Description,
			reason:         o.Reason,
			refID:          o.RefID,
			refType:        o.RefType,
			refTypeDisplay: o.RefTypeDisplay(),
		}

		r.amountDisplay = xwidget.RichTextSegmentsFromText(
			r.amountFormatted,
			widget.RichTextStyle{
				Alignment: fyne.TextAlignTrailing,
				ColorName: colorISKAmount(o.Amount),
			},
		)
		rows = append(rows, r)
	}
	return rows, nil
}

// ShowCharacterWalletJournalEntryWindowAsync shows a wallet journal entry for a character in a new window.
func ShowCharacterWalletJournalEntryWindowAsync(u baseUI, characterID int64, characterName string, refID int64) {
	key := fmt.Sprintf("walletjournalentry-%d-%d", characterID, refID)
	title := fmt.Sprintf("Character Wallet Transaction #%d", refID)
	w, created := u.GetOrCreateWindow(key, title, characterName)
	if !created {
		w.Show()
		return
	}

	go func() {
		o, err := u.Character().GetWalletJournalEntry(context.Background(), characterID, refID)
		if err != nil {
			ui.ShowErrorAndLog("Failed to show wallet transaction", err, u.IsDeveloperMode(), u.MainWindow())
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
				contextDefaultWidget.SetText("Failed to load related item: " + u.ErrorDisplay(err))
			}
			// TODO: Add support for industry jobs
			if v, ok := o.ContextIDType.Value(); ok {
				switch v {
				case "alliance_id", "character_id", "corporation_id", "system_id", "type_id":
					go func() {
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
					}()
				case "contract_id":
					go func() {
						c, err := u.Character().GetContract(context.Background(), characterID, o.ContextID.ValueOrZero())
						if err != nil {
							reportError(o, err)
							return
						}
						fyne.Do(func() {
							contextItem.Text = "Related contract"
							contextItem.Widget = ui.MakeLinkLabelWithWrap(c.NameDisplay(), func() {
								go contracts.ShowCharacterContract2(context.Background(), u, c.CharacterID, c.ContractID)
							})
							f.Refresh()
						})
					}()
				case "market_transaction_id":
					contextItem.Text = "Related market transaction"
					contextItem.Widget = ui.MakeLinkLabelWithWrap(fmt.Sprintf("#%d", o.ContextID.ValueOrZero()), func() {
						ShowCharacterWalletTransactionWindowAsync(u, characterID, characterName, o.ContextID.ValueOrZero())
					})
					f.Refresh()
				case "station_id", "structure_id":
					contextItem.Text = "Related location"
					go func() {
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
							contextItem.Widget = ui.MakeLocationLabel(el.ToShort(), u.InfoViewer().ShowLocation)
							f.Refresh()
						})
					}()
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
			if !o.ContextIDType.IsEmpty() {
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
	}()
}

// ShowCorporationWalletJournalEntryWindowAsync shows a wallet journal entry for a corporation in a new window.
func ShowCorporationWalletJournalEntryWindowAsync(u baseUI, corporationID int64, corporationName string, division app.Division, refID int64) {
	key := fmt.Sprintf("walletjournalentry-%d-%d", corporationID, refID)
	title := fmt.Sprintf("Corporation Wallet Transaction #%d", refID)
	w, created := u.GetOrCreateWindow(key, title, corporationName)
	if !created {
		w.Show()
		return
	}

	go func() {
		o, err := u.Corporation().GetWalletJournalEntry(context.Background(), corporationID, division, refID)
		if err != nil {
			ui.ShowErrorAndLog("Failed to show wallet transaction", err, u.IsDeveloperMode(), u.MainWindow())
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
				widget.NewFormItem("Owner", ui.MakeCharacterActionLabel(
					corporationID,
					corporationName,
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
			if v, ok := o.FirstParty.Value(); ok {
				items = append(items, widget.NewFormItem(
					"Second Party",
					ui.MakeEveEntityActionLabel(v, u.InfoViewer().Show),
				))
			}
			if v, ok := o.FirstParty.Value(); ok {
				items = append(items, widget.NewFormItem(
					"Tax",
					widget.NewLabel(o.Tax.StringFunc("?", ui.FormatISKAmount)),
				))
				items = append(items, widget.NewFormItem(
					"Tax Receiver",
					ui.MakeEveEntityActionLabel(v, u.InfoViewer().Show)),
				)
			}
			items = append(items, contextItem)
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
	}()
}

func colorISKAmount(amount optional.Optional[float64]) fyne.ThemeColorName {
	var color fyne.ThemeColorName
	if v, ok := amount.Value(); !ok {
		color = theme.ColorNameDisabled
	} else if v < 0 {
		color = theme.ColorNameError
	} else if v > 0 {
		color = theme.ColorNameSuccess
	} else {
		color = theme.ColorNameForeground
	}
	return color
}

func importanceISKAmount(amount optional.Optional[float64]) widget.Importance {
	if v, ok := amount.Value(); !ok {
		return widget.LowImportance
	} else if v > 0 {
		return widget.SuccessImportance
	} else if v < 0 {
		return widget.DangerImportance
	}
	return widget.MediumImportance
}

func makeLabelWithWrap(s string) *widget.Label {
	l := widget.NewLabel(s)
	l.Wrapping = fyne.TextWrapWord
	return l
}
