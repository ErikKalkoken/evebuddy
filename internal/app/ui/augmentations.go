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
	characterID            int32
	characterName          string
	implantCount           int
	implantTypeDescription string
	implantTypeID          int32
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

	collapseBranches *ttwidget.Button
	selectImplants   *kxwidget.FilterChipSelect
	selectTag        *kxwidget.FilterChipSelect
	top              *widget.Label
	treeData         iwidget.TreeData[characterAugmentationNode]
	tree             *iwidget.Tree[characterAugmentationNode]
	u                *baseUI
}

func newAugmentations(u *baseUI) *augmentations {
	top := widget.NewLabel("")
	top.Wrapping = fyne.TextWrapWord
	a := &augmentations{
		top: top,
		u:   u,
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

	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		if arg.section == app.SectionCharacterImplants {
			a.update()
		}
	})
	a.u.characterAdded.AddListener(func(_ context.Context, _ *app.Character) {
		a.update()
	})
	a.u.characterRemoved.AddListener(func(_ context.Context, _ *app.EntityShort[int32]) {
		a.update()
	})
	a.u.tagsChanged.AddListener(func(ctx context.Context, s struct{}) {
		a.update()
	})
	return a
}

func (a *augmentations) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHScroll(container.NewHBox(a.selectImplants, a.selectTag, a.collapseBranches))
	c := container.NewBorder(container.NewVBox(a.top, filter), nil, nil, nil, a.tree)
	return widget.NewSimpleRenderer(c)
}

func (a *augmentations) makeTree() *iwidget.Tree[characterAugmentationNode] {
	t := iwidget.NewTree(
		func(branch bool) fyne.CanvasObject {
			iconMain := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
			main := ttwidget.NewRichText()
			info := iwidget.NewTappableIcon(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil)
			return container.NewBorder(
				nil,
				nil,
				iconMain,
				info,
				main,
			)
		},
		func(n *characterAugmentationNode, b bool, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			main := border[0].(*ttwidget.RichText)
			main.Truncation = fyne.TextTruncateEllipsis
			iconMain := border[1].(*canvas.Image)
			info := border[2].(*iwidget.TappableIcon)
			if n.IsTop() {
				go a.u.setCharacterAvatar(n.characterID, func(r fyne.Resource) {
					fyne.Do(func() {
						iconMain.Resource = r
						iconMain.Refresh()
					})
				})
				var implants string
				if n.implantCount > 0 {
					implants = fmt.Sprintf("     %d implants", n.implantCount)
				}
				main.Segments = slices.Concat(
					iwidget.RichTextSegmentsFromText(n.characterName, widget.RichTextStyle{
						Inline: true,
					}),
					iwidget.RichTextSegmentsFromText(implants, widget.RichTextStyle{
						TextStyle: fyne.TextStyle{Italic: true},
					}),
				)
				main.Refresh()
				main.SetToolTip("")
				info.SetToolTip("Show character")
				info.OnTapped = func() {
					a.u.ShowInfoWindow(app.EveEntityCharacter, n.characterID)
				}
			} else {
				a.u.eis.InventoryTypeIconAsync(n.implantTypeID, app.IconPixelSize, func(r fyne.Resource) {
					iconMain.Resource = r
					iconMain.Refresh()
				})
				main.Segments = iwidget.RichTextSegmentsFromText(n.implantTypeName)
				main.Refresh()
				main.SetToolTip(n.implantTypeDescription)
				info.SetToolTip("Show implant")
				info.OnTapped = func() {
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
	if a.treeData.IsEmpty() {
		a.tree.Set(a.treeData)
		return
	}

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

		fyne.Do(func() {
			a.selectTag.SetOptions(tagOptions)
			a.tree.Set(td)
		})
	}()
}

func (a *augmentations) update() {
	td, err := a.updateTreeData()
	if err != nil {
		slog.Error("Failed to refresh augmentations UI", "err", err)
		fyne.Do(func() {
			a.top.Text = "ERROR: " + a.u.humanizeError(err)
			a.top.Importance = widget.DangerImportance
			a.top.Refresh()
			a.top.Show()
		})
		return
	}
	fyne.Do(func() {
		a.treeData = td
		a.filterTree()
		a.top.Hide()
	})
}

func (a *augmentations) updateTreeData() (iwidget.TreeData[characterAugmentationNode], error) {
	var td iwidget.TreeData[characterAugmentationNode]
	ctx := context.Background()
	characters, err := a.u.cs.ListCharactersShort(ctx)
	if err != nil {
		return td, err
	}
	characterImplants := make(map[int32][]*app.CharacterImplant)
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
