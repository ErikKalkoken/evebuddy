package wallets

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type overviewRow struct {
	assets        optional.Optional[float64]
	assetsDisplay string
	characterID   int64
	characterName string
	tags          set.Set[string]
	total         optional.Optional[float64]
	totalDisplay  string
	wallet        optional.Optional[float64]
	walletDisplay string
	searchTarget  string
	isTotal       bool
}

func (r overviewRow) eveEntity() *app.EveEntity {
	return &app.EveEntity{
		Category: app.EveEntityCharacter,
		ID:       r.characterID,
		Name:     r.characterName,
	}
}

type Overview struct {
	widget.BaseWidget

	OnUpdate func(expired int)

	footer       *widget.Label
	columnSorter *xwidget.ColumnSorter[overviewRow]
	main         fyne.CanvasObject
	rows         []overviewRow
	rowsFiltered []overviewRow
	search       *widget.Entry
	selectTag    *kxwidget.FilterChipSelect
	sortButton   *xwidget.SortButton
	u            baseUI
}

const (
	overviewColCharacter = iota + 1
	overviewColTags
	overviewColAssets
	overviewColWallet
	overviewColTotal
)

const valueWidth = 150

func NewOverview(u baseUI) *Overview {
	columns := xwidget.NewDataColumns([]xwidget.DataColumn[overviewRow]{
		ui.MakeEveEntityColumn(ui.MakeEveEntityColumnParams[overviewRow]{
			ColumnID: overviewColCharacter,
			EIS:      u.EVEImage(),
			GetEntity: func(r overviewRow) *app.EveEntity {
				return &app.EveEntity{
					ID:       r.characterID,
					Name:     r.characterName,
					Category: app.EveEntityCharacter,
				}
			},
			IsAvatar: true,
			Label:    "Character",
		}), {
			ID:    overviewColTags,
			Label: "Tags",
			Width: 150,
			Update: func(r overviewRow, co fyne.CanvasObject) {
				s := strings.Join(slices.Sorted(r.tags.All()), ", ")
				co.(*xwidget.RichText).SetWithText(s)
			},
		}, {
			ID:    overviewColAssets,
			Label: "Assets",
			Width: valueWidth,
			Update: func(r overviewRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.assetsDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b overviewRow) int {
				return optional.Compare(a.assets, b.assets)
			},
		}, {
			ID:    overviewColWallet,
			Label: "Wallet",
			Width: valueWidth,
			Update: func(r overviewRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.walletDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b overviewRow) int {
				return optional.Compare(a.wallet, b.wallet)
			},
		}, {
			ID:    overviewColTotal,
			Label: "Total",
			Width: valueWidth,
			Update: func(r overviewRow, co fyne.CanvasObject) {
				co.(*xwidget.RichText).SetWithText(r.totalDisplay, widget.RichTextStyle{
					Alignment: fyne.TextAlignTrailing,
					TextStyle: fyne.TextStyle{Bold: r.isTotal},
				})
			},
			Sort: func(a, b overviewRow) int {
				return optional.Compare(a.total, b.total)
			},
		}})
	a := &Overview{
		columnSorter: xwidget.NewColumnSorter(columns, overviewColCharacter, xwidget.SortAsc),
		footer:       widget.NewLabel(""),
		search:       widget.NewEntry(),
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.search.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.search.SetText("")
		a.filterRowsAsync(-1)
	})
	a.search.OnChanged = func(_ string) {
		a.filterRowsAsync(-1)
	}
	a.search.PlaceHolder = "Search characters"
	if a.u.IsMobile() {
		a.main = xwidget.MakeDataList(
			columns,
			&a.rowsFiltered,
			func(col int, r overviewRow) []widget.RichTextSegment {
				var s []widget.RichTextSegment
				// switch col {
				// case overviewColCharacter:
				// 	s = r.jc.Location.DisplayRichText()
				// case clonesColT:
				// 	s = xwidget.RichTextSegmentsFromText(r.jc.Location.RegionName())
				// case clonesColImplants:
				// 	s = xwidget.RichTextSegmentsFromText(fmt.Sprint(r.jc.ImplantsCount))
				// case clonesColCharacter:
				// 	s = xwidget.RichTextSegmentsFromText(r.jc.Character.Name)
				// case clonesColJumps:
				// 	s = xwidget.RichTextSegmentsFromText(r.jumps())
				// }
				return s
			},
			nil,
		)
	} else {
		a.main = xwidget.MakeDataTable(
			columns,
			&a.rowsFiltered,
			func() fyne.CanvasObject {
				x := xwidget.NewRichText()
				x.Truncation = fyne.TextTruncateClip
				return x
			},
			a.columnSorter,
			a.filterRowsAsync,
			func(_ int, r overviewRow) {
				u.InfoViewer().Show(r.eveEntity())
			},
		)
	}
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterRowsAsync(-1)
	})
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterRowsAsync(-1)
	}, a.u.MainWindow())

	// Signals
	a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		switch arg.Section {
		case app.SectionCharacterAssets, app.SectionCharacterWalletBalance:
			a.update(ctx)
		}
	})
	a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})
	a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})
	a.u.Signals().CharacterChanged.AddListener(func(ctx context.Context, characterID int64) {
		a.update(ctx)
	})
	return a
}

func (a *Overview) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(a.selectTag)
	if a.u.IsMobile() {
		filter.Add(a.sortButton)
	}
	var topBox *fyne.Container
	if a.u.IsMobile() {
		topBox = container.NewVBox(
			a.search,
			container.NewHScroll(filter),
		)
	} else {
		topBox = container.NewBorder(nil, nil, filter, nil, a.search)
	}
	c := container.NewBorder(
		topBox,
		a.footer,
		nil,
		nil,
		a.main,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *Overview) filterRowsAsync(sortCol int) {
	totalRows := len(a.rows)
	rows := slices.Clone(a.rows)
	selectTag := a.selectTag.Selected
	search := strings.ToLower(a.search.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(sortCol)

	go func() {
		if selectTag != "" {
			rows = slices.DeleteFunc(rows, func(r overviewRow) bool {
				return !r.tags.Contains(selectTag)
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r overviewRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)
		tagOptions := slices.Sorted(set.Union(xslices.Map(rows, func(r overviewRow) set.Set[string] {
			return r.tags
		})...).All())

		footer := fmt.Sprintf("Showing %d / %d characters", len(rows), totalRows)

		// add totals
		var assets, wallets, totals optional.Optional[float64]
		for _, r := range rows {
			assets = optional.Sum(assets, r.assets)
			wallets = optional.Sum(wallets, r.wallet)
			totals = optional.Sum(totals, r.total)
		}
		rows = append(rows, overviewRow{
			assets:        assets,
			assetsDisplay: formatISKValue(assets),
			characterID:   0,
			characterName: "Total",
			tags:          set.Set[string]{},
			total:         totals,
			totalDisplay:  formatISKValue(totals),
			wallet:        wallets,
			walletDisplay: formatISKValue(totals),
			searchTarget:  "",
			isTotal:       true,
		})

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectTag.SetOptions(tagOptions)
			a.rowsFiltered = rows
			a.main.Refresh()
		})
	}()
}

func (a *Overview) update(ctx context.Context) {
	rows, err := a.fetchRows(ctx)
	if err != nil {
		slog.Error("Failed to refresh wealth overview UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + a.u.ErrorDisplay(err)
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.rows = rows
		a.filterRowsAsync(-1)
	})
}

func (a *Overview) fetchRows(ctx context.Context) ([]overviewRow, error) {
	characters, err := a.u.Character().CharacterNames(ctx)
	if err != nil {
		return nil, err
	}
	oo, err := a.u.Character().ListWealthValues(ctx)
	if err != nil {
		return nil, err
	}
	var rows []overviewRow
	for _, o := range oo {
		tags, err := a.u.Character().ListTagsForCharacter(ctx, o.CharacterID)
		if err != nil {
			return rows, err
		}
		rows = append(rows, overviewRow{
			assets:        o.Assets,
			assetsDisplay: formatISKValue(o.Assets),
			characterID:   o.CharacterID,
			characterName: characters[o.CharacterID],
			searchTarget:  strings.ToLower(characters[o.CharacterID]),
			tags:          tags,
			total:         o.Total,
			totalDisplay:  formatISKValue(o.Total),
			wallet:        o.Wallet,
			walletDisplay: formatISKValue(o.Wallet),
		})
	}
	return rows, nil
}

func formatISKValue(v optional.Optional[float64]) string {
	return v.StringFunc("?", func(v float64) string {
		return humanize.FormatFloat(ui.FloatFormat, v)
	})
}
