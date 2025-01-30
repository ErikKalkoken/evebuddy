package ui

import (
	"context"
	"fmt"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/widgets"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
)

type jumpCloneNode struct {
	ImplantCount           int
	ImplantTypeID          int32
	ImplantTypeName        string
	ImplantTypeDescription string
	IsUnknown              bool
	JumpCloneID            int32
	JumpCloneName          string
	LocationID             int64
	LocationName           string
	SystemSecurityValue    float32
	SystemSecurityType     app.SolarSystemSecurityType
}

func (n jumpCloneNode) IsRoot() bool {
	return n.ImplantTypeID == 0
}

func (n jumpCloneNode) UID() widget.TreeNodeID {
	if n.JumpCloneID == 0 {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%d", n.JumpCloneID, n.ImplantTypeID)
}

// JumpClonesArea is the UI area that shows the skillqueue
type JumpClonesArea struct {
	Content    *fyne.Container
	top        *widget.Label
	treeData   *fynetree.FyneTree[jumpCloneNode]
	treeWidget *widget.Tree
	u          *BaseUI
}

func (u *BaseUI) NewJumpClonesArea() *JumpClonesArea {
	a := JumpClonesArea{
		top:      makeTopLabel(),
		treeData: fynetree.New[jumpCloneNode](),
		u:        u,
	}
	a.treeWidget = a.makeTree()
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.Content = container.NewBorder(top, nil, nil, nil, a.treeWidget)
	return &a
}

func (a *JumpClonesArea) makeTree() *widget.Tree {
	labelSizeName := theme.SizeNameText
	if a.u.IsMobile() {
		labelSizeName = theme.SizeNameCaptionText
	}
	t := widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return a.treeData.ChildUIDs(uid)
		},
		func(uid widget.TreeNodeID) bool {
			return a.treeData.IsBranch(uid)
		},
		func(branch bool) fyne.CanvasObject {
			icon := canvas.NewImageFromResource(IconCharacterplaceholder32Jpeg)
			icon.FillMode = canvas.ImageFillContain
			icon.SetMinSize(fyne.NewSquareSize(DefaultIconUnitSize))
			main := widgets.NewLabelWithSize("Template", labelSizeName)
			main.Truncation = fyne.TextTruncateEllipsis
			infoIcon := kwidget.NewTappableIcon(theme.InfoIcon(), nil)
			prefix := widgets.NewLabelWithSize("[8]", labelSizeName)
			return container.NewBorder(nil, nil, container.NewHBox(icon, prefix), infoIcon, main)
		},
		func(uid widget.TreeNodeID, b bool, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			main := border[0].(*widgets.Label)
			hbox := border[1].(*fyne.Container).Objects
			mainIcon := hbox[0].(*canvas.Image)
			prefix := hbox[1].(*widgets.Label)
			infoIcon := border[2].(*kwidget.TappableIcon)
			n, ok := a.treeData.Value(uid)
			if !ok {
				return
			}
			if n.IsRoot() {
				mainIcon.Resource = eveicon.GetResourceByName(eveicon.CloningCenter)
				mainIcon.Refresh()
				if !n.IsUnknown {
					infoIcon.OnTapped = func() {
						a.u.ShowLocationInfoWindow(n.LocationID)
					}
					infoIcon.Show()
				} else {
					infoIcon.Hide()
				}
				main.SetText(n.LocationName)
				if !n.IsUnknown {
					prefix.Text = fmt.Sprintf("%.1f", n.SystemSecurityValue)
					prefix.Importance = n.SystemSecurityType.ToImportance()
				} else {
					prefix.Text = "?"
					prefix.Importance = widget.LowImportance
				}
				prefix.Show()
			} else {
				RefreshImageResourceAsync(mainIcon, func() (fyne.Resource, error) {
					return a.u.EveImageService.InventoryTypeIcon(n.ImplantTypeID, DefaultIconPixelSize)
				})
				main.SetText(n.ImplantTypeName)
				infoIcon.Hide()
				prefix.Hide()
			}
		},
	)
	t.OnSelected = func(uid widget.TreeNodeID) {
		defer t.UnselectAll()
		n, ok := a.treeData.Value(uid)
		if !ok {
			return
		}
		if !n.IsRoot() {
			a.u.ShowTypeInfoWindow(n.ImplantTypeID, a.u.CharacterID(), DescriptionTab)
		}
	}
	return t
}

func (a *JumpClonesArea) Redraw() {
	var t string
	var i widget.Importance
	tree, err := a.newTreeData()
	if err != nil {
		slog.Error("Failed to refresh jump clones UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		clonesCount := len(tree.ChildUIDs(""))
		t, i = a.makeTopText(clonesCount)
	}
	a.treeData = tree
	a.top.Text = t
	a.top.Importance = i
	a.top.Refresh()
	a.treeWidget.Refresh()
}

func (a *JumpClonesArea) newTreeData() (*fynetree.FyneTree[jumpCloneNode], error) {
	tree := fynetree.New[jumpCloneNode]()
	if !a.u.HasCharacter() {
		return tree, nil
	}
	ctx := context.Background()
	clones, err := a.u.CharacterService.ListCharacterJumpClones(ctx, a.u.CharacterID())
	if err != nil {
		return tree, err
	}
	for _, c := range clones {
		n := jumpCloneNode{
			ImplantCount:  len(c.Implants),
			JumpCloneID:   c.JumpCloneID,
			JumpCloneName: c.Name,
			LocationID:    c.Location.ID,
		}
		// TODO: Refactor to use same location method for all unknown location cases
		if c.Location != nil {
			loc, err := a.u.EveUniverseService.GetEveLocation(ctx, c.Location.ID)
			if err != nil {
				slog.Error("get location for jump clone", "error", err)
			} else {
				n.LocationName = loc.Name
				n.SystemSecurityValue = float32(loc.SolarSystem.SecurityStatus)
				n.SystemSecurityType = loc.SolarSystem.SecurityType()
			}
		}
		if n.LocationName == "" {
			n.LocationName = fmt.Sprintf("Unknown location #%d", c.Location.ID)
			n.IsUnknown = true
		}
		uid := tree.MustAdd("", n.UID(), n)
		for _, i := range c.Implants {
			n := jumpCloneNode{
				ImplantTypeDescription: i.EveType.DescriptionPlain(),
				ImplantTypeID:          i.EveType.ID,
				ImplantTypeName:        i.EveType.Name,
				JumpCloneID:            c.JumpCloneID,
			}
			tree.MustAdd(uid, n.UID(), n)
		}
	}
	return tree, err
}

func (a *JumpClonesArea) makeTopText(total int) (string, widget.Importance) {
	if !a.u.HasCharacter() {
		return "No character", widget.LowImportance
	}
	hasData := a.u.StatusCacheService.CharacterSectionExists(a.u.CharacterID(), app.SectionJumpClones)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	return fmt.Sprintf("%d clones", total), widget.MediumImportance
}
