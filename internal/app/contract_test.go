package app_test

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestContractStatusString(t *testing.T) {
	xassert.Equal(t, "cancelled", app.ContractStatusCancelled.String())
}

func TestContractStatusDisplay(t *testing.T) {
	xassert.Equal(t, "Cancelled", app.ContractStatusCancelled.Display())
}

func TestContractStatusDisplayRichText(t *testing.T) {
	cases := []struct {
		status    app.ContractStatus
		wantText  string
		wantColor fyne.ThemeColorName
	}{
		{app.ContractStatusOutstanding, "Outstanding", theme.ColorNameWarning},
		{app.ContractStatusInProgress, "In Progress", theme.ColorNameForeground},
		{app.ContractStatusFinished, "Finished", theme.ColorNameSuccess},
		{app.ContractStatusFailed, "Failed", theme.ColorNameError},
		{app.ContractStatusReversed, "Reversed", theme.ColorNameSuccess},
		{app.ContractStatusUndefined, "?", theme.ColorNameForeground},
	}
	for _, tc := range cases {
		t.Run(tc.status.String(), func(t *testing.T) {
			got := tc.status.DisplayRichText()
			want := xwidget.RichTextSegmentsFromText(tc.wantText,
				widget.RichTextStyle{
					ColorName: tc.wantColor,
				},
			)
			xassert.Equal(t, want, got)
		})
	}
}

func TestContractStatusPredicates(t *testing.T) {
	cases := []struct {
		status       app.ContractStatus
		wantActive   bool
		wantHistory  bool
		wantFinished bool
	}{
		{app.ContractStatusUndefined, false, false, false},
		{app.ContractStatusOutstanding, true, false, false},
		{app.ContractStatusInProgress, true, false, false},
		{app.ContractStatusDeleted, false, true, false},
		{app.ContractStatusCancelled, false, true, false},
		{app.ContractStatusFinished, false, true, true},
		{app.ContractStatusFinishedContractor, false, true, true},
		{app.ContractStatusFinishedIssuer, false, true, true},
		{app.ContractStatusReversed, false, true, true},
		{app.ContractStatusFailed, false, false, false},
		{app.ContractStatusRejected, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.status.String(), func(t *testing.T) {
			xassert.Equal(t, tc.wantActive, tc.status.IsActive())
			xassert.Equal(t, tc.wantHistory, tc.status.IsHistory())
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

func TestCharacterContractHasIssue(t *testing.T) {
	cases := []struct {
		name    string
		status  app.ContractStatus
		expired time.Time
		want    bool
	}{
		{"status has issue", app.ContractStatusFailed, time.Now().Add(time.Hour), true},
		{"expired while active", app.ContractStatusOutstanding, time.Now().Add(-time.Hour), true},
		{"not expired and no issue", app.ContractStatusOutstanding, time.Now().Add(time.Hour), false},
		{"expired but inactive", app.ContractStatusFinished, time.Now().Add(-time.Hour), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cc := app.CharacterContract{Status: tc.status, DateExpired: tc.expired}
			xassert.Equal(t, tc.want, cc.HasIssue())
			corp := app.CorporationContract{Status: tc.status, DateExpired: tc.expired}
			xassert.Equal(t, tc.want, corp.HasIssue())
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
