package app_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
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
		{app.ContractStatusReversed, "Reversed", theme.ColorNameForeground},
	}
	for _, tc := range cases {
		t.Run(tc.status.String(), func(t *testing.T) {
			got := tc.status.DisplayRichText()
			want := iwidget.RichTextSegmentsFromText(tc.wantText,
				widget.RichTextStyle{
					ColorName: tc.wantColor,
				},
			)
			xassert.Equal(t, want, got)
		})
	}
}

func TestContractType(t *testing.T) {
	xassert.Equal(t, "auction", app.ContractTypeAuction.String())
}

func TestContractAvailabilityDisplay(t *testing.T) {
	xassert.Equal(t, "Private", app.ContractAvailabilityPrivate.Display())
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
