package app_test

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/ErikKalkoken/go-set"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestContractStatusString(t *testing.T) {
	xassert.Equal(t, "cancelled", app.ContractStatusCancelled.String())
}

func TestContractStatusDisplay(t *testing.T) {
	xassert.Equal(t, "Cancelled", app.ContractStatusCancelled.Display())
}

func TestContractCategories(t *testing.T) {
	xassert.Equal(t, []app.ContractCategory{
		app.ContractCategoryOutstanding,
		app.ContractCategoryInProgress,
		app.ContractCategoryRequiresAttention,
		app.ContractCategoryFinished,
		app.ContractCategoryOther,
	}, app.ContractCategories())
}

func TestContractCategoryIsOpen(t *testing.T) {
	cases := []struct {
		category app.ContractCategory
		want     bool
	}{
		{app.ContractCategoryOutstanding, true},
		{app.ContractCategoryInProgress, true},
		{app.ContractCategoryRequiresAttention, true},
		{app.ContractCategoryFinished, false},
		{app.ContractCategoryOther, false},
		{app.ContractCategoryUndefined, false},
	}
	for _, tc := range cases {
		t.Run(tc.category.Display(), func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.category.IsOpen())
		})
	}
}

func TestContractCategoryDisplayAndColor(t *testing.T) {
	cases := []struct {
		category  app.ContractCategory
		wantText  string
		wantColor fyne.ThemeColorName
	}{
		{app.ContractCategoryOutstanding, "Outstanding", theme.ColorNameWarning},
		{app.ContractCategoryInProgress, "In progress", theme.ColorNameForeground},
		{app.ContractCategoryRequiresAttention, "Requires attention", theme.ColorNameError},
		{app.ContractCategoryFinished, "Finished", theme.ColorNameSuccess},
		{app.ContractCategoryOther, "Other", theme.ColorNameForeground},
		{app.ContractCategoryUndefined, "?", theme.ColorNameForeground},
	}
	for _, tc := range cases {
		t.Run(tc.wantText, func(t *testing.T) {
			xassert.Equal(t, tc.wantText, tc.category.Display())
			xassert.Equal(t, tc.wantColor, tc.category.Color())
		})
	}
}

func TestContractStatusPredicates(t *testing.T) {
	cases := []struct {
		status       app.ContractStatus
		wantFinished bool
	}{
		{app.ContractStatusUndefined, false},
		{app.ContractStatusOutstanding, false},
		{app.ContractStatusInProgress, false},
		{app.ContractStatusDeleted, false},
		{app.ContractStatusCancelled, false},
		{app.ContractStatusFinished, true},
		{app.ContractStatusFinishedContractor, true},
		{app.ContractStatusFinishedIssuer, true},
		{app.ContractStatusReversed, true},
		{app.ContractStatusFailed, false},
		{app.ContractStatusRejected, false},
	}
	for _, tc := range cases {
		t.Run(tc.status.String(), func(t *testing.T) {
			xassert.Equal(t, tc.wantFinished, tc.status.IsFinished())
		})
	}
}

func TestContractStatusActive(t *testing.T) {
	xassert.Equal(t, set.Of(app.ContractStatusOutstanding, app.ContractStatusInProgress), app.ContractStatusActive)
}

func TestContractType(t *testing.T) {
	xassert.Equal(t, "auction", app.ContractTypeAuction.String())
}

func TestContractTypeUnknown(t *testing.T) {
	xassert.Equal(t, "?", app.ContractType(99).String())
}

func TestContractTypeHasItems(t *testing.T) {
	cases := map[app.ContractType]bool{
		app.ContractTypeAuction:      true,
		app.ContractTypeCourier:      false,
		app.ContractTypeItemExchange: true,
		app.ContractTypeLoan:         true,
		app.ContractTypeUnknown:      true,
	}
	for typ, want := range cases {
		t.Run(typ.String(), func(t *testing.T) {
			xassert.Equal(t, want, typ.HasItems())
		})
	}
}

func TestContractTypeDisplay(t *testing.T) {
	xassert.Equal(t, "Auction", app.ContractTypeAuction.Display())
}

func TestContractAvailabilityDisplay(t *testing.T) {
	xassert.Equal(t, "Private", app.ContractAvailabilityPrivate.Display())
}

func TestContractAvailabilityStringUnknown(t *testing.T) {
	xassert.Equal(t, "?", app.ContractAvailability(99).String())
}

func TestContractStateAt(t *testing.T) {
	now := time.Now().UTC() // UTC avoids DST shifts in the exact boundary cases
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)
	none := optional.Optional[time.Time]{}
	noDays := optional.Optional[int64]{}
	accepted := optional.New(now.AddDate(0, 0, -4))
	cases := []struct {
		name           string
		status         app.ContractStatus
		dateExpired    time.Time
		dateAccepted   optional.Optional[time.Time]
		daysToComplete optional.Optional[int64]
		issuerIsMine   bool
		acceptorIsMine bool
		want           app.ContractCategory
		wantExpired    bool
		wantOverdue    bool
	}{
		{"outstanding", app.ContractStatusOutstanding, future, none, noDays, true, true, app.ContractCategoryOutstanding, false, false},
		{"outstanding expired issuer mine", app.ContractStatusOutstanding, past, none, noDays, true, false, app.ContractCategoryRequiresAttention, true, false},
		{"outstanding expired issuer not mine", app.ContractStatusOutstanding, past, none, noDays, false, true, app.ContractCategoryOther, true, false},
		{"outstanding expires exactly now", app.ContractStatusOutstanding, now, none, noDays, true, true, app.ContractCategoryOutstanding, false, false},
		{"outstanding with past delivery data", app.ContractStatusOutstanding, future, accepted, optional.New[int64](3), true, true, app.ContractCategoryOutstanding, false, false},
		{"in progress", app.ContractStatusInProgress, past, optional.New(now.AddDate(0, 0, -1)), optional.New[int64](3), true, true, app.ContractCategoryInProgress, false, false},
		{"in progress overdue acceptor mine", app.ContractStatusInProgress, future, accepted, optional.New[int64](3), false, true, app.ContractCategoryRequiresAttention, false, true},
		{"in progress overdue acceptor not mine", app.ContractStatusInProgress, future, accepted, optional.New[int64](3), true, false, app.ContractCategoryInProgress, false, true},
		{"in progress delivery due exactly now", app.ContractStatusInProgress, future, optional.New(now.AddDate(0, 0, -3)), optional.New[int64](3), true, true, app.ContractCategoryInProgress, false, false},
		{"in progress without days to complete", app.ContractStatusInProgress, past, accepted, noDays, true, true, app.ContractCategoryInProgress, false, false},
		{"in progress without date accepted", app.ContractStatusInProgress, past, none, optional.New[int64](3), true, true, app.ContractCategoryInProgress, false, false},
		{"failed", app.ContractStatusFailed, past, accepted, optional.New[int64](3), true, true, app.ContractCategoryOther, false, false},
		{"rejected", app.ContractStatusRejected, past, none, noDays, true, true, app.ContractCategoryOther, false, false},
		{"finished", app.ContractStatusFinished, past, none, noDays, true, true, app.ContractCategoryFinished, false, false},
		{"finished courier past deadline", app.ContractStatusFinished, past, accepted, optional.New[int64](3), true, true, app.ContractCategoryFinished, false, false},
		{"finished contractor", app.ContractStatusFinishedContractor, past, none, noDays, true, true, app.ContractCategoryFinished, false, false},
		{"finished issuer", app.ContractStatusFinishedIssuer, past, none, noDays, true, true, app.ContractCategoryFinished, false, false},
		{"reversed", app.ContractStatusReversed, past, none, noDays, true, true, app.ContractCategoryOther, false, false},
		{"cancelled", app.ContractStatusCancelled, past, none, noDays, true, true, app.ContractCategoryOther, false, false},
		{"deleted", app.ContractStatusDeleted, past, none, noDays, true, true, app.ContractCategoryOther, false, false},
		{"undefined", app.ContractStatusUndefined, future, none, noDays, true, true, app.ContractCategoryOther, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			category, isExpired, isOverdue := app.ContractStateAt(tc.status, tc.dateExpired, tc.dateAccepted, tc.daysToComplete, tc.issuerIsMine, tc.acceptorIsMine, now)
			xassert.Equal(t, tc.want, category)
			xassert.Equal(t, tc.wantExpired, isExpired)
			xassert.Equal(t, tc.wantOverdue, isOverdue)
		})
	}
}

func TestContractIsExpiredAt(t *testing.T) {
	now := time.Now()
	xassert.Equal(t, true, app.ContractIsExpiredAt(now.Add(-time.Hour), now))
	xassert.Equal(t, false, app.ContractIsExpiredAt(now.Add(time.Hour), now))
}

func TestContractDeliveryDeadline(t *testing.T) {
	accepted := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	got, ok := app.ContractDeliveryDeadline(optional.New(accepted), optional.New[int64](3))
	xassert.Equal(t, true, ok)
	xassert.Equal(t, accepted.AddDate(0, 0, 3), got)
	_, ok = app.ContractDeliveryDeadline(optional.Optional[time.Time]{}, optional.New[int64](3))
	xassert.Equal(t, false, ok)
	_, ok = app.ContractDeliveryDeadline(optional.New(accepted), optional.Optional[int64]{})
	xassert.Equal(t, false, ok)
}

func TestContractIsOverdueAt(t *testing.T) {
	now := time.Now()
	xassert.Equal(t, true, app.ContractIsOverdueAt(optional.New(now.AddDate(0, 0, -4)), optional.New[int64](3), now))
	xassert.Equal(t, false, app.ContractIsOverdueAt(optional.New(now.AddDate(0, 0, -1)), optional.New[int64](3), now))
	xassert.Equal(t, false, app.ContractIsOverdueAt(optional.Optional[time.Time]{}, optional.New[int64](3), now))
}

func TestContractIssuerEffective(t *testing.T) {
	issuer := &app.EveEntity{ID: 1, Name: "Issuer"}
	issuerCorp := &app.EveEntity{ID: 2, Name: "Issuer Corp"}
	t.Run("for corporation", func(t *testing.T) {
		c := app.Contract{ForCorporation: true, Issuer: issuer, IssuerCorporation: issuerCorp}
		xassert.Equal(t, issuerCorp, c.IssuerEffective())
	})
	t.Run("for character", func(t *testing.T) {
		c := app.Contract{ForCorporation: false, Issuer: issuer, IssuerCorporation: issuerCorp}
		xassert.Equal(t, issuer, c.IssuerEffective())
	})
}

func TestContractNameDisplay(t *testing.T) {
	cases := []struct {
		name     string
		contract *app.Contract
		want     string
	}{
		{
			"normal courier",
			&app.Contract{
				Type: app.ContractTypeCourier,
				StartSolarSystem: optional.New(&app.EntityShort{
					Name: "start",
				}),
				EndSolarSystem: optional.New(&app.EntityShort{
					Name: "end",
				}),
				Volume: optional.New[float64](42),
			},
			"start >> end (42 m3)",
		},
		{
			"broken courier",
			&app.Contract{
				Type:   app.ContractTypeCourier,
				Volume: optional.New[float64](42),
			},
			"? >> ? (42 m3)",
		},
		{
			"courier without volume",
			&app.Contract{
				Type: app.ContractTypeCourier,
				StartSolarSystem: optional.New(&app.EntityShort{
					Name: "start",
				}),
				EndSolarSystem: optional.New(&app.EntityShort{
					Name: "end",
				}),
			},
			"start >> end",
		},
		{
			"courier ignores items",
			&app.Contract{
				Type:  app.ContractTypeCourier,
				Items: []string{"Jupiter"},
			},
			"? >> ?",
		},
		{
			"single item exchange",
			&app.Contract{
				Type:  app.ContractTypeItemExchange,
				Items: []string{"Jupiter"},
			},
			"Jupiter",
		},
		{
			"multiple item exchange",
			&app.Contract{
				Type:  app.ContractTypeItemExchange,
				Items: []string{"Jupiter", "Mars"},
			},
			"[Multiple Items]",
		},
		{
			"single auction",
			&app.Contract{
				Type:  app.ContractTypeAuction,
				Items: []string{"Jupiter"},
			},
			"Jupiter",
		},
		{
			"multiple auction",
			&app.Contract{
				Type:  app.ContractTypeAuction,
				Items: []string{"Jupiter", "Mars"},
			},
			"[Multiple Items]",
		},
		{
			"single item with empty name",
			&app.Contract{
				Type:  app.ContractTypeItemExchange,
				Items: []string{""},
			},
			"[Single Item]",
		},
		{
			"no items",
			&app.Contract{
				Type: app.ContractTypeItemExchange,
			},
			"[Empty]",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.contract.NameDisplay()
			xassert.Equal(t, tc.want, got)
		})
	}
}

func TestContractUpdateNotification(t *testing.T) {
	acceptor := optional.New(&app.EveEntity{Name: "Bruce"})
	cases := []struct {
		name     string
		contract app.Contract
		want     string
	}{
		{
			"courier accepted",
			app.Contract{Type: app.ContractTypeCourier, Status: app.ContractStatusInProgress, Acceptor: acceptor},
			"Contract '? >> ?' has been accepted by Bruce",
		},
		{
			"courier delivered",
			app.Contract{Type: app.ContractTypeCourier, Status: app.ContractStatusFinished, Acceptor: acceptor},
			"Contract '? >> ?' has been delivered",
		},
		{
			"courier failed",
			app.Contract{Type: app.ContractTypeCourier, Status: app.ContractStatusFailed, Acceptor: acceptor},
			"Contract '? >> ?' has been failed by Bruce",
		},
		{
			"courier accepted without acceptor",
			app.Contract{Type: app.ContractTypeCourier, Status: app.ContractStatusInProgress},
			"Contract '? >> ?' has been accepted by ?",
		},
		{
			"courier outstanding",
			app.Contract{Type: app.ContractTypeCourier, Status: app.ContractStatusOutstanding},
			"",
		},
		{
			"item exchange accepted",
			app.Contract{Type: app.ContractTypeItemExchange, Status: app.ContractStatusFinished, Acceptor: acceptor, Items: []string{"Jupiter"}},
			"Contract 'Jupiter' has been accepted by Bruce",
		},
		{
			"item exchange outstanding",
			app.Contract{Type: app.ContractTypeItemExchange, Status: app.ContractStatusOutstanding, Items: []string{"Jupiter"}},
			"",
		},
		{
			"auction finished",
			app.Contract{Type: app.ContractTypeAuction, Status: app.ContractStatusFinished, Acceptor: acceptor, Items: []string{"Jupiter"}},
			"",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.contract.UpdateNotification())
		})
	}
}

func TestContractTopBid(t *testing.T) {
	t.Run("should return highest bid", func(t *testing.T) {
		b1 := &app.ContractBid{BidID: 1, Amount: 100}
		b2 := &app.ContractBid{BidID: 2, Amount: 300}
		b3 := &app.ContractBid{BidID: 3, Amount: 200}
		got, ok := app.ContractTopBid([]*app.ContractBid{b1, b2, b3})
		xassert.Equal(t, true, ok)
		xassert.Equal(t, b2, got)
	})
	t.Run("should return earliest of equal bids", func(t *testing.T) {
		early := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
		late := early.Add(time.Hour)
		cases := []struct {
			name   string
			winner *app.ContractBid
			other  *app.ContractBid
		}{
			{
				"earlier date",
				&app.ContractBid{BidID: 2, Amount: 100, DateBid: early},
				&app.ContractBid{BidID: 1, Amount: 100, DateBid: late},
			},
			{
				"same date, lower ID",
				&app.ContractBid{BidID: 1, Amount: 100, DateBid: early},
				&app.ContractBid{BidID: 2, Amount: 100, DateBid: early},
			},
		}
		for _, tc := range cases {
			for _, bids := range [][]*app.ContractBid{{tc.winner, tc.other}, {tc.other, tc.winner}} {
				t.Run(tc.name, func(t *testing.T) {
					got, ok := app.ContractTopBid(bids)
					xassert.Equal(t, true, ok)
					xassert.Equal(t, tc.winner, got)
				})
			}
		}
	})
	t.Run("should prefer higher amount over earlier date", func(t *testing.T) {
		early := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
		b1 := &app.ContractBid{BidID: 1, Amount: 100, DateBid: early}
		b2 := &app.ContractBid{BidID: 2, Amount: 200, DateBid: early.Add(time.Hour)}
		got, ok := app.ContractTopBid([]*app.ContractBid{b1, b2})
		xassert.Equal(t, true, ok)
		xassert.Equal(t, b2, got)
	})
	t.Run("should report when there are no bids", func(t *testing.T) {
		_, ok := app.ContractTopBid(nil)
		xassert.Equal(t, false, ok)
	})
}
