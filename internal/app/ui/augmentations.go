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

type characterImplantsNode struct {
	characterID            int32
	characterName          string
	implantCount           int
	implantTypeDescription string
	implantTypeID          int32
	implantTypeName        string
	tags                   set.Set[string]
}

func (n characterImplantsNode) IsTop() bool {
	return n.implantTypeID == 0
}

func (n characterImplantsNode) UID() widget.TreeNodeID {
	if n.characterID == 0 {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%d", n.characterID, n.implantTypeID)
}

const (
	augmentationsImplantsNone = "No implants"
	augmentationsImplantsSome = "Has implants"
)

type augmentations struct {
	widget.BaseWidget

	collapseAll    *ttwidget.Button
	selectImplants *kxwidget.FilterChipSelect
	selectTag      *kxwidget.FilterChipSelect
	top            *widget.Label
	treeData       iwidget.TreeData[characterImplantsNode]
	tree           *iwidget.Tree[characterImplantsNode]
	u              *baseUI
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
	a.collapseAll = ttwidget.NewButtonWithIcon("", theme.NewThemedResource(icons.ArrowCollapseVerticalSvg), func() {
		a.tree.CloseAllBranches()
	})
	a.collapseAll.SetToolTip("Collapse branches")

	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		if arg.section == app.SectionCharacterImplants {
			a.update()
		}
	})
	return a
}

func (a *augmentations) CreateRenderer() fyne.WidgetRenderer {
	filter := container.NewHScroll(container.NewHBox(a.selectImplants, a.selectTag, a.collapseAll))
	c := container.NewBorder(container.NewVBox(a.top, filter), nil, nil, nil, a.tree)
	return widget.NewSimpleRenderer(c)
}

func (a *augmentations) makeTree() *iwidget.Tree[characterImplantsNode] {
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
		func(n characterImplantsNode, b bool, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			main := border[0].(*ttwidget.RichText)
			main.Truncation = fyne.TextTruncateEllipsis
			iconMain := border[1].(*canvas.Image)
			info := border[2].(*iwidget.TappableIcon)
			if n.IsTop() {
				go a.u.updateCharacterAvatar(n.characterID, func(r fyne.Resource) {
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
				iwidget.RefreshImageAsync(iconMain, func() (fyne.Resource, error) {
					return a.u.eis.InventoryTypeIcon(n.implantTypeID, app.IconPixelSize)
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
	t.OnSelectedNode = func(n characterImplantsNode) {
		defer t.UnselectAll()
		if n.IsTop() {
			t.ToggleBranch(n.UID())
		}
	}
	return t
}

func (a *augmentations) filterTree() {
	if a.treeData.IsEmpty() {
		a.tree.Set(a.treeData)
		return
	}
	var del []func(c characterImplantsNode) bool // f returns true when c is to be deleted
	if x := a.selectTag.Selected; x != "" {
		del = append(del, func(c characterImplantsNode) bool {
			return !c.tags.Contains(x)
		})
	}
	if x := a.selectImplants.Selected; x != "" {
		switch x {
		case augmentationsImplantsNone:
			del = append(del, func(c characterImplantsNode) bool {
				return c.implantCount != 0
			})
		case augmentationsImplantsSome:
			del = append(del, func(c characterImplantsNode) bool {
				return c.implantCount == 0
			})
		}
	}
	td := a.treeData.Clone()
	if len(del) > 0 {
		characters, _ := td.Children(iwidget.TreeRootID)
		for _, c := range characters {
			var toDelete bool
			for _, d := range del {
				toDelete = toDelete || d(c)
			}
			if !toDelete {
				continue
			}
			err := td.Delete(c.UID())
			if err != nil {
				slog.Error("Failed to remove a character from an augmentations tree", "node", c)
			}
		}
	}
	characters, _ := td.Children(iwidget.TreeRootID)
	a.selectTag.SetOptions(slices.Sorted(set.Union(xslices.Map(characters, func(n characterImplantsNode) set.Set[string] {
		return n.tags
	})...).All()))
	a.tree.Set(td)
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
	a.treeData = td
	fyne.Do(func() {
		a.filterTree()
		a.top.Hide()
	})
}

func (a *augmentations) updateTreeData() (iwidget.TreeData[characterImplantsNode], error) {
	var tree iwidget.TreeData[characterImplantsNode]
	ctx := context.Background()
	characters, err := a.u.cs.ListCharactersShort(ctx)
	if err != nil {
		return tree, err
	}
	m := make(map[int32][]*app.CharacterImplant)
	for _, c := range characters {
		m[c.ID] = make([]*app.CharacterImplant, 0)
	}
	oo, err := a.u.cs.ListAllImplants(ctx)
	if err != nil {
		return tree, err
	}
	for _, o := range oo {
		_, ok := m[o.CharacterID]
		if !ok {
			slog.Warn("Unknown character for implant", "characterID", o.CharacterID)
			continue
		}
		m[o.CharacterID] = append(m[o.CharacterID], o)
	}
	for k := range m {
		slices.SortFunc(m[k], func(a, b *app.CharacterImplant) int {
			return cmp.Compare(a.SlotNum, b.SlotNum)
		})
	}
	for _, c := range characters {
		tags, err := a.u.cs.ListTagsForCharacter(ctx, c.ID)
		if err != nil {
			return tree, err
		}
		n := characterImplantsNode{
			characterID:   c.ID,
			characterName: c.Name,
			implantCount:  len(m[c.ID]),
			tags:          tags,
		}
		uid := tree.MustAdd(iwidget.TreeRootID, n)
		for _, o := range m[c.ID] {
			n := characterImplantsNode{
				implantTypeDescription: o.EveType.DescriptionPlain(),
				implantTypeID:          o.EveType.ID,
				implantTypeName:        o.EveType.Name,
				characterID:            c.ID,
			}
			tree.MustAdd(uid, n)
		}
	}
	return tree, nil
}
