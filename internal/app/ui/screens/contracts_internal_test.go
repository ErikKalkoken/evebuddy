package screens

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

func TestContractsFilter_Match(t *testing.T) {
	r := contractRow{
		assigneeName: "Alice",
		category:     app.ContractCategoryOutstanding,
		issuerName:   "Bruce",
		tags:         set.Of("alpha"),
		typeName:     "Item Exchange",
	}
	finished := r
	finished.category = app.ContractCategoryFinished
	outstanding := app.ContractCategoryOutstanding.Display()
	for _, tc := range []struct {
		name   string
		filter contractsFilter
		row    contractRow
		want   bool
	}{
		{"no filter", contractsFilter{}, r, true},
		{"no filter includes finished", contractsFilter{}, finished, true},
		{"assignee matches", contractsFilter{assignee: "Alice"}, r, true},
		{"assignee differs", contractsFilter{assignee: "Other"}, r, false},
		{"issuer differs", contractsFilter{issuer: "Other"}, r, false},
		{"status matches", contractsFilter{status: outstanding}, r, true},
		{"status differs", contractsFilter{status: outstanding}, finished, false},
		{"tag matches", contractsFilter{tag: "alpha"}, r, true},
		{"tag missing", contractsFilter{tag: "bravo"}, r, false},
		{"type differs", contractsFilter{typeName: "Courier"}, r, false},
		{"all match", contractsFilter{assignee: "Alice", issuer: "Bruce", status: outstanding, tag: "alpha", typeName: "Item Exchange"}, r, true},
		{"one of many differs", contractsFilter{assignee: "Alice", typeName: "Courier"}, r, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.filter.match(tc.row))
		})
	}
}

func TestContractsStatusOptions(t *testing.T) {
	assert.Equal(t, []string{"Outstanding", "In progress", "Requires attention", "Finished", "Other"}, contractsStatusOptions())
}

func TestMakeContractStatusDisplay(t *testing.T) {
	got := makeContractStatusDisplay(app.ContractStatusOutstanding, app.ContractCategoryRequiresAttention)
	want := xwidget.RichTextSegmentsFromText("Outstanding", widget.RichTextStyle{ColorName: theme.ColorNameError})
	assert.Equal(t, want, got)
}

func TestContracts_Filter(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	rows := []contractRow{
		{contractID: 1, category: app.ContractCategoryOutstanding, typeName: "Courier"},
		{contractID: 2, category: app.ContractCategoryInProgress, typeName: "Item Exchange"},
		{contractID: 3, category: app.ContractCategoryFinished, typeName: "Courier"},
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
		require.Len(t, a.rowsFiltered, 3) // all by default
		return a
	}
	finished := app.ContractCategoryFinished.Display()
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newContracts(t, true, false)
		a.filterChip.SetSelected(map[string]string{contractsFilterType: "Courier"})
		assert.Len(t, a.rowsFiltered, 2)
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newContracts(t, false, false)
		a.selectType.SetSelected("Courier")
		assert.Len(t, a.rowsFiltered, 2)
	})
	t.Run("can filter status on mobile", func(t *testing.T) {
		a := newContracts(t, true, false)
		a.filterChip.SetSelected(map[string]string{contractsFilterStatus: finished})
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 3, a.rowsFiltered[0].contractID)
		}
	})
	t.Run("can filter status on desktop", func(t *testing.T) {
		a := newContracts(t, false, false)
		a.selectStatus.SetSelected(finished)
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 3, a.rowsFiltered[0].contractID)
		}
	})
	t.Run("can combine status and filter on mobile", func(t *testing.T) {
		a := newContracts(t, true, false)
		a.filterChip.SetSelected(map[string]string{
			contractsFilterStatus: app.ContractCategoryOutstanding.Display(),
			contractsFilterType:   "Courier",
		})
		if assert.Len(t, a.rowsFiltered, 1) {
			assert.EqualValues(t, 1, a.rowsFiltered[0].contractID)
		}
	})
	t.Run("shows all filters on mobile", func(t *testing.T) {
		a := newContracts(t, true, false)
		assert.Equal(t, map[string]string{
			contractsFilterAssignee: "",
			contractsFilterIssuer:   "",
			contractsFilterStatus:   "",
			contractsFilterTag:      "",
			contractsFilterType:     "",
		}, a.filterChip.Selected())
	})
	t.Run("hides tag filter for corporation on mobile", func(t *testing.T) {
		a := newContracts(t, true, true)
		assert.Equal(t, map[string]string{
			contractsFilterAssignee: "",
			contractsFilterIssuer:   "",
			contractsFilterStatus:   "",
			contractsFilterType:     "",
		}, a.filterChip.Selected())
	})
}
