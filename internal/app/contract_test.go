package app_test

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

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

func TestContractType(t *testing.T) {
	xassert.Equal(t, "auction", app.ContractTypeAuction.String())
}

func TestContractTypeUnknown(t *testing.T) {
	xassert.Equal(t, "?", app.ContractType(99).String())
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

func TestContractCategoryAt(t *testing.T) {
	now := time.Now()
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
	}{
		{"outstanding", app.ContractStatusOutstanding, future, none, noDays, true, true, app.ContractCategoryOutstanding},
		{"outstanding expired issuer mine", app.ContractStatusOutstanding, past, none, noDays, true, false, app.ContractCategoryRequiresAttention},
		{"outstanding expired issuer not mine", app.ContractStatusOutstanding, past, none, noDays, false, true, app.ContractCategoryOther},
		{"in progress", app.ContractStatusInProgress, past, optional.New(now.AddDate(0, 0, -1)), optional.New[int64](3), true, true, app.ContractCategoryInProgress},
		{"in progress overdue acceptor mine", app.ContractStatusInProgress, future, accepted, optional.New[int64](3), false, true, app.ContractCategoryRequiresAttention},
		{"in progress overdue acceptor not mine", app.ContractStatusInProgress, future, accepted, optional.New[int64](3), true, false, app.ContractCategoryInProgress},
		{"in progress without days to complete", app.ContractStatusInProgress, past, accepted, noDays, true, true, app.ContractCategoryInProgress},
		{"in progress without date accepted", app.ContractStatusInProgress, past, none, optional.New[int64](3), true, true, app.ContractCategoryInProgress},
		{"failed", app.ContractStatusFailed, future, none, noDays, true, true, app.ContractCategoryOther},
		{"rejected", app.ContractStatusRejected, future, none, noDays, true, true, app.ContractCategoryOther},
		{"finished", app.ContractStatusFinished, past, none, noDays, true, true, app.ContractCategoryFinished},
		{"finished contractor", app.ContractStatusFinishedContractor, past, none, noDays, true, true, app.ContractCategoryFinished},
		{"finished issuer", app.ContractStatusFinishedIssuer, past, none, noDays, true, true, app.ContractCategoryFinished},
		{"reversed", app.ContractStatusReversed, past, none, noDays, true, true, app.ContractCategoryOther},
		{"cancelled", app.ContractStatusCancelled, past, none, noDays, true, true, app.ContractCategoryOther},
		{"deleted", app.ContractStatusDeleted, past, none, noDays, true, true, app.ContractCategoryOther},
		{"undefined", app.ContractStatusUndefined, future, none, noDays, true, true, app.ContractCategoryOther},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := app.ContractCategoryAt(tc.status, tc.dateExpired, tc.dateAccepted, tc.daysToComplete, tc.issuerIsMine, tc.acceptorIsMine, now)
			xassert.Equal(t, tc.want, got)
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

func TestCharacterContractIssuerEffective(t *testing.T) {
	issuer := &app.EveEntity{ID: 1, Name: "Issuer"}
	issuerCorp := &app.EveEntity{ID: 2, Name: "Issuer Corp"}
	t.Run("for corporation", func(t *testing.T) {
		cc := app.CharacterContract{ForCorporation: true, Issuer: issuer, IssuerCorporation: issuerCorp}
		xassert.Equal(t, issuerCorp, cc.IssuerEffective())
		corp := app.CorporationContract{ForCorporation: true, Issuer: issuer, IssuerCorporation: issuerCorp}
		xassert.Equal(t, issuerCorp, corp.IssuerEffective())
	})
	t.Run("for character", func(t *testing.T) {
		cc := app.CharacterContract{ForCorporation: false, Issuer: issuer, IssuerCorporation: issuerCorp}
		xassert.Equal(t, issuer, cc.IssuerEffective())
		corp := app.CorporationContract{ForCorporation: false, Issuer: issuer, IssuerCorporation: issuerCorp}
		xassert.Equal(t, issuer, corp.IssuerEffective())
	})
}

func TestCorporationContractNameDisplay(t *testing.T) {
	cc := app.CorporationContract{
		Type:  app.ContractTypeItemExchange,
		Items: []string{"Jupiter"},
	}
	xassert.Equal(t, "Jupiter", cc.NameDisplay())
}

func TestContractNameDisplay(t *testing.T) {
	cases := []struct {
		name     string
		contract *app.CharacterContract
		want     string
	}{
		{
			"normal courier",
			&app.CharacterContract{
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
			&app.CharacterContract{
				Type:   app.ContractTypeCourier,
				Volume: optional.New[float64](42),
			},
			"? >> ? (42 m3)",
		},
		{
			"single item exchange",
			&app.CharacterContract{
				Type:  app.ContractTypeItemExchange,
				Items: []string{"Jupiter"},
			},
			"Jupiter",
		},
		{
			"multiple item exchange",
			&app.CharacterContract{
				Type:  app.ContractTypeItemExchange,
				Items: []string{"Jupiter", "Mars"},
			},
			"[Multiple Items]",
		},
		{
			"single auction",
			&app.CharacterContract{
				Type:  app.ContractTypeAuction,
				Items: []string{"Jupiter"},
			},
			"Jupiter",
		},
		{
			"multiple auction",
			&app.CharacterContract{
				Type:  app.ContractTypeAuction,
				Items: []string{"Jupiter", "Mars"},
			},
			"[Multiple Items]",
		},
		{
			"single item with empty name",
			&app.CharacterContract{
				Type:  app.ContractTypeItemExchange,
				Items: []string{""},
			},
			"[Single Item]",
		},
		{
			"no items",
			&app.CharacterContract{
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

func TestCharacterContractDisplayName(t *testing.T) {
	cases := []struct {
		name     string
		contract *app.CharacterContract
		want     string
	}{
		{
			"courier contract",
			&app.CharacterContract{
				Type:             app.ContractTypeCourier,
				Volume:           optional.New[float64](10),
				StartSolarSystem: optional.New(&app.EntityShort{Name: "Start"}),
				EndSolarSystem:   optional.New(&app.EntityShort{Name: "End"}),
			},
			"Start >> End (10 m3)",
		},
		{
			"courier contract without solar systems",
			&app.CharacterContract{
				Type:   app.ContractTypeCourier,
				Volume: optional.New[float64](10),
			},
			"? >> ? (10 m3)",
		},
		{
			"non-courier contract with multiple items",
			&app.CharacterContract{
				Type:  app.ContractTypeItemExchange,
				Items: []string{"first", "second"},
			},
			"[Multiple Items]",
		},
		{
			"non-courier contract with single items",
			&app.CharacterContract{
				Type:  app.ContractTypeItemExchange,
				Items: []string{"first"},
			},
			"first",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			xassert.Equal(t, tc.want, tc.contract.NameDisplay())
		})
	}
}
