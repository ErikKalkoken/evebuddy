package screens

import (
	"fmt"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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
	inProgress := r
	inProgress.category = app.ContractCategoryInProgress
	attention := r
	attention.category = app.ContractCategoryRequiresAttention
	other := r
	other.category = app.ContractCategoryOther
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
		{"all open matches outstanding", contractsFilter{status: contractsStatusAllOpen}, r, true},
		{"all open matches in progress", contractsFilter{status: contractsStatusAllOpen}, inProgress, true},
		{"all open matches requires attention", contractsFilter{status: contractsStatusAllOpen}, attention, true},
		{"all open but finished", contractsFilter{status: contractsStatusAllOpen}, finished, false},
		{"all open but other", contractsFilter{status: contractsStatusAllOpen}, other, false},
		{"all matches finished", contractsFilter{status: contractsStatusAll}, finished, true},
		{"all matches outstanding", contractsFilter{status: contractsStatusAll}, r, true},
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
	assert.Equal(t, []string{
		"All open contracts",
		"Outstanding",
		"In progress",
		"Requires attention",
		"Finished",
		"Other",
		"All contracts",
	}, contractsStatusOptions())
}

func TestMakeContractStatusDisplay(t *testing.T) {
	got := makeContractStatusDisplay(app.ContractStatusOutstanding, app.ContractCategoryRequiresAttention)
	want := xwidget.RichTextSegmentsFromText("Outstanding (expired)", widget.RichTextStyle{ColorName: theme.ColorNameError})
	assert.Equal(t, want, got)
}

func TestContractStatusText(t *testing.T) {
	for _, tc := range []struct {
		status   app.ContractStatus
		category app.ContractCategory
		want     string
	}{
		{app.ContractStatusOutstanding, app.ContractCategoryRequiresAttention, "Outstanding (expired)"},
		{app.ContractStatusInProgress, app.ContractCategoryRequiresAttention, "In Progress (overdue)"},
		{app.ContractStatusOutstanding, app.ContractCategoryOutstanding, "Outstanding"},
		{app.ContractStatusInProgress, app.ContractCategoryInProgress, "In Progress"},
		{app.ContractStatusFailed, app.ContractCategoryRequiresAttention, "Failed"},
		{app.ContractStatusRejected, app.ContractCategoryRequiresAttention, "Rejected"},
		{app.ContractStatusFinished, app.ContractCategoryFinished, "Finished"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, contractStatusText(tc.status, tc.category))
		})
	}
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
		return a
	}
	contractIDs := func(a *Contracts) []int64 {
		return xslices.Map(a.rowsFiltered, func(r contractRow) int64 {
			return r.contractID
		})
	}
	finished := app.ContractCategoryFinished.Display()
	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("shows all open contracts by default mobile=%v", isMobile), func(t *testing.T) {
			a := newContracts(t, isMobile, false)
			assert.Equal(t, contractsStatusAllOpen, a.selectStatus.Selected)
			assert.ElementsMatch(t, []int64{1, 2}, contractIDs(a))
		})
		t.Run(fmt.Sprintf("can filter status mobile=%v", isMobile), func(t *testing.T) {
			a := newContracts(t, isMobile, false)
			a.selectStatus.SetSelected(finished)
			assert.ElementsMatch(t, []int64{3}, contractIDs(a))
		})
		t.Run(fmt.Sprintf("can show all contracts mobile=%v", isMobile), func(t *testing.T) {
			a := newContracts(t, isMobile, false)
			a.selectStatus.SetSelected(contractsStatusAll)
			assert.ElementsMatch(t, []int64{1, 2, 3}, contractIDs(a))
		})
	}
	t.Run("can filter on mobile", func(t *testing.T) {
		a := newContracts(t, true, false)
		a.filterChip.SetSelected(map[string]string{contractsFilterType: "Courier"})
		assert.ElementsMatch(t, []int64{1}, contractIDs(a))
	})
	t.Run("can filter on desktop", func(t *testing.T) {
		a := newContracts(t, false, false)
		a.selectType.SetSelected("Courier")
		assert.ElementsMatch(t, []int64{1}, contractIDs(a))
	})
	t.Run("can combine status and filter on mobile", func(t *testing.T) {
		a := newContracts(t, true, false)
		a.selectStatus.SetSelected(contractsStatusAll)
		a.filterChip.SetSelected(map[string]string{contractsFilterType: "Courier"})
		assert.ElementsMatch(t, []int64{1, 3}, contractIDs(a))
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

func TestContracts_OnUpdateCountsOpenContracts(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	expired := time.Now().Add(-time.Hour)
	newUI := func(t *testing.T) *testdouble.UIFake {
		return testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
	}
	t.Run("characters", func(t *testing.T) {
		c := factory.CreateCharacterFull()
		for _, arg := range []storage.CreateCharacterContractParams{
			{Status: app.ContractStatusOutstanding},
			{Status: app.ContractStatusOutstanding, DateExpired: expired},
			{Status: app.ContractStatusInProgress},
			{Status: app.ContractStatusFailed},
			{Status: app.ContractStatusFinished},
		} {
			arg.CharacterID = c.ID
			factory.CreateCharacterContract(arg)
		}
		a := NewContractsForCharacters(newUI(t))
		var got int
		a.OnUpdate = func(open int) {
			got = open
		}
		a.update(t.Context())
		assert.Equal(t, 4, got)
	})
	t.Run("corporation", func(t *testing.T) {
		corporation := factory.CreateCorporation()
		for _, arg := range []storage.CreateCorporationContractParams{
			{Status: app.ContractStatusOutstanding},
			{Status: app.ContractStatusOutstanding, DateExpired: expired},
			{Status: app.ContractStatusRejected},
			{Status: app.ContractStatusFinished},
		} {
			arg.CorporationID = corporation.ID
			factory.CreateCorporationContract(arg)
		}
		a := NewContractsForCorporation(newUI(t))
		a.corporation.Store(corporation)
		var got int
		a.OnUpdate = func(open int) {
			got = open
		}
		a.update(t.Context())
		assert.Equal(t, 3, got)
	})
}
