package ui

import (
	"context"
	"fmt"
	"log/slog"
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

// OverviewArea is the UI area that shows an overview of all the user's characters.
type OverviewArea struct {
	Content fyne.CanvasObject

	rows []overviewCharacter
	body fyne.CanvasObject
	top  *widget.Label
	u    *BaseUI
}

func NewOverviewArea(u *BaseUI) *OverviewArea {
	a := OverviewArea{
		rows: make([]overviewCharacter, 0),
		top:  makeTopLabel(),
		u:    u,
	}
	top := container.NewVBox(a.top, widget.NewSeparator())
	headers := []headerDef{
		{"Name", 250},
		{"Corporation", 250},
		{"Alliance", 250},
		{"Security", 50},
		{"Unread", 100},
		{"Wallet", 100},
		{"Assets", 100},
		{"Last Login", 100},
		{"Home", 250},
		{"Age", 100},
	}
	makeDataLabel := func(col int, c overviewCharacter) (string, fyne.TextAlign, widget.Importance) {
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
	showCharacterInfo := func(oc overviewCharacter) {
		u.ShowCharacterInfoWindow(oc.id)
	}
	if a.u.IsDesktop() {
		a.body = makeDataTableForDesktop(headers, &a.rows, makeDataLabel, showCharacterInfo)
	} else {
		a.body = makeDataTableForMobile(headers, &a.rows, makeDataLabel, showCharacterInfo)
	}
	a.Content = container.NewBorder(top, nil, nil, nil, a.body)
	return &a
}

func (a *OverviewArea) Refresh() {
	t, i, err := func() (string, widget.Importance, error) {
		totals, err := a.updateCharacters()
		if err != nil {
			return "", 0, err
		}
		if len(a.rows) == 0 {
			return "No characters", widget.LowImportance, nil
		}
		walletText := ihumanize.OptionalFloat(totals.wallet, 1, "?")
		assetsText := ihumanize.OptionalFloat(totals.assets, 1, "?")
		unreadText := ihumanize.Optional(totals.unread, "?")
		s := fmt.Sprintf(
			"%d characters • %s ISK wallet • %s ISK assets • %s unread",
			len(a.rows),
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
	ctx := context.Background()
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
	a.rows = cc
	return totals, nil
}
