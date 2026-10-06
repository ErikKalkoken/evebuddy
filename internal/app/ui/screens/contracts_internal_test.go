package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
)

func TestContractsFilter_Match(t *testing.T) {
	r := contractRow{
		assigneeName: "Alice",
		isActive:     true,
		issuerName:   "Bruce",
		status:       app.ContractStatusOutstanding,
		tags:         set.Of("alpha"),
		typeName:     "Item Exchange",
	}
	inProgress := r
	inProgress.status = app.ContractStatusInProgress
	withIssue := r
	withIssue.hasIssue = true
	history := contractRow{isHistory: true}
	for _, tc := range []struct {
		name   string
		filter contractsFilter
		row    contractRow
		want   bool
	}{
		{"no filter", contractsFilter{}, r, true},
		{"assignee matches", contractsFilter{assignee: "Alice"}, r, true},
		{"assignee differs", contractsFilter{assignee: "Other"}, r, false},
		{"issuer differs", contractsFilter{issuer: "Other"}, r, false},
		{"tag matches", contractsFilter{tag: "alpha"}, r, true},
		{"tag missing", contractsFilter{tag: "bravo"}, r, false},
		{"type differs", contractsFilter{typeName: "Courier"}, r, false},
		{"all active matches", contractsFilter{status: contractStatusAllActive}, r, true},
		{"all active but history", contractsFilter{status: contractStatusAllActive}, history, false},
		{"outstanding matches", contractsFilter{status: contractStatusOutstanding}, r, true},
		{"outstanding but in progress", contractsFilter{status: contractStatusOutstanding}, inProgress, false},
		{"in progress matches", contractsFilter{status: contractStatusInProgress}, inProgress, true},
		{"in progress but outstanding", contractsFilter{status: contractStatusInProgress}, r, false},
		{"has issue matches", contractsFilter{status: contractStatusHasIssue}, withIssue, true},
		{"has issue but none", contractsFilter{status: contractStatusHasIssue}, r, false},
		{"history matches", contractsFilter{status: contractStatusHistory}, history, true},
		{"history but active", contractsFilter{status: contractStatusHistory}, r, false},
		{"all match", contractsFilter{assignee: "Alice", issuer: "Bruce", status: contractStatusAllActive, tag: "alpha", typeName: "Item Exchange"}, r, true},
		{"one of many differs", contractsFilter{assignee: "Alice", typeName: "Courier"}, r, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestContracts_Filter(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []contractRow{
		{contractID: 1, isActive: true, typeName: "Courier"},
		{contractID: 2, isActive: true, typeName: "Item Exchange"},
		{contractID: 3, isHistory: true, typeName: "Courier"},
	}
	newContracts := func(t *testing.T, isMobile, forCorporation bool) *Contracts {
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		})
		var a *Contracts
		if forCorporation {
			a = NewContractsForCorporation(u)
		} else {
			a = NewContractsForCharacters(u)
		}
		a.rows = rows
		a.filterRowsAsync("")
		require.Len(t, a.rowsFiltered, 2) // active by default
		return a
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newContracts(t, true, false)
		a.filterChip.SetSelected(map[string]string{contractsFilterType: "Courier"})
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 1, a.rowsFiltered[0].contractID)
		}
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newContracts(t, false, false)
		a.selectType.SetSelected("Courier")
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 1, a.rowsFiltered[0].contractID)
		}
	})
	t.Run("can switch status on mobile", func(t *testing.T) {
		a := newContracts(t, true, false)
		a.selectStatus.SetSelected(contractStatusHistory)
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 3, a.rowsFiltered[0].contractID)
		}
	})
	t.Run("can combine status and filter on mobile", func(t *testing.T) {
		a := newContracts(t, true, false)
		a.selectStatus.SetSelected(contractStatusHistory)
		a.filterChip.SetSelected(map[string]string{contractsFilterType: "Courier"})
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 3, a.rowsFiltered[0].contractID)
		}
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newContracts(t, true, false)
		assert.Equal(t, map[string]string{
			contractsFilterAssignee: "",
			contractsFilterIssuer:   "",
			contractsFilterTag:      "",
			contractsFilterType:     "",
		}, a.filterChip.Selected())
	})
	t.Run("hides tag filter for corporation on mobile", func(t *testing.T) {
		a := newContracts(t, true, true)
		assert.Equal(t, map[string]string{
			contractsFilterAssignee: "",
			contractsFilterIssuer:   "",
			contractsFilterType:     "",
		}, a.filterChip.Selected())
	})
}
