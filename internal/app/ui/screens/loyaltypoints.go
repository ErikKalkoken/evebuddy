package screens

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// Names of the loyalty points filters, used as labels on desktop and as option names on mobile.
const (
	loyaltyPointsFilterCharacter = "Character"
	loyaltyPointsFilterFaction   = "Faction"
	loyaltyPointsFilterTag       = "Tag"
)

// loyaltyPointsFilter is the selected value of each loyalty points filter. Empty means not filtered.
type loyaltyPointsFilter struct {
	character string
	faction   string
	tag       string
}

// matchCorporation reports whether corporation node c passes the corporation filters.
func (f loyaltyPointsFilter) matchCorporation(c *loyaltyPointsNode) bool {
	return f.faction == "" || c.factionName == f.faction
}

// matchCharacter reports whether character node o passes the character filters.
func (f loyaltyPointsFilter) matchCharacter(o *loyaltyPointsNode) bool {
	switch {
	case f.character != "" && o.characterName != f.character,
		f.tag != "" && !o.tags.Contains(f.tag):
		return false
	}
	return true
}

type loyaltyPointsNode struct {
	characterID     int64
	characterName   string
	corporationID   int64
	corporationName string
	factionID       int64
	factionName     string
	isCorporation   bool
	points          int64
	searchTarget    string
	tags            set.Set[string]
	totalPoints     int64
}

func (n loyaltyPointsNode) UID() widget.TreeNodeID {
	return fmt.Sprintf("%v-%d-%d", n.isCorporation, n.characterID, n.corporationID)
}

type LoyaltyPoints struct {
	widget.BaseWidget

	filterChip       *xwidget.FilterChipCompact // only on mobile
	filterRun        latestRun
	footer           *widget.Label
	collapseBranches *ttwidget.Button
	columnSorter     *xwidget.ColumnSorter[*loyaltyPointsNode]
	data             map[*loyaltyPointsNode][]*loyaltyPointsNode
	searchEntry      *xwidget.SearchEntry
	selectCharacter  *kxwidget.FilterChipSelect // select chips only on desktop
	selectFaction    *kxwidget.FilterChipSelect
	selectTag        *kxwidget.FilterChipSelect
	sortChip         *kxwidget.SortChip
	top              *widget.Label
	tree             *xwidget.Tree[loyaltyPointsNode]
	u                baseUI
}

func NewLoyaltyPoints(u baseUI) *LoyaltyPoints {
	top := widget.NewLabel("")
	top.Wrapping = fyne.TextWrapWord
	columnSorter := xwidget.NewColumnSorter(xwidget.NewDataColumns([]xwidget.DataColumn[*loyaltyPointsNode]{{
		Label: "Corporation",
		Sort: func(a, b *loyaltyPointsNode) int {
			return strings.Compare(a.corporationName, b.corporationName)
		},
	}, {
		Label: "Points",
		Sort: func(a, b *loyaltyPointsNode) int {
			return cmp.Compare(a.totalPoints, b.totalPoints)
		},
	}}),
		"Corporation",
		xwidget.SortAsc,
	)
	a := &LoyaltyPoints{
		columnSorter: columnSorter,
		footer:       ui.NewLabelWithTruncation(""),
		top:          top,
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.tree = a.makeTree()
	if a.u.IsMobile() {
		a.filterChip = xwidget.NewFilterChipCompact(nil, func(map[string]string) {
			a.filterTreeAsync()
		})
	} else {
		makeSelect := func(label string) *kxwidget.FilterChipSelect {
			return kxwidget.NewFilterChipSelect(label, []string{}, func(string) {
				a.filterTreeAsync()
			})
		}
		a.selectCharacter = makeSelect(loyaltyPointsFilterCharacter)
		a.selectFaction = makeSelect(loyaltyPointsFilterFaction)
		a.selectTag = makeSelect(loyaltyPointsFilterTag)
	}
	a.collapseBranches = ttwidget.NewButtonWithIcon("", theme.NewThemedResource(icons.CollapseAllSvg), func() {
		a.tree.CloseAllBranches()
	})
	a.collapseBranches.SetToolTip("Collapse branches")
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterTreeAsync()
	})

	placeholder := "Search corporations"
	if a.u.IsMobile() {
		placeholder = "Search" // shares the row with the chips
	}
	a.searchEntry = xwidget.NewSearchEntry(placeholder, func(s string) {
		if len(s) == 1 {
			return
		}
		a.filterTreeAsync()
	})

	// signals
	a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
		a.update(ctx)
	})

	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if arg.Section == app.SectionCharacterLoyaltyPoints {
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
	return a
}

func (a *LoyaltyPoints) CreateRenderer() fyne.WidgetRenderer {
	var top *fyne.Container
	if a.u.IsMobile() {
		top = container.NewVBox(
			a.top,
			container.NewBorder(nil, nil, nil, container.NewHBox(a.collapseBranches, a.filterChip, a.sortChip), a.searchEntry),
		)
	} else {
		filter := container.NewHScroll(container.NewHBox(
			a.selectFaction,
			a.selectCharacter,
			a.selectTag,
			a.sortChip,
		))
		top = container.NewVBox(a.top, filter, container.NewBorder(nil, nil, nil, a.collapseBranches, a.searchEntry))
	}
	c := container.NewBorder(
		top,
		a.footer,
		nil,
		nil,
		a.tree,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *LoyaltyPoints) makeTree() *xwidget.Tree[loyaltyPointsNode] {
	t := xwidget.NewTree(
		func(_ bool) fyne.CanvasObject {
			return newLoyaltyPointsListItem(
				a.u.EVEImage().EveEntityLogoAsync,
				a.u.InfoViewer().Show,
			)
		},
		func(n *loyaltyPointsNode, _ bool, co fyne.CanvasObject) {
			x := co.(*loyaltyPointsListItem)
			if n.isCorporation {
				o := &app.EveEntity{
					Category: app.EveEntityCorporation,
					ID:       n.corporationID,
					Name:     n.corporationName,
				}
				x.set(o, n.totalPoints, true)
			} else {
				o := &app.EveEntity{
					Category: app.EveEntityCharacter,
					ID:       n.characterID,
					Name:     n.characterName,
				}
				x.set(o, n.points, false)
			}
		},
	)
	t.OnSelectedNode = func(n *loyaltyPointsNode) {
		defer t.UnselectAll()
		if n.isCorporation {
			t.ToggleBranchNode(n)
		}
	}
	return t
}

// currentFilter returns the selected filters: from the compact chip on mobile
// and from the filter chips on desktop.
func (a *LoyaltyPoints) currentFilter() loyaltyPointsFilter {
	if a.filterChip != nil {
		s := a.filterChip.Selected()
		return loyaltyPointsFilter{
			character: s[loyaltyPointsFilterCharacter],
			faction:   s[loyaltyPointsFilterFaction],
			tag:       s[loyaltyPointsFilterTag],
		}
	}
	return loyaltyPointsFilter{
		character: a.selectCharacter.Selected,
		faction:   a.selectFaction.Selected,
		tag:       a.selectTag.Selected,
	}
}

func (a *LoyaltyPoints) filterTreeAsync() {
	isLatest := a.filterRun.start()
	data := maps.Clone(a.data)
	filter := a.currentFilter()
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort("")

	runAsync(func() {
		// filter data
		data2 := make(map[*loyaltyPointsNode][]*loyaltyPointsNode)
		for c := range data {
			if !filter.matchCorporation(c) {
				continue
			}
			if len(search) > 1 && !strings.Contains(c.searchTarget, search) {
				continue
			}

			var characters []*loyaltyPointsNode
			var total int64
			for _, o := range data[c] {
				if filter.matchCharacter(o) {
					characters = append(characters, o)
					total += o.points
				}
			}
			if len(characters) == 0 {
				continue
			}
			c2 := *c // the tree on screen still holds c
			c2.totalPoints = total
			data2[&c2] = characters
		}

		// sort corporations
		corporations := slices.Collect(maps.Keys(data2))
		a.columnSorter.SortRows(corporations, sortCol, dir, doSort)

		// build tree
		td := xwidget.NewTreeData[loyaltyPointsNode]()
		var factionOptions, characterOptions []string
		var tags set.Set[string]
		for _, c := range corporations {
			err := td.Add(nil, c, true)
			if err != nil {
				slog.Error("loyaltypoints: Add corporation", "corporation", c, "error", err)
				continue
			}
			factionOptions = append(factionOptions, c.factionName)
			slices.SortFunc(data2[c], func(a, b *loyaltyPointsNode) int {
				return strings.Compare(a.characterName, b.characterName)
			})
			for _, o := range data2[c] {
				err := td.Add(c, o, false)
				if err != nil {
					slog.Error("loyaltypoints: Add character", "character", o, "error", err)
					continue
				}
				characterOptions = append(characterOptions, o.characterName)
				tags.AddSeq(o.tags.All())
			}
		}
		tagOptions := slices.Collect(tags.All())

		bottom := fmt.Sprintf("Showing %d / %d corporations", len(corporations), len(data))
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.footer.Text = bottom
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			if a.filterChip != nil {
				a.filterChip.SetOptions(
					xwidget.NewFilterOptionMultiChoice(loyaltyPointsFilterFaction, factionOptions),
					xwidget.NewFilterOptionMultiChoice(loyaltyPointsFilterCharacter, characterOptions),
					xwidget.NewFilterOptionMultiChoice(loyaltyPointsFilterTag, tagOptions),
				)
			} else {
				a.selectCharacter.SetOptions(characterOptions)
				a.selectFaction.SetOptions(factionOptions)
				a.selectTag.SetOptions(tagOptions)
			}
			a.tree.Set(td)
		})
	})
}

func (a *LoyaltyPoints) update(ctx context.Context) {
	data, err := a.fetchData(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh loyaltyPoints UI", "err", err)
		fyne.Do(func() {
			a.top.Text = "ERROR: " + a.u.ErrorDisplay(err)
			a.top.Importance = widget.DangerImportance
			a.top.Refresh()
			a.top.Show()
			a.tree.Clear()
			a.tree.CloseAllBranches()
		})
		return
	}
	fyne.Do(func() {
		a.data = data
		a.filterTreeAsync()
		a.top.Hide()
	})
}

func (a *LoyaltyPoints) fetchData(ctx context.Context) (map[*loyaltyPointsNode][]*loyaltyPointsNode, error) {
	data := make(map[*loyaltyPointsNode][]*loyaltyPointsNode)

	characterNames, err := a.u.Character().CharacterNames(ctx)
	if err != nil {
		return nil, err
	}

	characterTags := make(map[int64]set.Set[string])
	for id := range characterNames {
		tags, err := a.u.Character().ListTagsForCharacter(ctx, id)
		if err != nil {
			return nil, err
		}
		characterTags[id] = tags
	}

	entries, err := a.u.Character().ListAllLoyaltyPointEntries(ctx)
	if err != nil {
		return nil, err
	}
	entries = slices.DeleteFunc(entries, func(x *app.CharacterLoyaltyPointEntry) bool {
		return x.LoyaltyPoints == 0
	})

	var corporations []*loyaltyPointsNode
	corporationEntries := make(map[int64][]*app.CharacterLoyaltyPointEntry)
	for _, o := range entries {
		k := o.Corporation.ID
		if corporationEntries[k] == nil {
			c := &loyaltyPointsNode{
				isCorporation:   true,
				corporationID:   o.Corporation.ID,
				corporationName: o.Corporation.Name,
				searchTarget:    strings.ToLower(o.Corporation.Name),
			}
			if f, ok := o.Faction.Value(); ok {
				c.factionID = f.ID
				c.factionName = f.Name
			}
			corporations = append(corporations, c)
		}
		corporationEntries[k] = append(corporationEntries[k], o)
	}

	for _, c := range corporations {
		for _, o := range corporationEntries[c.corporationID] {
			character := &loyaltyPointsNode{
				isCorporation: false,
				corporationID: o.Corporation.ID,
				characterID:   o.CharacterID,
				characterName: characterNames[o.CharacterID],
				points:        o.LoyaltyPoints,
				tags:          characterTags[o.CharacterID],
			}
			data[c] = append(data[c], character)
		}
	}
	return data, nil
}
