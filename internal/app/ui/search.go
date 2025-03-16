package ui

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/character"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/infowindow"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xiter"
)

type resultNode struct {
	category character.SearchCategory
	count    int
	ee       *app.EveEntity
}

func (sn resultNode) isCategory() bool {
	return sn.ee == nil
}

func (sn resultNode) UID() widget.TreeNodeID {
	if sn.isCategory() {
		return "C_" + string(sn.category)
	}
	return fmt.Sprintf("EE_%d", sn.ee.ID)
}

func (sn resultNode) String() string {
	if sn.isCategory() {
		return fmt.Sprintf("%s (%d)", sn.category.String(), sn.count)
	}
	return sn.ee.Name
}

type SearchArea struct {
	Content fyne.CanvasObject

	categories    *widget.CheckGroup
	entry         *widget.Entry
	indicator     *widget.ProgressBarInfinite
	resultCount   *widget.Label
	results       *iwidget.Tree[resultNode]
	searchOptions *widget.Accordion
	strict        *kxwidget.Switch
	u             *BaseUI
	w             fyne.Window
}

func NewSearchArea(u *BaseUI) *SearchArea {
	a := &SearchArea{
		entry:       widget.NewEntry(),
		indicator:   widget.NewProgressBarInfinite(),
		resultCount: widget.NewLabel(""),
		u:           u,
		w:           u.Window,
	}

	options := makeOptions()
	a.categories = widget.NewCheckGroup(options, nil)
	a.categories.Horizontal = true
	a.categories.Selected = options

	a.strict = kxwidget.NewSwitch(nil)
	a.resultCount = widget.NewLabel("")
	a.resultCount.Hide()

	a.results = a.makeResults()
	a.entry.ActionItem = iwidget.NewIconButton(theme.CancelIcon(), func() {
		a.Reset()
	})
	a.entry.PlaceHolder = "Search New Eden"
	a.entry.OnSubmitted = func(s string) {
		go a.doSearch(s)
	}
	a.indicator.Hide()

	a.searchOptions = widget.NewAccordion(
		widget.NewAccordionItem(
			"Search options",
			container.NewVBox(
				container.NewHScroll(a.categories),
				container.NewHBox(a.strict, widget.NewLabel("strict search")),
			),
		),
	)
	c := container.NewBorder(
		container.NewVBox(
			a.entry,
			a.searchOptions,
			widget.NewSeparator(),
			a.indicator,
			a.resultCount,
		),
		nil,
		nil,
		nil,
		a.results,
	)
	a.Content = c
	return a
}

func (a *SearchArea) Focus() {
	a.w.Canvas().Focus(a.entry)
}

func (a *SearchArea) Reset() {
	a.entry.SetText("")
	a.clearResults()
	a.categories.SetSelected(makeOptions())
	a.strict.SetState(false)
	a.searchOptions.CloseAll()
}

func (a *SearchArea) SetWindow(w fyne.Window) {
	a.w = w
}

func (a *SearchArea) makeResults() *iwidget.Tree[resultNode] {
	supportedCategories := infowindow.SupportedEveEntities()
	t := iwidget.NewTree(
		func(b bool) fyne.CanvasObject {
			name := widget.NewLabel("Template")
			image := container.NewPadded(iwidget.NewImageFromResource(icon.Questionmark32Png, fyne.NewSquareSize(app.IconUnitSize)))
			info := iwidget.NewIconButton(theme.InfoIcon(), nil)
			return container.NewBorder(
				nil,
				nil,
				image,
				info,
				name,
			)
		},
		func(n resultNode, b bool, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			border[0].(*widget.Label).SetText(n.String())
			image := border[1].(*fyne.Container).Objects[0].(*canvas.Image)
			info := border[2].(*iwidget.IconButton)
			if n.isCategory() {
				info.Hide()
				image.Hide()
				return
			}
			if imageCategory := n.ee.Category.ToEveImage(); imageCategory != "" {
				go func() {
					image.Show()
					ctx := context.Background()
					res, err := func() (fyne.Resource, error) {
						switch n.ee.Category {
						case app.EveEntityInventoryType:
							et, err := a.u.EveUniverseService.GetOrCreateEveTypeESI(ctx, n.ee.ID)
							if err != nil {
								return nil, err
							}
							switch et.Group.Category.ID {
							case app.EveCategorySKINs:
								return a.u.EveImageService.InventoryTypeSKIN(et.ID, app.IconPixelSize)
							case app.EveCategoryBlueprint:
								return a.u.EveImageService.InventoryTypeBPO(et.ID, app.IconPixelSize)
							default:
								return a.u.EveImageService.InventoryTypeIcon(et.ID, app.IconPixelSize)
							}
						default:
							return a.u.EveImageService.EntityIcon(n.ee.ID, imageCategory, app.IconPixelSize)
						}
					}()
					if err != nil {
						res = theme.BrokenImageIcon()
						slog.Error("failed to load image", "error", err)
					}
					image.Resource = res
					image.Refresh()
				}()
			} else {
				image.Hide()
			}
			if supportedCategories.Contains(n.ee.Category) {
				info.OnTapped = func() {
					iw := infowindow.New(a.u.CurrentCharacterID, a.u.CharacterService, a.u.EveUniverseService, a.u.EveImageService, a.w)
					iw.ShowEveEntity(n.ee)
				}
				info.Show()
			} else {
				info.Hide()
			}
		},
	)
	t.OnSelected = func(n resultNode) {
		defer t.UnselectAll()
		if n.isCategory() {
			t.ToggleBranch(n)
		}
	}
	return t
}

func (a *SearchArea) clearResults() {
	a.results.Clear()
	a.resultCount.Hide()
}

func (a *SearchArea) doSearch(search string) {
	a.clearResults()
	if search == "" {
		return
	}
	a.indicator.Show()
	a.indicator.Start()
	defer func() {
		a.indicator.Stop()
		a.indicator.Hide()
	}()
	categories := slices.Collect(xiter.MapSlice(a.categories.Selected, func(o string) character.SearchCategory {
		return option2searchCategory(o)
	}))
	results, total, err := a.u.CharacterService.SearchESI(
		context.Background(),
		a.u.CurrentCharacterID(),
		search,
		categories,
		a.strict.On,
	)
	if err != nil {
		d2 := iwidget.NewErrorDialog("Search failed", err, a.u.Window)
		d2.Show()
		return
	}
	if total == 500 {
		a.resultCount.Importance = widget.WarningImportance
		a.resultCount.Wrapping = fyne.TextWrapWord
		a.resultCount.SetText(fmt.Sprintf(
			"Search for \"%s\" exceeded the server limit of 500 results "+
				"and may not contain the items you are looking for.",
			search,
		))
	} else {
		a.resultCount.Importance = widget.MediumImportance
		a.resultCount.SetText(fmt.Sprintf("%d Results", total))
	}
	a.resultCount.Show()
	if total == 0 {
		return
	}
	t := iwidget.NewTreeData[resultNode]()
	var categoriesFound int
	for _, c := range categories {
		_, ok := results[c]
		if !ok {
			continue
		}
		categoriesFound++
		n := resultNode{category: c, count: len(results[c])}
		parentUID, err := t.Add(iwidget.RootUID, n)
		if err != nil {
			panic(err)
		}
		for _, o := range results[c] {
			n := resultNode{ee: o}
			t.Add(parentUID, n)
		}
	}
	a.results.Set(t)
	if categoriesFound == 1 {
		a.results.OpenAllBranches()
	}
}

var searchCategory2optionMap = map[character.SearchCategory]string{
	character.SearchAgent:         "Agents",
	character.SearchAlliance:      "Alliances",
	character.SearchCharacter:     "Characters",
	character.SearchCorporation:   "Corporations",
	character.SearchFaction:       "Factions",
	character.SearchInventoryType: "Types",
	character.SearchSolarSystem:   "Systems",
	character.SearchStation:       "Stations",
}

func searchCategory2option(c character.SearchCategory) string {
	return searchCategory2optionMap[c]
}

func option2searchCategory(o string) character.SearchCategory {
	for k, v := range searchCategory2optionMap {
		if v == o {
			return k
		}
	}
	panic("not found")
}

func makeOptions() []string {
	options := slices.Collect(xiter.MapSlice(character.SearchCategories(), func(c character.SearchCategory) string {
		return searchCategory2option(c)
	}))
	slices.Sort(options)
	return options
}
