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
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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

	bottom           *widget.Label
	collapseBranches *ttwidget.Button
	selectImplants   *kxwidget.FilterChipSelect
	selectTag        *kxwidget.FilterChipSelect
	tree             *iwidget.Tree[characterAugmentationNode]
	treeData         iwidget.TreeData[characterAugmentationNode]
	u                *baseUI
}

func newAugmentations(u *baseUI) *augmentations {
	a := &augmentations{
		bottom: newLabelWithWrap(),
		u:      u,
	}
	a.ExtendBaseWidget(a)
	a.tree = a.makeTree()
	a.selectImplants = kxwidget.NewFilterChipSelect("Implants", []string{
		augmentationsImplantsNone,
		augmentationsImplantsSome,
	}, func(_ string) {
		a.filterTree()
	})
	a.selectTag = kxwidget.NewFilterChipSelect("Tag", []string{}, func(string) {
		a.filterTree()
	})
	a.collapseBranches = ttwidget.NewButtonWithIcon("", theme.NewThemedResource(icons.CollapseAllSvg), func() {
		a.tree.CloseAllBranches()
	})
	a.collapseBranches.SetToolTip("Collapse branches")

	a.u.characterSectionChanged.AddListener(func(ctx context.Context, arg characterSectionUpdated) {
		if arg.section == app.SectionCharacterImplants {
			a.update(ctx)
		}
	})
	a.u.characterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
		a.update(ctx)
	})
	a.u.characterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
		a.update(ctx)
	})
	a.u.tagsChanged.AddListener(func(ctx context.Context, s struct{}) {
		a.update(ctx)
	})
	return a
}

func (a *augmentations) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHScroll(container.NewHBox(a.selectImplants, a.selectTag, a.collapseBranches))
	c := container.NewBorder(container.NewHScroll(filter), a.bottom, nil, nil, a.tree)
	return widget.NewSimpleRenderer(c)
}

func (a *augmentations) makeTree() *iwidget.Tree[characterAugmentationNode] {
	t := iwidget.NewTree(
		func(branch bool) fyne.CanvasObject {
			iconMain := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
			main := ttwidget.NewLabel("Template")
			main.Truncation = fyne.TextTruncateEllipsis
			implants := widget.NewLabel("9 implants")
			iconInfo := iwidget.NewTappableIcon(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil)
			return container.NewBorder(
				nil,
				nil,
				iconMain,
				container.NewHBox(implants, iconInfo),
				main,
			)
		},
		func(n *characterAugmentationNode, b bool, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			main := border[0].(*ttwidget.Label)
			iconMain := border[1].(*canvas.Image)
			hbox := border[2].(*fyne.Container).Objects
			implants := hbox[0].(*widget.Label)
			iconInfo := hbox[1].(*iwidget.TappableIcon)
			if n.IsTop() {
				a.u.eis.CharacterPortraitAsync(n.characterID, app.IconPixelSize, func(r fyne.Resource) {
					iconMain.Resource = r
					iconMain.CornerRadius = app.IconUnitSize / 2
					iconMain.Refresh()
				})
				if n.implantCount > 0 {
					implants.SetText(fmt.Sprintf("%d implants", n.implantCount))
					implants.Show()
				} else {
					implants.Hide()
				}
				main.SetText(n.characterName)
				main.Refresh()
				main.SetToolTip("")
				iconInfo.SetToolTip("Show character")
				iconInfo.OnTapped = func() {
					a.u.ShowInfoWindow(app.EveEntityCharacter, n.characterID)
				}
			} else {
				a.u.eis.InventoryTypeIconAsync(n.implantTypeID, app.IconPixelSize, func(r fyne.Resource) {
					iconMain.Resource = r
					iconMain.CornerRadius = 0
					iconMain.Refresh()
				})
				main.SetText(n.implantTypeName)
				main.SetToolTip(n.implantTypeDescription)
				implants.Hide()
				iconInfo.SetToolTip("Show implant")
				iconInfo.OnTapped = func() {
					a.u.ShowTypeInfoWindowWithCharacter(n.implantTypeID, n.characterID)
				}
			}
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

func (a *augmentations) filterTree() {
	total, _ := a.treeData.ChildrenCount(nil)
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
		var bottom string
		if total > 0 {
			count, _ := td.ChildrenCount(nil)
			bottom = fmt.Sprintf("Showing %d / %d characters", count, total)
		} else {
			bottom = ""
		}
		fyne.Do(func() {
			a.bottom.SetText(bottom)
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
			a.bottom.Text = "ERROR: " + a.u.humanizeError(err)
			a.bottom.Importance = widget.DangerImportance
			a.bottom.Refresh()
		})
		return
	}
	fyne.Do(func() {
		a.treeData = td
		a.filterTree()
	})
}

func (a *augmentations) fetchData(ctx context.Context) (iwidget.TreeData[characterAugmentationNode], error) {
	var td iwidget.TreeData[characterAugmentationNode]
	characters, err := a.u.cs.ListCharactersShort(ctx)
	if err != nil {
		return td, err
	}
	characterImplants := make(map[int64][]*app.CharacterImplant)
	for _, c := range characters {
		characterImplants[c.ID] = make([]*app.CharacterImplant, 0)
	}
	implants, err := a.u.cs.ListAllImplants(ctx)
	if err != nil {
		return td, err
	}
	for _, im := range implants {
		_, ok := characterImplants[im.CharacterID]
		if !ok {
			slog.Warn("Unknown character for implant", "characterID", im.CharacterID)
			continue
		}
		characterImplants[im.CharacterID] = append(characterImplants[im.CharacterID], im)
	}
	for k := range characterImplants {
		slices.SortFunc(characterImplants[k], func(a, b *app.CharacterImplant) int {
			return cmp.Compare(a.SlotNum, b.SlotNum)
		})
	}
	for _, c := range characters {
		tags, err := a.u.cs.ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			return td, err
		}
		implantCount := len(characterImplants[c.ID])
		clone := &characterAugmentationNode{
			characterID:   c.ID,
			characterName: c.Name,
			implantCount:  implantCount,
			tags:          tags,
		}
		err = td.Add(nil, clone, implantCount > 0)
		if err != nil {
			return td, err
		}
		for _, o := range characterImplants[c.ID] {
			implant := &characterAugmentationNode{
				implantTypeDescription: o.EveType.DescriptionPlain(),
				implantTypeID:          o.EveType.ID,
				implantTypeName:        o.EveType.Name,
				characterID:            c.ID,
			}
			err := td.Add(clone, implant, false)
			if err != nil {
				return td, err
			}
		}
	}
	return td, nil
}
