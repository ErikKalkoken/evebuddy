package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
)

type overviewCharacter struct {
	alliance      string
	assetValue    optional.Optional[float64]
	birthday      time.Time
	corporation   string
	home          *app.EntityShort[int64]
	id            int32
	lastLoginAt   optional.Optional[time.Time]
	name          string
	security      float64
	unreadCount   optional.Optional[int]
	walletBalance optional.Optional[float64]
}

var overviewHeaders = []headerDef{
	{"Name", 20},
	{"Corporation", 20},
	{"Alliance", 20},
	{"Security", 5},
	{"Unread", 5},
	{"Wallet", 5},
	{"Assets", 5},
	{"Last Login", 10},
	{"Home", 20},
	{"Age", 10},
}

// OverviewArea is the UI area that shows an overview of all the user's characters.
type OverviewArea struct {
	Content fyne.CanvasObject

	characters []overviewCharacter
	body       fyne.CanvasObject
	top        *widget.Label
	u          *BaseUI
}

func (u *BaseUI) NewOverviewArea() *OverviewArea {
	a := OverviewArea{
		characters: make([]overviewCharacter, 0),
		top:        widget.NewLabel(""),
		u:          u,
	}
	a.top.TextStyle.Bold = true
	a.top.Wrapping = fyne.TextWrapWord
	top := container.NewVBox(a.top, widget.NewSeparator())
	if a.u.IsDesktop() {
		a.body = a.makeTable()
	} else {
		a.body = a.makeList()
	}
	a.Content = container.NewBorder(top, nil, nil, nil, a.body)
	return &a
}

func (a *OverviewArea) makeList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.characters)
		},
		func() fyne.CanvasObject {
			return makeListRowObject(overviewHeaders)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			f := co.(*fyne.Container).Objects
			if id >= len(a.characters) || id < 0 {
				return
			}
			c := a.characters[id]
			for col := range len(overviewHeaders) {
				row := f[col*2].(*fyne.Container).Objects[1].(*fyne.Container).Objects
				data := row[1].(*widget.Label)
				data.Text, data.Alignment, data.Importance = a.makeDataLabel(col, c)
				data.Truncation = fyne.TextTruncateEllipsis
				bg := f[col*2].(*fyne.Container).Objects[0]
				if col == 0 {
					bg.Show()
					data.TextStyle.Bold = true
					label := row[0].(*widget.Label)
					label.TextStyle.Bold = true
					label.Refresh()
				} else {
					bg.Hide()
				}
				data.Refresh()
				divider := f[col*2+1]
				if col > 0 && col < len(overviewHeaders)-1 {
					divider.Show()
				} else {
					divider.Hide()
				}
			}
		},
	)
	l.OnSelected = func(id widget.ListItemID) {
		l.UnselectAll()
	}
	return l
}

func (a *OverviewArea) makeTable() *widget.Table {
	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.characters), len(overviewHeaders)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			cell := co.(*widget.Label)
			if tci.Row >= len(a.characters) || tci.Row < 0 {
				return
			}
			c := a.characters[tci.Row]
			cell.Text, cell.Alignment, cell.Importance = a.makeDataLabel(tci.Col, c)
			cell.Truncation = fyne.TextTruncateClip
			cell.Refresh()
		},
	)
	t.ShowHeaderRow = true
	if a.u.IsDesktop() {
		t.StickyColumnCount = 1
	}
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := overviewHeaders[tci.Col]
		label := co.(*widget.Label)
		label.SetText(s.text)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
	}

	for i, h := range overviewHeaders {
		x := widget.NewLabel(strings.Repeat("w", h.maxChars))
		w := x.MinSize().Width
		t.SetColumnWidth(i, w)
	}
	return t
}

func (*OverviewArea) makeDataLabel(col int, c overviewCharacter) (string, fyne.TextAlign, widget.Importance) {
	var align fyne.TextAlign
	var importance widget.Importance
	var text string
	switch col {
	case 0:
		text = c.name
	case 1:
		text = c.corporation
	case 2:
		text = c.alliance
	case 3:
		text = fmt.Sprintf("%.1f", c.security)
		if c.security > 0 {
			importance = widget.SuccessImportance
		} else if c.security < 0 {
			importance = widget.DangerImportance
		}
		align = fyne.TextAlignTrailing
	case 4:
		text = ihumanize.Optional(c.unreadCount, "?")
		align = fyne.TextAlignTrailing
	case 5:
		text = ihumanize.OptionalFloat(c.walletBalance, 1, "?")
		align = fyne.TextAlignTrailing
	case 6:
		text = ihumanize.OptionalFloat(c.assetValue, 1, "?")
		align = fyne.TextAlignTrailing
	case 7:
		text = ihumanize.Optional(c.lastLoginAt, "?")
		align = fyne.TextAlignTrailing
	case 8:
		text = EntityNameOrFallback(c.home, "?")
	case 9:
		text = humanize.RelTime(c.birthday, time.Now(), "", "")
		align = fyne.TextAlignTrailing
	}
	return text, align, importance
}

func (a *OverviewArea) Refresh() {
	t, i, err := func() (string, widget.Importance, error) {
		totals, err := a.updateCharacters()
		if err != nil {
			return "", 0, err
		}
		if len(a.characters) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		walletText := ihumanize.OptionalFloat(totals.wallet, 1, "?")
		assetsText := ihumanize.OptionalFloat(totals.assets, 1, "?")
		unreadText := ihumanize.Optional(totals.unread, "?")
		s := fmt.Sprintf(
			"%d characters • %s ISK wallet • %s ISK assets • %s unread",
			len(a.characters),
			walletText,
			assetsText,
			unreadText,
		)
		return s, widget.MediumImportance, nil
	}()
	if err != nil {
		slog.Error("Failed to refresh overview UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	}
	a.top.Text = t
	a.top.Importance = i
	a.body.Refresh()
}

type overviewTotals struct {
	unread optional.Optional[int]
	wallet optional.Optional[float64]
	assets optional.Optional[float64]
}

func (a *OverviewArea) updateCharacters() (overviewTotals, error) {
	var totals overviewTotals
	var err error
	ctx := context.TODO()
	mycc, err := a.u.CharacterService.ListCharacters(ctx)
	if err != nil {
		return totals, err
	}
	cc := make([]overviewCharacter, len(mycc))
	for i, m := range mycc {
		c := overviewCharacter{
			alliance:      m.EveCharacter.AllianceName(),
			birthday:      m.EveCharacter.Birthday,
			corporation:   m.EveCharacter.Corporation.Name,
			lastLoginAt:   m.LastLoginAt,
			id:            m.ID,
			name:          m.EveCharacter.Name,
			security:      m.EveCharacter.SecurityStatus,
			walletBalance: m.WalletBalance,
		}
		if m.Home != nil {
			c.home = &app.EntityShort[int64]{
				ID:   m.Home.ID,
				Name: m.Home.DisplayName(),
			}
		}
		cc[i] = c
	}
	for i, c := range cc {
		total, unread, err := a.u.CharacterService.GetCharacterMailCounts(ctx, c.id)
		if err != nil {
			return totals, err
		}
		if total > 0 {
			cc[i].unreadCount = optional.New(unread)
		}
	}
	for i, c := range cc {
		v, err := a.u.CharacterService.CharacterAssetTotalValue(ctx, c.id)
		if err != nil {
			return totals, err
		}
		cc[i].assetValue = v
	}
	for _, c := range cc {
		if !c.unreadCount.IsEmpty() {
			totals.unread.Set(totals.unread.ValueOrZero() + c.unreadCount.ValueOrZero())
		}
		if !c.walletBalance.IsEmpty() {
			totals.wallet.Set(totals.wallet.ValueOrZero() + c.walletBalance.ValueOrZero())
		}
		if !c.assetValue.IsEmpty() {
			totals.assets.Set(totals.assets.ValueOrZero() + c.assetValue.ValueOrZero())
		}
	}
	a.characters = cc

	// FIXME
	// var hasUnread bool
	// for _, c := range a.characters {
	// 	if c.unreadCount.ValueOrZero() > 0 {
	// 		hasUnread = true
	// 		break
	// 	}
	// }
	// if hasUnread {
	// 	a.u.showMailIndicator()
	// }
	return totals, nil
}
