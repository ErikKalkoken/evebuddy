package ui

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/ErikKalkoken/go-set"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type loyaltyPointsNode struct {
	characterID     int64
	characterName   string
	corporationID   int64
	corporationName string
	factionID       int64
	factionName     string
	points          int64
	searchTarget    string
	totalPoints     int64
	tags            set.Set[string]
}

func (n loyaltyPointsNode) IsTop() bool {
	return n.characterID == 0
}

type loyaltyPoints struct {
	widget.BaseWidget

	footer           *widget.Label
	collapseBranches *ttwidget.Button
	columnSorter     *iwidget.ColumnSorter[*loyaltyPointsNode]
	data             map[*loyaltyPointsNode][]*loyaltyPointsNode
	searchBox        *widget.Entry
	selectCharacter  *kxwidget.FilterChipSelect
	selectFaction    *kxwidget.FilterChipSelect
	selectTag        *kxwidget.FilterChipSelect
	sortButton       *iwidget.SortButton
	top              *widget.Label
	tree             *iwidget.Tree[loyaltyPointsNode]
	u                *baseUI
}

const (
	loyaltyPointsColCorporation = iota + 1
	loyaltyPointsColPoints
)

func newLoyaltyPoints(u *baseUI) *loyaltyPoints {
	top := widget.NewLabel("")
	top.Wrapping = fyne.TextWrapWord
	columnSorter := iwidget.NewColumnSorter(iwidget.NewDataColumns([]iwidget.DataColumn[*loyaltyPointsNode]{{
		ID:    loyaltyPointsColCorporation,
		Label: "Corporation",
		Sort: func(a, b *loyaltyPointsNode) int {
			return strings.Compare(a.corporationName, b.corporationName)
		},
	}, {
		ID:    loyaltyPointsColPoints,
		Label: "Points",
		Sort: func(a, b *loyaltyPointsNode) int {
			return cmp.Compare(a.totalPoints, b.totalPoints)
		},
	}}),
		characterLoyaltyPointsColCorporation,
		iwidget.SortAsc,
	)
	a := &loyaltyPoints{
		columnSorter: columnSorter,
		footer:       newLabelWithTruncation(),
		top:          top,
		u:            u,
	}
	a.ExtendBaseWidget(a)
	a.tree = a.makeTree()
	a.selectCharacter = kxwidget.NewFilterChipSelect("Character", []string{}, func(_ string) {
		a.filterTreeAsync()
	})
	a.selectFaction = kxwidget.NewFilterChipSelect("Faction", []string{}, func(_ string) {
		a.filterTreeAsync()
	})
	a.collapseBranches = ttwidget.NewButtonWithIcon("", theme.NewThemedResource(icons.CollapseAllSvg), func() {
		a.tree.CloseAllBranches()
	})
	a.collapseBranches.SetToolTip("Collapse branches")
	a.sortButton = a.columnSorter.NewSortButton(func() {
		a.filterTreeAsync()
	}, a.u.window)
	a.searchBox = widget.NewEntry()
	a.searchBox.SetPlaceHolder("Search corporations")
	a.searchBox.ActionItem = kxwidget.NewIconButton(theme.CancelIcon(), func() {
		a.searchBox.SetText("")
	})
	a.searchBox.OnChanged = func(s string) {
		if len(s) == 1 {
			return
		}
		a.filterTreeAsync()
	}
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterTreeAsync()
	})

	// signals
	a.u.signals.CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if arg.Section == app.SectionCharacterLoyaltyPoints {
			a.update(ctx)
		}
	})
	a.u.signals.CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.u.signals.CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})
	a.u.signals.TagsChanged.AddListener(func(ctx context.Context, s struct{}) {
		a.update(ctx)
	})
	return a
}

func (a *loyaltyPoints) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHScroll(container.NewHBox(
		a.selectFaction,
		a.selectCharacter,
		a.selectTag,
		a.sortButton,
	))
	c := container.NewBorder(
		container.NewVBox(a.top, filter, container.NewBorder(nil, nil, nil, a.collapseBranches, a.searchBox)),
		a.footer,
		nil,
		nil,
		a.tree,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *loyaltyPoints) makeTree() *iwidget.Tree[loyaltyPointsNode] {
	t := iwidget.NewTree(
		func(branch bool) fyne.CanvasObject {
			icon1 := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
			icon2 := iwidget.NewTappableIcon(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil)
			name := widget.NewLabel("Template")
			points := widget.NewLabel("99.999.999")
			return container.NewBorder(
				nil,
				nil,
				icon1,
				container.NewHBox(points, icon2),
				name,
			)
		},
		func(n *loyaltyPointsNode, b bool, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			name := border[0].(*widget.Label)
			name.Truncation = fyne.TextTruncateEllipsis
			icon1 := border[1].(*canvas.Image)
			hbox := border[2].(*fyne.Container).Objects
			points := hbox[0].(*widget.Label)
			icon2 := hbox[1].(*iwidget.TappableIcon)
			if n.IsTop() {
				a.u.eis.CorporationLogoAsync(n.corporationID, app.IconPixelSize, func(r fyne.Resource) {
					icon1.Resource = r
					icon1.CornerRadius = 0
					icon1.Refresh()
				})
				name.SetText(n.corporationName)
				points.Text = ihumanize.Comma(n.totalPoints)
				points.TextStyle = fyne.TextStyle{Bold: true}
				icon2.OnTapped = func() {
					a.u.ShowInfoWindow(app.EveEntityCorporation, n.corporationID)
				}
			} else {
				a.u.eis.CharacterPortraitAsync(n.characterID, app.IconPixelSize, func(r fyne.Resource) {
					icon1.Resource = r
					icon1.CornerRadius = app.IconUnitSize / 2
					icon1.Refresh()
				})
				name.SetText(n.characterName)
				points.Text = ihumanize.Comma(n.points)
				points.TextStyle = fyne.TextStyle{}
				icon2.OnTapped = func() {
					a.u.ShowInfoWindow(app.EveEntityCharacter, n.characterID)
				}
			}
			points.Refresh()
		},
	)
	t.OnSelectedNode = func(n *loyaltyPointsNode) {
		defer t.UnselectAll()
		if n.IsTop() {
			t.ToggleBranchNode(n)
		}
	}
	return t
}

func (a *loyaltyPoints) filterTreeAsync() {
	data := maps.Clone(a.data)
	character := a.selectCharacter.Selected
	faction := a.selectFaction.Selected
	tag := a.selectTag.Selected
	search := strings.ToLower(a.searchBox.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort(-1)

	go func() {
		// filter data
		data2 := make(map[*loyaltyPointsNode][]*loyaltyPointsNode)
		for c := range data {
			if faction != "" && faction != c.factionName {
				continue
			}
			if len(search) > 1 && !strings.Contains(c.searchTarget, search) {
				continue
			}

			var characters []*loyaltyPointsNode
			for _, o := range data[c] {
				if character != "" {
					if o.characterName != character {
						continue
					}
				}
				if tag != "" {
					if !o.tags.Contains(tag) {
						continue
					}
				}
				characters = append(characters, o)
			}
			if len(characters) == 0 {
				continue
			}

			c.totalPoints = 0
			for _, character := range characters {
				data2[c] = append(data2[c], character)
				c.totalPoints += character.points
			}
		}

		// sort corporations
		corporations := slices.Collect(maps.Keys(data2))
		a.columnSorter.SortRows(corporations, sortCol, dir, doSort)

		// build tree
		var td iwidget.TreeData[loyaltyPointsNode]
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
			a.footer.Text = bottom
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectCharacter.SetOptions(characterOptions)
			a.selectFaction.SetOptions(factionOptions)
			a.selectTag.SetOptions(tagOptions)
			a.tree.Set(td)
		})
	}()
}

func (a *loyaltyPoints) update(ctx context.Context) {
	data, err := a.fetchData(ctx)
	if err != nil {
		slog.Error("Failed to refresh loyaltyPoints UI", "err", err)
		fyne.Do(func() {
			a.top.Text = "ERROR: " + a.u.HumanizeError(err)
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

func (a *loyaltyPoints) fetchData(ctx context.Context) (map[*loyaltyPointsNode][]*loyaltyPointsNode, error) {
	data := make(map[*loyaltyPointsNode][]*loyaltyPointsNode)

	characterNames, err := a.u.cs.CharacterNames(ctx)
	if err != nil {
		return nil, err
	}

	characterTags := make(map[int64]set.Set[string])
	for id := range characterNames {
		tags, err := a.u.cs.ListTagsForCharacter(ctx, id)
		if err != nil {
			return nil, err
		}
		characterTags[id] = tags
	}

	entries, err := a.u.cs.ListAllLoyaltyPointEntries(ctx)
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
