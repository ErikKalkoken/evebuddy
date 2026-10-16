package storage_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

// TestContractEnumsRoundTrip ensures every enum value is stored and read back unchanged for both owners.
func TestContractEnumsRoundTrip(t *testing.T) {
	db, st, f := testutil.NewDBInMemory()
	defer db.Close()
	type values struct {
		availability app.ContractAvailability
		status       app.ContractStatus
		typ          app.ContractType
	}
	var cases []values
	for _, v := range []app.ContractAvailability{
		app.ContractAvailabilityAlliance,
		app.ContractAvailabilityCorporation,
		app.ContractAvailabilityPrivate,
		app.ContractAvailabilityPublic,
	} {
		cases = append(cases, values{v, app.ContractStatusOutstanding, app.ContractTypeItemExchange})
	}
	for _, v := range []app.ContractStatus{
		app.ContractStatusCancelled,
		app.ContractStatusDeleted,
		app.ContractStatusFailed,
		app.ContractStatusFinished,
		app.ContractStatusFinishedContractor,
		app.ContractStatusFinishedIssuer,
		app.ContractStatusInProgress,
		app.ContractStatusOutstanding,
		app.ContractStatusRejected,
		app.ContractStatusReversed,
	} {
		cases = append(cases, values{app.ContractAvailabilityPublic, v, app.ContractTypeItemExchange})
	}
	for _, v := range []app.ContractType{
		app.ContractTypeAuction,
		app.ContractTypeCourier,
		app.ContractTypeItemExchange,
		app.ContractTypeLoan,
		app.ContractTypeUnknown,
	} {
		cases = append(cases, values{app.ContractAvailabilityPublic, app.ContractStatusOutstanding, v})
	}
	for _, tc := range cases {
		name := tc.availability.String() + "/" + tc.status.String() + "/" + tc.typ.String()
		t.Run("character "+name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			o := f.CreateCharacterContract(storage.CreateCharacterContractParams{
				Availability:   tc.availability,
				Status:         tc.status,
				StatusNotified: tc.status,
				Type:           tc.typ,
			})
			got, err := st.GetCharacterContract(t.Context(), o.CharacterID, o.ContractID)
			require.NoError(t, err)
			xassert.Equal(t, tc.availability, got.Availability)
			xassert.Equal(t, tc.status, got.Status)
			xassert.Equal(t, tc.status, got.StatusNotified)
			xassert.Equal(t, tc.typ, got.Type)
		})
		t.Run("corporation "+name, func(t *testing.T) {
			testutil.MustTruncateTables(db)
			o := f.CreateCorporationContract(storage.CreateCorporationContractParams{
				Availability:   tc.availability,
				Status:         tc.status,
				StatusNotified: tc.status,
				Type:           tc.typ,
			})
			got, err := st.GetCorporationContract(t.Context(), o.CorporationID, o.ContractID)
			require.NoError(t, err)
			xassert.Equal(t, tc.availability, got.Availability)
			xassert.Equal(t, tc.status, got.Status)
			xassert.Equal(t, tc.status, got.StatusNotified)
			xassert.Equal(t, tc.typ, got.Type)
		})
	}
}
