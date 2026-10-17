package contractsync

import (
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestSplit(t *testing.T) {
	contracts := []ESIContract{
		{ContractId: 1},
		{ContractId: 2},
		{ContractId: 3},
	}
	current := []*app.Contract{{ContractID: 2}, {ContractID: 4}}
	newContracts, existing := split(contracts, current)
	xassert.Equal(t, []int64{1, 3}, contractIDs(newContracts))
	xassert.Equal(t, []existingContract{{ESI: contracts[1], Stored: current[0]}}, existing)
}

func TestStaleIDs(t *testing.T) {
	cases := []struct {
		status    app.ContractStatus
		wantStale bool
	}{
		{app.ContractStatusCancelled, true},
		{app.ContractStatusDeleted, true},
		{app.ContractStatusFailed, true},
		{app.ContractStatusInProgress, true},
		{app.ContractStatusOutstanding, true},
		{app.ContractStatusRejected, true},
		{app.ContractStatusFinished, false},
		{app.ContractStatusFinishedContractor, false},
		{app.ContractStatusFinishedIssuer, false},
		{app.ContractStatusReversed, false},
	}
	for _, tc := range cases {
		t.Run(tc.status.String(), func(t *testing.T) {
			current := []*app.Contract{
				{ContractID: 1, Status: tc.status},
				{ContractID: 2, Status: tc.status}, // still in ESI
			}
			contracts := []ESIContract{{ContractId: 2}}
			got := staleIDs(current, contracts)
			xassert.Equal(t, tc.wantStale, got.Contains(1))
			xassert.Equal(t, false, got.Contains(2))
		})
	}
}

func contractIDs(contracts []ESIContract) []int64 {
	var ids []int64
	for _, c := range contracts {
		ids = append(ids, c.ContractId)
	}
	return ids
}
