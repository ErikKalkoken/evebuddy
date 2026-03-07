package ui

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
	"github.com/ErikKalkoken/evebuddy/internal/app/uiservices"
	"github.com/ErikKalkoken/evebuddy/internal/app/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/app/xwindow"
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
	characterID      int64
	corporationID    int64
	date             time.Time
	dateFormatted    string
	description      string
	division         app.Division
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
	u            uiservices.UIServices
}

func NewCharacterWalletJournal(u uiservices.UIServices) *WalletJournal {
	a := newWalletJournal(u, app.DivisionZero)
	a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.Update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.CharacterID {
			return
		}
		if arg.Section == app.SectionCharacterWalletJournal {
			a.Update(ctx)
		}
	})
	return a
}

func NewCorporationWalletJournal(u uiservices.UIServices, d app.Division) *WalletJournal {
	a := newWalletJournal(u, d)
	a.u.Signals().CurrentCorporationExchanged.AddListener(
		func(ctx context.Context, c *app.Corporation) {
			a.corporation.Store(c)
			a.Update(ctx)
		},
	)
	a.u.Signals().CorporationSectionChanged.AddListener(func(ctx context.Context, arg app.CorporationSectionUpdated) {
		if corporationIDOrZero(a.corporation.Load()) != arg.CorporationID {
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

func newWalletJournal(u uiservices.UIServices, division app.Division) *WalletJournal {
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
				return humanize.FormatFloat(app.FloatFormat, v)
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
		footer:       newLabelWithTruncation(),
		top:          newLabelWithTruncation(),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	if app.IsMobile() {
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
					ShowCorporationWalletJournalEntryWindowAsync(a.u, r.corporationID, r.division, r.refID)
				} else {
					ShowCharacterWalletJournalEntryWindowAsync(a.u, r.characterID, r.refID)
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
	if app.IsMobile() {
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
			ShowCorporationWalletJournalEntryWindowAsync(a.u, r.corporationID, r.division, r.refID)
		} else {
			ShowCharacterWalletJournalEntryWindowAsync(a.u, r.characterID, r.refID)
		}
	}
	l.HideSeparators = true
	return l
}

func (a *WalletJournal) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	type_ := a.selectType.Selected
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		if type_ != "" {
			rows = slices.DeleteFunc(rows, func(r walletJournalRow) bool {
				return r.refTypeDisplay != type_
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
	var err error
	var rows []walletJournalRow
	characterID := characterIDOrZero(a.character.Load())
	hasData := a.u.StatusCache().HasCharacterSection(characterID, app.SectionCharacterWalletJournal)
	if hasData {
		rows2, err2 := a.fetchCharacterRows(ctx, characterID)
		if err2 != nil {
			slog.Error("Failed to refresh wallet journal UI", "err", err2)
			err = err2
		} else {
			rows = rows2
		}
	}
	t, i := makeTopText(characterID, hasData, err, func() (string, widget.Importance) {
		return "", widget.MediumImportance
	})
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync(-1)
	})
}

func (a *WalletJournal) updateCorporation(ctx context.Context) {
	var err error
	var rows []walletJournalRow
	corporationID := corporationIDOrZero(a.corporation.Load())
	hasData := a.u.StatusCache().HasCorporationSection(corporationID, app.CorporationSectionWalletJournal(a.division))
	if hasData {
		rows2, err2 := a.fetchCorporationRows(ctx, corporationID, a.division)
		if err2 != nil {
			slog.Error("Failed to refresh wallet journal UI", "err", err2)
			err = err2
		} else {
			rows = rows2
		}
	}
	t, i := makeTopText(corporationID, hasData, err, func() (string, widget.Importance) {
		return "", widget.MediumImportance
	})
	fyne.Do(func() {
		a.top.Text = t
		a.top.Importance = i
		a.top.Refresh()
	})
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync(-1)
	})
}

func (a *WalletJournal) fetchCharacterRows(ctx context.Context, characterID int64) ([]walletJournalRow, error) {
	entries, err := a.u.Character().ListWalletJournalEntries(ctx, characterID)
	if err != nil {
		return nil, err
	}
	var rows []walletJournalRow
	for _, o := range entries {
		r := walletJournalRow{
			amount: o.Amount,
			amountFormatted: o.Amount.StringFunc("?", func(v float64) string {
				return humanize.FormatFloat(app.FloatFormat, v)
			}),
			balance: o.Balance,
			balanceFormatted: o.Balance.StringFunc("?", func(v float64) string {
				return humanize.FormatFloat(app.FloatFormat, v)
			}),
			characterID:    characterID,
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

func (a *WalletJournal) fetchCorporationRows(ctx context.Context, corporationID int64, division app.Division) ([]walletJournalRow, error) {
	entries, err := a.u.Corporation().ListWalletJournalEntries(ctx, corporationID, division)
	if err != nil {
		return nil, err
	}
	var rows []walletJournalRow
	for _, o := range entries {
		r := walletJournalRow{
			amount: o.Amount,
			amountFormatted: o.Amount.StringFunc("?", func(v float64) string {
				return humanize.FormatFloat(app.FloatFormat, v)
			}),
			balance: o.Balance,
			balanceFormatted: o.Balance.StringFunc("?", func(v float64) string {
				return humanize.FormatFloat(app.FloatFormat, v)
			}),
			corporationID:  corporationID,
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
func ShowCharacterWalletJournalEntryWindowAsync(u uiservices.UIServices, characterID int64, refID int64) {
	title := fmt.Sprintf("Character Wallet Transaction #%d", refID)
	w, created := u.GetOrCreateWindow(
		fmt.Sprintf("walletjournalentry-%d-%d", characterID, refID),
		title,
		u.StatusCache().CharacterName(characterID),
	)
	if !created {
		w.Show()
		return
	}

	go func() {
		o, err := u.Character().GetWalletJournalEntry(context.Background(), characterID, refID)
		if err != nil {
			xdialog.ShowErrorAndLog("Failed to show wallet transaction", err, u.MainWindow())
			return
		}

		fyne.Do(func() {
			f := widget.NewForm()
			f.Orientation = widget.Adaptive

			amount := widget.NewLabel(o.Amount.StringFunc("?", formatISKAmount))
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
				contextDefaultWidget.SetText("Failed to load related item: " + app.ErrorDisplay(err))
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
							contextItem.Widget = makeEveEntityActionLabel(ee, u.InfoWindow().ShowEntity)
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
							contextItem.Widget = makeLinkLabelWithWrap(c.NameDisplay(), func() {
								showCharacterContractWindow(u, c.CharacterID, c.ContractID)
							})
							f.Refresh()
						})
					}()
				case "market_transaction_id":
					contextItem.Text = "Related market transaction"
					contextItem.Widget = makeLinkLabelWithWrap(fmt.Sprintf("#%d", o.ContextID.ValueOrZero()), func() {
						ShowCharacterWalletTransactionWindowAsync(u, o.CharacterID, o.ContextID.ValueOrZero())
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
							contextItem.Widget = makeLocationLabel(el.ToShort(), u.InfoWindow().ShowLocation)
							f.Refresh()
						})
					}()
				}
			}
			items := []*widget.FormItem{
				widget.NewFormItem("Owner", makeCharacterActionLabel(
					characterID,
					u.StatusCache().CharacterName(characterID),
					u.InfoWindow().ShowEntity,
				)),
				widget.NewFormItem("Date", widget.NewLabel(o.Date.Format(app.DateTimeFormatWithSeconds))),
				widget.NewFormItem("Type", makeLabelWithWrap(o.RefTypeDisplay())),
				widget.NewFormItem("Amount", amount),
				widget.NewFormItem("Balance", widget.NewLabel(o.Balance.StringFunc("?", formatISKAmount))),
				widget.NewFormItem("Description", makeLabelWithWrap(o.Description)),
				widget.NewFormItem("Reason", makeLabelWithWrap(reason)),
			}
			if v, ok := o.FirstParty.Value(); ok {
				items = append(items, widget.NewFormItem(
					"First Party",
					makeEveEntityActionLabel(v, u.InfoWindow().ShowEntity),
				))
			}
			if v, ok := o.SecondParty.Value(); ok {
				items = append(items, widget.NewFormItem(
					"Second Party",
					makeEveEntityActionLabel(v, u.InfoWindow().ShowEntity),
				))
			}
			if v, ok := o.TaxReceiver.Value(); ok {
				items = append(items, widget.NewFormItem(
					"Tax",
					widget.NewLabel(o.Tax.StringFunc("?", formatISKAmount)),
				))
				items = append(items, widget.NewFormItem(
					"Tax Receiver",
					makeEveEntityActionLabel(v, u.InfoWindow().ShowEntity)),
				)
			}
			if !o.ContextIDType.IsEmpty() {
				items = append(items, contextItem)
			}
			if app.IsDeveloperMode() {
				items = append(items, widget.NewFormItem("Ref ID", xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(refID))))
			}

			for _, it := range items {
				f.AppendItem(it)
			}
			xwindow.Set(xwindow.Params{
				Title:   title,
				Content: f,
				Window:  w,
			})
			w.Show()
		})
	}()
}

// ShowCorporationWalletJournalEntryWindowAsync shows a wallet journal entry for a corporation in a new window.
func ShowCorporationWalletJournalEntryWindowAsync(u uiservices.UIServices, corporationID int64, division app.Division, refID int64) {
	title := fmt.Sprintf("Corporation Wallet Transaction #%d", refID)
	w, created := u.GetOrCreateWindow(
		fmt.Sprintf("walletjournalentry-%d-%d", corporationID, refID),
		title,
		u.StatusCache().CorporationName(corporationID),
	)
	if !created {
		w.Show()
		return
	}

	go func() {
		o, err := u.Corporation().GetWalletJournalEntry(context.Background(), corporationID, division, refID)
		if err != nil {
			xdialog.ShowErrorAndLog("Failed to show wallet transaction", err, u.MainWindow())
			return
		}

		fyne.Do(func() {
			f := widget.NewForm()
			f.Orientation = widget.Adaptive

			amount := widget.NewLabel(o.Amount.StringFunc("?", formatISKAmount))
			amount.Importance = importanceISKAmount(o.Amount)
			reason := o.Reason.ValueOrFallback("-")

			contextDefaultWidget := widget.NewLabel("?")
			contextItem := widget.NewFormItem("Related item", contextDefaultWidget)
			// ctx := context.Background()
			// reportError := func(o *app.CorporationWalletJournalEntry, err error) {
			// 	slog.Error("Failed to fetch related context", "contextIDType", o.ContextIDType, "contextID", o.ContextID, "error", err)
			// 	contextDefaultWidget.SetText("Failed to load related item: " + app.ErrorDisplay(err))
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
			// 		contextItem.Widget = makeEveEntityActionLabel(ee, u.InfoWindow().ShowEntity)
			// 		f.Refresh()
			// 	}()
			// case "contract_id":
			// 	c, err := u.Character().GetContract(ctx, corporationID, int64(o.ContextID))
			// 	if err != nil {
			// 		reportError(o, err)
			// 		break
			// 	}
			// 	contextItem.Text = "Related contract"
			// 	contextItem.Widget = makeLinkLabelWithWrap(c.NameDisplay(), func() {
			// 		showContract(u, c.CorporationID, c.ContractID)
			// 	})
			// case "market_transaction_id":
			// 	contextItem.Text = "Related market transaction"
			// 	contextItem.Widget = makeLinkLabelWithWrap(fmt.Sprintf("#%d", o.ContextID), func() {
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
			// 		contextItem.Widget = makeLocationLabel(el.ToShort(), u.InfoWindow().ShowLocation)
			// 		f.Refresh()
			// 	}()
			// }
			items := []*widget.FormItem{
				widget.NewFormItem("Owner", makeCharacterActionLabel(
					corporationID,
					u.StatusCache().CorporationName(corporationID),
					u.InfoWindow().ShowEntity,
				)),
				widget.NewFormItem("Date", widget.NewLabel(o.Date.Format(app.DateTimeFormatWithSeconds))),
				widget.NewFormItem("Type", makeLabelWithWrap(o.RefTypeDisplay())),
				widget.NewFormItem("Amount", amount),
				widget.NewFormItem("Balance", widget.NewLabel(o.Balance.StringFunc("?", formatISKAmount))),
				widget.NewFormItem("Description", makeLabelWithWrap(o.Description)),
				widget.NewFormItem("Reason", makeLabelWithWrap(reason)),
			}
			if v, ok := o.FirstParty.Value(); ok {
				items = append(items, widget.NewFormItem(
					"First Party",
					makeEveEntityActionLabel(v, u.InfoWindow().ShowEntity),
				))
			}
			if v, ok := o.FirstParty.Value(); ok {
				items = append(items, widget.NewFormItem(
					"Second Party",
					makeEveEntityActionLabel(v, u.InfoWindow().ShowEntity),
				))
			}
			if v, ok := o.FirstParty.Value(); ok {
				items = append(items, widget.NewFormItem(
					"Tax",
					widget.NewLabel(o.Tax.StringFunc("?", formatISKAmount)),
				))
				items = append(items, widget.NewFormItem(
					"Tax Receiver",
					makeEveEntityActionLabel(v, u.InfoWindow().ShowEntity)),
				)
			}
			items = append(items, contextItem)
			if app.IsDeveloperMode() {
				items = append(items, widget.NewFormItem("Ref ID", xwidget.NewTappableLabelWithClipboardCopy(fmt.Sprint(refID))))
			}

			for _, it := range items {
				f.AppendItem(it)
			}
			xwindow.Set(xwindow.Params{
				Title:   title,
				Content: f,
				Window:  w,
			})
			w.Show()
		})
	}()
}
