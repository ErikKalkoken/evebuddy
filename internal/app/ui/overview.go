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
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/app/humanize"
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

// overviewArea is the UI area that shows an overview of all the user's characters.
type overviewArea struct {
	characters []overviewCharacter
	content    *fyne.Container
	table      *widget.Table
	top        *widget.Label
	u          *UI
}

func (u *UI) newOverviewArea() *overviewArea {
	a := overviewArea{
		characters: make([]overviewCharacter, 0),
		top:        widget.NewLabel(""),
		u:          u,
	}
	a.top.TextStyle.Bold = true

	top := container.NewVBox(a.top, widget.NewSeparator())
	a.table = a.makeTable()
	a.content = container.NewBorder(top, nil, nil, nil, a.table)
	return &a
}

func (a *overviewArea) makeTable() *widget.Table {
	var headers = []struct {
		text     string
		maxChars int
	}{
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

	t := widget.NewTable(
		func() (rows int, cols int) {
			return len(a.characters), len(headers)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			l := co.(*widget.Label)
			if tci.Row >= len(a.characters) || tci.Row < 0 {
				return
			}
			c := a.characters[tci.Row]
			l.Alignment = fyne.TextAlignLeading
			l.Importance = widget.MediumImportance
			var text string
			switch tci.Col {
			case 0:
				text = c.name
			case 1:
				text = c.corporation
			case 2:
				text = c.alliance
			case 3:
				text = fmt.Sprintf("%.1f", c.security)
				if c.security > 0 {
					l.Importance = widget.SuccessImportance
				} else if c.security < 0 {
					l.Importance = widget.DangerImportance
				}
				l.Alignment = fyne.TextAlignTrailing
			case 4:
				text = ihumanize.Optional(c.unreadCount, "?")
				l.Alignment = fyne.TextAlignTrailing
			case 5:
				text = ihumanize.OptionalFloat(c.walletBalance, 1, "?")
				l.Alignment = fyne.TextAlignTrailing
			case 6:
				text = ihumanize.OptionalFloat(c.assetValue, 1, "?")
				l.Alignment = fyne.TextAlignTrailing
			case 7:
				text = ihumanize.Optional(c.lastLoginAt, "?")
			case 8:
				text = entityNameOrFallback(c.home, "?")
			case 9:
				text = humanize.RelTime(c.birthday, time.Now(), "", "")
				l.Alignment = fyne.TextAlignTrailing
			}
			l.Text = text
			l.Truncation = fyne.TextTruncateClip
			l.Refresh()
		},
	)
	t.ShowHeaderRow = true
	t.StickyColumnCount = 1
	t.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("Template")
	}
	t.UpdateHeader = func(tci widget.TableCellID, co fyne.CanvasObject) {
		s := headers[tci.Col]
		label := co.(*widget.Label)
		label.SetText(s.text)
	}
	t.OnSelected = func(tci widget.TableCellID) {
		defer t.UnselectAll()
	}

	for i, h := range headers {
		x := widget.NewLabel(strings.Repeat("w", h.maxChars))
		w := x.MinSize().Width
		t.SetColumnWidth(i, w)
	}
	return t
}

func (a *overviewArea) refresh() {
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
	a.table.Refresh()
}

type overviewTotals struct {
	unread optional.Optional[int]
	wallet optional.Optional[float64]
	assets optional.Optional[float64]
}

func (a *overviewArea) updateCharacters() (overviewTotals, error) {
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
	var hasUnread bool
	for _, c := range a.characters {
		if c.unreadCount.ValueOrZero() > 0 {
			hasUnread = true
			break
		}
	}
	if hasUnread {
		a.u.showMailIndicator()
	}
	return totals, nil
}
