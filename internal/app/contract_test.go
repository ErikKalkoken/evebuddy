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
		wantActive   bool
		wantFinished bool
	}{
		{app.ContractStatusUndefined, false, false},
		{app.ContractStatusOutstanding, true, false},
		{app.ContractStatusInProgress, true, false},
		{app.ContractStatusDeleted, false, false},
		{app.ContractStatusCancelled, false, false},
		{app.ContractStatusFinished, false, true},
		{app.ContractStatusFinishedContractor, false, true},
		{app.ContractStatusFinishedIssuer, false, true},
		{app.ContractStatusReversed, false, true},
		{app.ContractStatusFailed, false, false},
		{app.ContractStatusRejected, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.status.String(), func(t *testing.T) {
			xassert.Equal(t, tc.wantActive, tc.status.IsActive())
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

func TestContractCategory(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)
	cases := []struct {
		name           string
		status         app.ContractStatus
		dateExpired    time.Time
		dateAccepted   optional.Optional[time.Time]
		daysToComplete optional.Optional[int64]
		want           app.ContractCategory
	}{
		{"outstanding", app.ContractStatusOutstanding, future, optional.Optional[time.Time]{}, optional.Optional[int64]{}, app.ContractCategoryOutstanding},
		{"outstanding expired", app.ContractStatusOutstanding, past, optional.Optional[time.Time]{}, optional.Optional[int64]{}, app.ContractCategoryRequiresAttention},
		{"in progress", app.ContractStatusInProgress, past, optional.New(now.AddDate(0, 0, -1)), optional.New[int64](3), app.ContractCategoryInProgress},
		{"in progress overdue", app.ContractStatusInProgress, future, optional.New(now.AddDate(0, 0, -4)), optional.New[int64](3), app.ContractCategoryRequiresAttention},
		{"in progress without days to complete", app.ContractStatusInProgress, past, optional.New(now.AddDate(0, 0, -4)), optional.Optional[int64]{}, app.ContractCategoryInProgress},
		{"in progress without date accepted", app.ContractStatusInProgress, past, optional.Optional[time.Time]{}, optional.New[int64](3), app.ContractCategoryInProgress},
		{"failed", app.ContractStatusFailed, future, optional.Optional[time.Time]{}, optional.Optional[int64]{}, app.ContractCategoryRequiresAttention},
		{"rejected", app.ContractStatusRejected, future, optional.Optional[time.Time]{}, optional.Optional[int64]{}, app.ContractCategoryRequiresAttention},
		{"finished", app.ContractStatusFinished, past, optional.Optional[time.Time]{}, optional.Optional[int64]{}, app.ContractCategoryFinished},
		{"finished contractor", app.ContractStatusFinishedContractor, past, optional.Optional[time.Time]{}, optional.Optional[int64]{}, app.ContractCategoryFinished},
		{"finished issuer", app.ContractStatusFinishedIssuer, past, optional.Optional[time.Time]{}, optional.Optional[int64]{}, app.ContractCategoryFinished},
		{"reversed", app.ContractStatusReversed, past, optional.Optional[time.Time]{}, optional.Optional[int64]{}, app.ContractCategoryFinished},
		{"cancelled", app.ContractStatusCancelled, past, optional.Optional[time.Time]{}, optional.Optional[int64]{}, app.ContractCategoryOther},
		{"deleted", app.ContractStatusDeleted, past, optional.Optional[time.Time]{}, optional.Optional[int64]{}, app.ContractCategoryOther},
		{"undefined", app.ContractStatusUndefined, future, optional.Optional[time.Time]{}, optional.Optional[int64]{}, app.ContractCategoryOther},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cc := app.CharacterContract{Status: tc.status, DateExpired: tc.dateExpired, DateAccepted: tc.dateAccepted, DaysToComplete: tc.daysToComplete}
			xassert.Equal(t, tc.want, cc.Category())
			corp := app.CorporationContract{Status: tc.status, DateExpired: tc.dateExpired, DateAccepted: tc.dateAccepted, DaysToComplete: tc.daysToComplete}
			xassert.Equal(t, tc.want, corp.Category())
		})
	}
}

func TestCharacterContractIsExpired(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		cc := app.CharacterContract{DateExpired: time.Now().Add(-time.Hour)}
		xassert.Equal(t, true, cc.IsExpired())
		corp := app.CorporationContract{DateExpired: time.Now().Add(-time.Hour)}
		xassert.Equal(t, true, corp.IsExpired())
	})
	t.Run("not expired", func(t *testing.T) {
		cc := app.CharacterContract{DateExpired: time.Now().Add(time.Hour)}
		xassert.Equal(t, false, cc.IsExpired())
		corp := app.CorporationContract{DateExpired: time.Now().Add(time.Hour)}
		xassert.Equal(t, false, corp.IsExpired())
	})
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
