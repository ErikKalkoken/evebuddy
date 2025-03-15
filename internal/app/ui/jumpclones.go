package ui

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icon"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	"github.com/ErikKalkoken/evebuddy/internal/fynetree"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
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
	Content  *fyne.Container
	OnReDraw func(clonesCount int)

	top  *widget.RichText
	tree *fynetree.Tree[jumpCloneNode]
	u    *BaseUI
}

func NewJumpClonesArea(u *BaseUI) *JumpClonesArea {
	ntop := widget.NewRichText()
	ntop.Wrapping = fyne.TextWrapWord
	a := JumpClonesArea{
		top: ntop,
		u:   u,
	}
	a.tree = a.makeTree()
	top := container.NewVBox(a.top, widget.NewSeparator())
	a.Content = container.NewBorder(top, nil, nil, nil, a.tree)
	return &a
}

func (a *JumpClonesArea) makeTree() *fynetree.Tree[jumpCloneNode] {
	t := fynetree.NewTree(
		func(branch bool) fyne.CanvasObject {
			iconMain := iwidget.NewImageFromResource(
				icon.Characterplaceholder64Jpeg,
				fyne.NewSquareSize(app.IconUnitSize),
			)
			main := widget.NewLabel("Template")
			main.Truncation = fyne.TextTruncateEllipsis
			iconInfo := kxwidget.NewTappableIcon(theme.InfoIcon(), nil)
			spacer := canvas.NewRectangle(color.Transparent)
			spacer.SetMinSize(fyne.NewSize(40, 10))
			prefix := widget.NewLabel("-9.9")
			return container.NewBorder(
				nil,
				nil,
				container.NewHBox(iconMain, container.NewStack(spacer, prefix)),
				iconInfo,
				main,
			)
		},
		func(n jumpCloneNode, b bool, co fyne.CanvasObject) {
			border := co.(*fyne.Container).Objects
			main := border[0].(*widget.Label)
			hbox := border[1].(*fyne.Container).Objects
			iconMain := hbox[0].(*canvas.Image)
			spacer := hbox[1].(*fyne.Container).Objects[0]
			prefix := hbox[1].(*fyne.Container).Objects[1].(*widget.Label)
			iconInfo := border[2].(*kxwidget.TappableIcon)
			if n.IsRoot() {
				iconMain.Resource = eveicon.GetResourceByName(eveicon.CloningCenter)
				iconMain.Refresh()
				if !n.IsUnknown {
					iconInfo.OnTapped = func() {
						a.u.ShowLocationInfoWindow(n.LocationID)
					}
					iconInfo.Show()
				} else {
					iconInfo.Hide()
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
				spacer.Show()
			} else {
				appwidget.RefreshImageResourceAsync(iconMain, func() (fyne.Resource, error) {
					return a.u.EveImageService.InventoryTypeIcon(n.ImplantTypeID, app.IconPixelSize)
				})
				main.SetText(n.ImplantTypeName)
				iconInfo.OnTapped = func() {
					a.u.ShowTypeInfoWindow(n.ImplantTypeID)
				}
				prefix.Hide()
				spacer.Hide()
			}
		},
	)
	t.OnSelected = func(n jumpCloneNode) {
		defer t.UnselectAll()
	}
	return t
}

func (a *JumpClonesArea) Redraw() {
	tree, err := a.newTreeData()
	if err != nil {
		slog.Error("Failed to refresh jump clones UI", "err", err)
		iwidget.SetRichText(a.top, widget.TextSegment{
			Text: "ERROR: " + ihumanize.Error(err),
			Style: widget.RichTextStyle{
				ColorName: theme.ColorNameError,
				TextStyle: fyne.TextStyle{Bold: true},
			}})

	} else {
		a.RefreshTop()
	}
	a.tree.Set(tree)
	if a.OnReDraw != nil {
		a.OnReDraw(a.ClonesCount())
	}
}

func (a *JumpClonesArea) newTreeData() (*fynetree.TreeData[jumpCloneNode], error) {
	tree := fynetree.NewTreeData[jumpCloneNode]()
	if !a.u.HasCharacter() {
		return tree, nil
	}
	ctx := context.Background()
	clones, err := a.u.CharacterService.ListCharacterJumpClones(ctx, a.u.CurrentCharacterID())
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
		uid := tree.MustAdd(fynetree.RootUID, n)
		for _, i := range c.Implants {
			n := jumpCloneNode{
				ImplantTypeDescription: i.EveType.DescriptionPlain(),
				ImplantTypeID:          i.EveType.ID,
				ImplantTypeName:        i.EveType.Name,
				JumpCloneID:            c.JumpCloneID,
			}
			tree.MustAdd(uid, n)
		}
	}
	return tree, err
}

func (a *JumpClonesArea) RefreshTop() {
	boldTextStyle := fyne.TextStyle{Bold: true}
	defaultStyle := widget.RichTextStyle{
		ColorName: theme.ColorNameForeground,
		TextStyle: boldTextStyle,
	}
	defaultStyleInline := defaultStyle
	defaultStyleInline.Inline = true
	s := widget.TextSegment{
		Text:  "",
		Style: defaultStyle,
	}
	c := a.u.CurrentCharacter()
	if c == nil {
		s.Text = "No character"
		s.Style.ColorName = theme.ColorNameDisabled
		iwidget.SetRichText(a.top, s)
		return
	}
	hasData := a.u.StatusCacheService.CharacterSectionExists(c.ID, app.SectionJumpClones)
	if !hasData {
		s.Text = "Waiting for character data to be loaded..."
		s.Style.ColorName = theme.ColorNameWarning
		iwidget.SetRichText(a.top, s)
	}
	var nextJumpColor fyne.ThemeColorName
	var nextJump, lastJump string
	if c.NextCloneJump.IsEmpty() {
		nextJump = "?"
		nextJumpColor = theme.ColorNameForeground
	} else if c.NextCloneJump.MustValue().IsZero() {
		nextJump = "NOW"
		nextJumpColor = theme.ColorNameSuccess
	} else {
		nextJump = ihumanize.Duration(time.Until(c.NextCloneJump.MustValue()))
		nextJumpColor = theme.ColorNameError
	}
	if x := c.LastCloneJumpAt.ValueOrZero(); x.IsZero() {
		lastJump = "?"
	} else {
		lastJump = humanize.Time(x)
	}
	iwidget.SetRichText(
		a.top,
		widget.TextSegment{
			Text:  fmt.Sprintf("%d clones • Next available jump: ", a.ClonesCount()),
			Style: defaultStyleInline,
		},
		widget.TextSegment{
			Text: nextJump,
			Style: widget.RichTextStyle{
				ColorName: nextJumpColor,
				TextStyle: boldTextStyle,
				Inline:    true,
			},
		},
		widget.TextSegment{
			Text:  fmt.Sprintf(" • Last jump: %s", lastJump),
			Style: defaultStyle,
		},
	)
}

func (a *JumpClonesArea) ClonesCount() int {
	return len(a.tree.Data().ChildUIDs(""))
}

func (a *JumpClonesArea) StartUpdateTicker() {
	ticker := time.NewTicker(time.Second * 15)
	go func() {
		for {
			a.RefreshTop()
			<-ticker.C
		}
	}()
}
