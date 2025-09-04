package ui

import (
	"context"
	"fmt"
	"image/color"
	"log/slog"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/eveicon"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
)

type jumpCloneNode struct {
	implantCount           int
	implantTypeID          int32
	implantTypeName        string
	implantTypeDescription string
	isUnknown              bool
	jumpCloneID            int32
	jumpCloneName          string
	locationID             int64
	locationName           string
	systemSecurityValue    float32
	systemSecurityType     app.SolarSystemSecurityType
}

func (n jumpCloneNode) isTop() bool {
	return n.implantTypeID == 0
}

func (n jumpCloneNode) UID() widget.TreeNodeID {
	if n.jumpCloneID == 0 {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%d", n.jumpCloneID, n.implantTypeID)
}

type characterJumpClones struct {
	widget.BaseWidget

	character *app.Character
	top       *iwidget.RichText
	tree      *iwidget.Tree[jumpCloneNode]
	u         *baseUI
}

func newCharacterJumpClones(u *baseUI) *characterJumpClones {
	top := iwidget.NewRichText()
	top.Wrapping = fyne.TextWrapWord
	a := &characterJumpClones{
		top: top,
		u:   u,
	}
	a.ExtendBaseWidget(a)
	a.tree = a.makeTree()

	a.u.characterExchanged.AddListener(func(_ context.Context, c *app.Character) {
		a.character = c
	})
	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character) != arg.characterID {
			return
		}
		if arg.section == app.SectionCharacterJumpClones {
			a.update()
		}
	})
	return a
}

func (a *characterJumpClones) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(a.top, nil, nil, nil, a.tree)
	return widget.NewSimpleRenderer(c)
}

func (a *characterJumpClones) makeTree() *iwidget.Tree[jumpCloneNode] {
	t := iwidget.NewTree(
		func(branch bool) fyne.CanvasObject {
			iconMain := iwidget.NewImageFromResource(icons.BlankSvg, fyne.NewSquareSize(app.IconUnitSize))
			main := ttwidget.NewRichText()
			main.Truncation = fyne.TextTruncateEllipsis
			iconInfo := iwidget.NewTappableIcon(theme.NewThemedResource(icons.InformationSlabCircleSvg), nil)
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
			main := border[0].(*ttwidget.RichText)
			hbox := border[1].(*fyne.Container).Objects
			iconMain := hbox[0].(*canvas.Image)
			spacer := hbox[1].(*fyne.Container).Objects[0]
			prefix := hbox[1].(*fyne.Container).Objects[1].(*widget.Label)
			iconInfo := border[2].(*iwidget.TappableIcon)
			if n.isTop() {
				iconMain.Resource = eveicon.FromName(eveicon.CloningCenter)
				iconMain.Refresh()
				if !n.isUnknown {
					prefix.Text = fmt.Sprintf("%.1f", n.systemSecurityValue)
					prefix.Importance = n.systemSecurityType.ToImportance()
					iconInfo.OnTapped = func() {
						a.u.ShowLocationInfoWindow(n.locationID)
					}
					iconInfo.SetToolTip("Show location")
					iconInfo.Show()
				} else {
					prefix.Text = "?"
					prefix.Importance = widget.LowImportance
					iconInfo.Hide()
				}
				var implants string
				if n.implantCount > 0 {
					implants = fmt.Sprintf("     %d implants", n.implantCount)
				}
				main.Segments = slices.Concat(
					iwidget.RichTextSegmentsFromText(n.locationName, widget.RichTextStyle{
						Inline: true,
					}),
					iwidget.RichTextSegmentsFromText(implants, widget.RichTextStyle{
						TextStyle: fyne.TextStyle{Italic: true},
					}),
				)
				main.Refresh()
				main.SetToolTip("")
				prefix.Show()
				spacer.Show()
			} else {
				iwidget.RefreshImageAsync(iconMain, func() (fyne.Resource, error) {
					return a.u.eis.InventoryTypeIcon(n.implantTypeID, app.IconPixelSize)
				})
				main.Segments = iwidget.RichTextSegmentsFromText(n.implantTypeName)
				main.Refresh()
				main.SetToolTip(n.implantTypeDescription)
				prefix.Hide()
				spacer.Hide()
				iconInfo.OnTapped = func() {
					a.u.ShowTypeInfoWindow(n.implantTypeID)
				}
				iconInfo.SetToolTip("Show implant")
				iconInfo.Show()
			}
		},
	)
	t.OnSelectedNode = func(n jumpCloneNode) {
		defer t.UnselectAll()
		if n.isTop() {
			t.ToggleBranch(n.UID())
		}
	}
	return t
}

func (a *characterJumpClones) update() {
	td, err := a.updateTreeData()
	if err != nil {
		slog.Error("Failed to refresh jump clones UI", "err", err)
		fyne.Do(func() {
			a.top.Set(iwidget.RichTextSegmentsFromText("ERROR: "+a.u.humanizeError(err), widget.RichTextStyle{
				ColorName: theme.ColorNameError,
			}))
		})
	} else {
		n, _ := td.ChildrenCount(iwidget.TreeRootID)
		a.refreshTop(n)
		fyne.Do(func() {
			a.tree.Set(td)
		})
	}
}

func (a *characterJumpClones) updateTreeData() (iwidget.TreeData[jumpCloneNode], error) {
	var tree iwidget.TreeData[jumpCloneNode]
	characterID := characterIDOrZero(a.character)
	if characterID == 0 {
		return tree, nil
	}
	ctx := context.Background()
	clones, err := a.u.cs.ListJumpClones(ctx, characterID)
	if err != nil {
		return tree, err
	}
	for _, c := range clones {
		n := jumpCloneNode{
			implantCount:  len(c.Implants),
			jumpCloneID:   c.CloneID,
			jumpCloneName: c.Name,
			locationID:    c.Location.ID,
		}
		// TODO: Refactor to use same location method for all unknown location cases
		if c.Location != nil && !c.Location.Name.IsEmpty() && !c.Location.SecurityStatus.IsEmpty() {
			n.locationName = c.Location.Name.ValueOrZero()
			n.systemSecurityValue = c.Location.SecurityStatus.MustValue()
			n.systemSecurityType = app.NewSolarSystemSecurityTypeFromValue(n.systemSecurityValue)
		}
		if n.locationName == "" {
			n.locationName = fmt.Sprintf("Unknown location #%d", c.Location.ID)
			n.isUnknown = true
		}
		uid := tree.MustAdd(iwidget.TreeRootID, n)
		for _, i := range c.Implants {
			n := jumpCloneNode{
				implantTypeDescription: i.EveType.DescriptionPlain(),
				implantTypeID:          i.EveType.ID,
				implantTypeName:        i.EveType.Name,
				jumpCloneID:            c.CloneID,
			}
			tree.MustAdd(uid, n)
		}
	}
	return tree, err
}

func (a *characterJumpClones) refreshTop(cloneCount int) {
	segs := a.makeTopText(cloneCount, a.character, a.u.services())
	fyne.Do(func() {
		a.top.Set(segs)
	})
}

func (*characterJumpClones) makeTopText(cloneCount int, character *app.Character, s services) []widget.RichTextSegment {
	defaultStyle := widget.RichTextStyle{
		ColorName: theme.ColorNameForeground,
	}
	ts := &widget.TextSegment{
		Text:  "",
		Style: defaultStyle,
	}
	if character == nil {
		ts.Text = "No character"
		ts.Style.ColorName = theme.ColorNameDisabled
		return []widget.RichTextSegment{ts}
	}
	hasData := s.scs.HasCharacterSection(character.ID, app.SectionCharacterJumpClones)
	if !hasData {
		ts.Text = "Waiting for character data to be loaded..."
		ts.Style.ColorName = theme.ColorNameWarning
		return []widget.RichTextSegment{ts}
	}
	var nextJumpColor fyne.ThemeColorName
	var nextJump, lastJump string
	if character.NextCloneJump.IsEmpty() {
		nextJump = "?"
		nextJumpColor = theme.ColorNameForeground
	} else if character.NextCloneJump.MustValue().IsZero() {
		nextJump = "NOW"
		nextJumpColor = theme.ColorNameSuccess
	} else {
		nextJump = ihumanize.Duration(time.Until(character.NextCloneJump.MustValue()))
		nextJumpColor = theme.ColorNameError
	}
	if x := character.LastCloneJumpAt.ValueOrZero(); x.IsZero() {
		lastJump = "?"
	} else {
		lastJump = humanize.Time(x)
	}
	defaultStyleInline := defaultStyle
	defaultStyleInline.Inline = true
	segs := []widget.RichTextSegment{
		&widget.TextSegment{
			Text:  fmt.Sprintf("%d clones • Next available jump: ", cloneCount),
			Style: defaultStyleInline,
		},
		&widget.TextSegment{
			Text: nextJump,
			Style: widget.RichTextStyle{
				ColorName: nextJumpColor,
				Inline:    true,
			},
		},
		&widget.TextSegment{
			Text:  fmt.Sprintf(" • Last jump: %s", lastJump),
			Style: defaultStyle,
		},
	}
	return segs
}

func (a *characterJumpClones) startUpdateTicker() {
	ticker := time.NewTicker(time.Second * 15)
	go func() {
		for {
			<-ticker.C
			var n int
			fyne.DoAndWait(func() {
				n, _ = a.tree.Data().ChildrenCount(iwidget.TreeRootID)
			})
			a.refreshTop(n)
		}
	}()
}
