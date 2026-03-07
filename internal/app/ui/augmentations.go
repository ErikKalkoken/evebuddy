package ui

import (
	"cmp"
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
	"github.com/ErikKalkoken/go-set"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/uiservices"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type characterAugmentationNode struct {
	characterID            int64
	characterName          string
	implantCount           int
	implantTypeDescription string
	implantTypeID          int64
	implantTypeName        string
	tags                   set.Set[string]
}

func (n characterAugmentationNode) IsTop() bool {
	return n.implantTypeID == 0
}

const (
	augmentationsImplantsNone = "No implants"
	augmentationsImplantsSome = "Has implants"
)

type augmentations struct {
	widget.BaseWidget

	footer           *widget.Label
	collapseBranches *ttwidget.Button
	selectImplants   *kxwidget.FilterChipSelect
	selectTag        *kxwidget.FilterChipSelect
	tree             *xwidget.Tree[characterAugmentationNode]
	treeData         xwidget.TreeData[characterAugmentationNode]
	u         uiservices.UIServices
}

func newAugmentations(u         uiservices.UIServices) *augmentations {
	a := &augmentations{
		footer: newLabelWithTruncation(),
		u:      u,
	}
	a.ExtendBaseWidget(a)
	a.tree = a.makeTree()
	a.selectImplants = kxwidget.NewFilterChipSelect("Implants", []string{
		augmentationsImplantsNone,
		augmentationsImplantsSome,
	}, func(_ string) {
		a.filterTreeAsync()
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterTreeAsync()
	})
	a.collapseBranches = ttwidget.NewButtonWithIcon("", theme.NewThemedResource(icons.CollapseAllSvg), func() {
		a.tree.CloseAllBranches()
	})
	a.collapseBranches.SetToolTip("Collapse branches")

	a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if arg.Section == app.SectionCharacterImplants {
			a.update(ctx)
		}
	})
	a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})
	a.u.Signals().TagsChanged.AddListener(func(ctx context.Context, s struct{}) {
		a.update(ctx)
	})
	return a
}

func (a *augmentations) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHBox(
		a.selectImplants,
		a.selectTag,
		a.collapseBranches,
	)
	c := container.NewBorder(
		container.NewHScroll(filter),
		a.footer,
		nil,
		nil,
		a.tree,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *augmentations) makeTree() *xwidget.Tree[characterAugmentationNode] {
	t := xwidget.NewTree(
		func(_ bool) fyne.CanvasObject {
			return newAugmentationNodeItem(
				a.u.EVEImage().CharacterPortraitAsync,
				a.u.EVEImage().InventoryTypeIconAsync,
				func(id int64) {
					a.u.InfoWindow().Show(app.EveEntityCharacter, id)
				},
				a.u.InfoWindow().ShowTypeWithCharacter,
			)
		},
		func(n *characterAugmentationNode, _ bool, co fyne.CanvasObject) {
			co.(*augmentationNodeItem).set(n)
		},
	)
	t.OnSelectedNode = func(n *characterAugmentationNode) {
		defer t.UnselectAll()
		if n.IsTop() {
			t.ToggleBranchNode(n)
		}
	}
	return t
}

func (a *augmentations) filterTreeAsync() {
	total := a.treeData.ChildrenCount(nil)
	tag := a.selectTag.Selected
	implants := a.selectImplants.Selected
	td := a.treeData.Clone()

	go func() {
		var del []func(c *characterAugmentationNode) bool // f returns true when c is to be deleted
		if tag != "" {
			del = append(del, func(c *characterAugmentationNode) bool {
				return !c.tags.Contains(tag)
			})
		}
		if implants != "" {
			switch implants {
			case augmentationsImplantsNone:
				del = append(del, func(c *characterAugmentationNode) bool {
					return c.implantCount != 0
				})
			case augmentationsImplantsSome:
				del = append(del, func(c *characterAugmentationNode) bool {
					return c.implantCount == 0
				})
			}
		}

		if len(del) > 0 {
			characters := td.Children(nil)
			for _, c := range characters {
				var toDelete bool
				for _, d := range del {
					toDelete = toDelete || d(c)
				}
				if !toDelete {
					continue
				}
				err := td.Delete(c)
				if err != nil {
					slog.Error("Failed to remove a character from an augmentations tree", "node", c)
				}
			}
		}
		characters := td.Children(nil)
		tagOptions := slices.Sorted(set.Union(xslices.Map(characters, func(n *characterAugmentationNode) set.Set[string] {
			return n.tags
		})...).All())
		footer := fmt.Sprintf("Showing %d / %d characters", td.ChildrenCount(nil), total)

		fyne.Do(func() {
			a.footer.Text = footer
			a.footer.Importance = widget.MediumImportance
			a.footer.Refresh()
			a.selectTag.SetOptions(tagOptions)
			a.tree.Set(td)
		})
	}()
}

func (a *augmentations) update(ctx context.Context) {
	td, err := a.fetchData(ctx)
	if err != nil {
		slog.Error("Failed to refresh augmentations UI", "err", err)
		fyne.Do(func() {
			a.footer.Text = "ERROR: " + app.ErrorDisplay(err)
			a.footer.Importance = widget.DangerImportance
			a.footer.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.treeData = td
		a.filterTreeAsync()
	})
}

func (a *augmentations) fetchData(ctx context.Context) (xwidget.TreeData[characterAugmentationNode], error) {
	var td xwidget.TreeData[characterAugmentationNode]
	characterImplants := make(map[int64][]*app.CharacterImplant)
	implants, err := a.u.Character().ListAllImplants(ctx)
	if err != nil {
		return td, err
	}
	for _, im := range implants {
		characterImplants[im.CharacterID] = append(characterImplants[im.CharacterID], im)
	}
	for k := range characterImplants {
		slices.SortFunc(characterImplants[k], func(a, b *app.CharacterImplant) int {
			return cmp.Compare(a.SlotNum, b.SlotNum)
		})
	}

	characters, err := a.u.Character().CharacterNames(ctx)
	if err != nil {
		return td, err
	}
	for characterID, implants := range characterImplants {
		tags, err := a.u.Character().ListTagsForCharacter(ctx, characterID)
		if err != nil {
			return td, err
		}
		implantCount := len(implants)
		clone := &characterAugmentationNode{
			characterID:   characterID,
			characterName: characters[characterID],
			implantCount:  implantCount,
			tags:          tags,
		}
		err = td.Add(nil, clone, implantCount > 0)
		if err != nil {
			return td, err
		}
		for _, o := range implants {
			implant := &characterAugmentationNode{
				characterID:            characterID,
				implantTypeDescription: o.EveType.DescriptionPlain(),
				implantTypeID:          o.EveType.ID,
				implantTypeName:        o.EveType.Name,
			}
			err := td.Add(clone, implant, false)
			if err != nil {
				return td, err
			}
		}
	}
	return td, nil
}

type augmentationNodeItem struct {
	widget.BaseWidget

	iconInfo              *xwidget.TappableIcon
	iconMain              *canvas.Image
	implants              *widget.Label
	loadCharacterPortrait loadFuncAsync
	loadTypeIcon          loadFuncAsync
	main                  *ttwidget.Label
	showCharacter         func(int64)
	showType              func(int64, int64)
}

func newAugmentationNodeItem(
	loadCharacterPortrait, loadTypeIcon loadFuncAsync,
	showCharacter func(int64),
	showType func(int64, int64),
) *augmentationNodeItem {
	iconMain := xwidget.NewImageFromResource(
		icons.BlankSvg,
		fyne.NewSquareSize(app.IconUnitSize),
	)
	main := ttwidget.NewLabel("")
	main.Truncation = fyne.TextTruncateEllipsis
	implants := widget.NewLabel("")
	iconInfo := xwidget.NewTappableIcon(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil)
	w := &augmentationNodeItem{
		iconInfo:              iconInfo,
		loadTypeIcon:          loadTypeIcon,
		iconMain:              iconMain,
		implants:              implants,
		main:                  main,
		loadCharacterPortrait: loadCharacterPortrait,
		showCharacter:         showCharacter,
		showType:              showType,
	}
	w.ExtendBaseWidget(w)

	return w
}

func (w *augmentationNodeItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		nil,
		nil,
		w.iconMain,
		container.NewHBox(w.implants, w.iconInfo),
		w.main,
	)
	return widget.NewSimpleRenderer(c)
}

func (w *augmentationNodeItem) set(n *characterAugmentationNode) {
	if n.IsTop() {
		w.loadCharacterPortrait(n.characterID, app.IconPixelSize, func(r fyne.Resource) {
			w.iconMain.Resource = r
			w.iconMain.CornerRadius = app.IconUnitSize / 2
			w.iconMain.Refresh()
		})
		if n.implantCount > 0 {
			w.implants.SetText(fmt.Sprintf("%d implants", n.implantCount))
			w.implants.Show()
		} else {
			w.implants.Hide()
		}
		w.main.SetText(n.characterName)
		w.main.Refresh()
		w.main.SetToolTip("")
		w.iconInfo.SetToolTip("Show character")
		w.iconInfo.OnTapped = func() {
			w.showCharacter(n.characterID)
		}
	} else {
		w.loadTypeIcon(n.implantTypeID, app.IconPixelSize, func(r fyne.Resource) {
			w.iconMain.Resource = r
			w.iconMain.CornerRadius = 0
			w.iconMain.Refresh()
		})
		w.main.SetText(n.implantTypeName)
		w.main.SetToolTip(n.implantTypeDescription)
		w.implants.Hide()
		w.iconInfo.SetToolTip("Show implant")
		w.iconInfo.OnTapped = func() {
			w.showType(n.implantTypeID, n.characterID)
		}
	}
}
