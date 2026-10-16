package screens

import (
	"fmt"
	"slices"
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
	"github.com/ErikKalkoken/evebuddy/internal/optional"
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
	got := makeContractStatusDisplay("Outstanding (expired)", app.ContractCategoryRequiresAttention)
	want := xwidget.RichTextSegmentsFromText("Outstanding (expired)", widget.RichTextStyle{ColorName: theme.ColorNameError})
	assert.Equal(t, want, got)
}

func TestContractStatusText(t *testing.T) {
	for _, tc := range []struct {
		status    app.ContractStatus
		isExpired bool
		isOverdue bool
		want      string
	}{
		{app.ContractStatusOutstanding, true, false, "Outstanding (expired)"},
		{app.ContractStatusInProgress, true, true, "In Progress (overdue)"},
		{app.ContractStatusOutstanding, false, false, "Outstanding"},
		{app.ContractStatusInProgress, true, false, "In Progress"},
		{app.ContractStatusFailed, true, true, "Failed"},
		{app.ContractStatusRejected, true, false, "Rejected"},
		{app.ContractStatusFinished, true, false, "Finished"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			assert.Equal(t, tc.want, contractStatusText(tc.status, tc.isExpired, tc.isOverdue))
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

func TestContracts_OnUpdateCountsContractsRequiringAttention(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	expired := time.Now().Add(-time.Hour)
	overdueAccepted := optional.New(time.Now().AddDate(0, 0, -4))
	newUI := func(t *testing.T) *testdouble.UIFake {
		return testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
	}
	t.Run("characters", func(t *testing.T) {
		c := factory.CreateCharacterFull()
		other := factory.CreateEveEntityCharacter()
		for _, arg := range []storage.CreateCharacterContractParams{
			{Status: app.ContractStatusOutstanding},
			{Status: app.ContractStatusOutstanding, DateExpired: expired},
			{Status: app.ContractStatusOutstanding, DateExpired: expired, IssuerID: other.ID},
			{Status: app.ContractStatusInProgress, AcceptorID: c.ID, DateAccepted: overdueAccepted, DaysToComplete: optional.New[int64](3)},
			{Status: app.ContractStatusInProgress, AcceptorID: other.ID, DateAccepted: overdueAccepted, DaysToComplete: optional.New[int64](3)},
			{Status: app.ContractStatusInProgress},
			{Status: app.ContractStatusFailed},
			{Status: app.ContractStatusFinished},
		} {
			arg.CharacterID = c.ID
			factory.CreateCharacterContract(arg)
		}
		a := NewContractsForCharacters(newUI(t))
		var got int
		a.OnUpdate = func(attention int) {
			got = attention
		}
		a.update(t.Context())
		assert.Equal(t, 2, got)
	})
	t.Run("corporation", func(t *testing.T) {
		ec := factory.CreateEveCorporation()
		corporation := factory.CreateCorporation(ec.ID)
		x := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: ec.ID})
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{ID: x.ID})
		_, err := st.GetOrCreateEveEntity(t.Context(), storage.CreateEveEntityParams{
			ID:       c.ID,
			Name:     c.EveCharacter.Name,
			Category: app.EveEntityCharacter,
		})
		if err != nil {
			t.Fatal(err)
		}
		member := factory.CreateEveEntityCharacter()
		for _, arg := range []storage.CreateCorporationContractParams{
			{Status: app.ContractStatusOutstanding},
			{Status: app.ContractStatusOutstanding, DateExpired: expired, IssuerID: c.ID},
			{Status: app.ContractStatusOutstanding, DateExpired: expired, IssuerID: member.ID, ForCorporation: true},
			{Status: app.ContractStatusOutstanding, DateExpired: expired, IssuerID: member.ID},
			{Status: app.ContractStatusInProgress, AcceptorID: ec.ID, DateAccepted: overdueAccepted, DaysToComplete: optional.New[int64](3)},
			{Status: app.ContractStatusRejected},
			{Status: app.ContractStatusFinished},
		} {
			arg.CorporationID = corporation.ID
			factory.CreateCorporationContract(arg)
		}
		a := NewContractsForCorporation(newUI(t))
		a.corporation.Store(corporation)
		var got int
		a.OnUpdate = func(attention int) {
			got = attention
		}
		a.update(t.Context())
		assert.Equal(t, 3, got)
	})
}

// Test values for rows created with init.
const (
	myID     = 1 // one of the user's characters
	otherID  = 2 // someone else
	myCorpID = 3 // corporation of one of the user's characters
)

var (
	myIDs      = set.Of[int64](myID)
	myCorpIDs  = set.Of[int64](myCorpID)
	noAcceptor = optional.Optional[*app.EveEntity]{}
)

func TestContractRow_UpdateTimeDependent(t *testing.T) {
	now := time.Now()
	t.Run("outstanding becomes expired", func(t *testing.T) {
		r := contractRow{status: app.ContractStatusOutstanding, dateExpired: now.Add(time.Hour)}
		r.init(myID, noAcceptor, myIDs, myCorpIDs, now)
		assert.Equal(t, app.ContractCategoryOutstanding, r.category)
		assert.False(t, r.isExpired)
		assert.False(t, r.updateTimeDependent(now))
		assert.True(t, r.updateTimeDependent(now.Add(2*time.Hour)))
		assert.Equal(t, app.ContractCategoryRequiresAttention, r.category)
		assert.True(t, r.isExpired)
		assert.Equal(t, "Outstanding (expired)", r.statusText)
	})
	t.Run("in progress becomes overdue", func(t *testing.T) {
		r := contractRow{
			status:         app.ContractStatusInProgress,
			dateAccepted:   optional.New(now),
			dateExpired:    now.Add(time.Hour),
			daysToComplete: optional.New[int64](3),
		}
		r.init(otherID, optional.New(&app.EveEntity{ID: myID}), myIDs, myCorpIDs, now)
		assert.Equal(t, app.ContractCategoryInProgress, r.category)
		assert.True(t, r.updateTimeDependent(now.AddDate(0, 0, 4)))
		assert.Equal(t, app.ContractCategoryRequiresAttention, r.category)
		assert.Equal(t, "In Progress (overdue)", r.statusText)
		assert.Equal(t, contractDeadlineDelivery, r.deadlineKind)
		assert.Equal(t, overdueSegments(), r.timeLeftDisplay)
	})
	t.Run("time left changes with time", func(t *testing.T) {
		r := contractRow{status: app.ContractStatusOutstanding, dateExpired: now.Add(3 * time.Hour)}
		r.init(myID, noAcceptor, myIDs, myCorpIDs, now)
		assert.Equal(t, "3h 0m", r.timeLeftText)
		assert.False(t, r.updateTimeDependent(now.Add(10*time.Second)), "no change within a minute")
		assert.True(t, r.updateTimeDependent(now.Add(5*time.Minute)))
		assert.Equal(t, "2h 55m", r.timeLeftText)
		assert.Equal(t, app.ContractCategoryOutstanding, r.category)
	})
	t.Run("expired contract of others does not require attention", func(t *testing.T) {
		r := contractRow{status: app.ContractStatusOutstanding, dateExpired: now.Add(time.Hour)}
		r.init(otherID, noAcceptor, myIDs, myCorpIDs, now)
		assert.True(t, r.updateTimeDependent(now.Add(2*time.Hour)))
		assert.Equal(t, app.ContractCategoryOther, r.category)
		assert.Equal(t, "Outstanding (expired)", r.statusText)
	})
	t.Run("overdue contract of others still shows as overdue", func(t *testing.T) {
		r := contractRow{
			status:         app.ContractStatusInProgress,
			dateAccepted:   optional.New(now),
			dateExpired:    now.Add(time.Hour),
			daysToComplete: optional.New[int64](3),
		}
		r.init(myID, optional.New(&app.EveEntity{ID: otherID}), myIDs, myCorpIDs, now)
		assert.True(t, r.updateTimeDependent(now.AddDate(0, 0, 4)))
		assert.Equal(t, app.ContractCategoryInProgress, r.category)
		assert.Equal(t, "In Progress (overdue)", r.statusText)
	})
}

func TestNewContractRow_Owner(t *testing.T) {
	o := &app.Contract{
		ID:         7,
		ContractID: 42,
		Issuer:     &app.EveEntity{ID: otherID},
		Status:     app.ContractStatusOutstanding,
	}
	t.Run("character", func(t *testing.T) {
		r := newContractRow(o, myID, "Bruce", false, myIDs, myCorpIDs)
		assert.EqualValues(t, myID, r.ownerID)
		assert.Equal(t, "Bruce", r.ownerName)
		assert.False(t, r.isCorporation)
		assert.Equal(t, &app.EveEntity{ID: myID, Name: "Bruce", Category: app.EveEntityCharacter}, r.owner())
		assert.EqualValues(t, 7, r.objectID)
		assert.EqualValues(t, 42, r.contractID)
	})
	t.Run("corporation", func(t *testing.T) {
		r := newContractRow(o, myCorpID, "Wayne Enterprises", true, myIDs, myCorpIDs)
		assert.EqualValues(t, myCorpID, r.ownerID)
		assert.Equal(t, "Wayne Enterprises", r.ownerName)
		assert.True(t, r.isCorporation)
		assert.Equal(t, &app.EveEntity{ID: myCorpID, Name: "Wayne Enterprises", Category: app.EveEntityCorporation}, r.owner())
	})
}

func TestNewContractRow_RequiresAttention(t *testing.T) {
	now := time.Now()
	corporation := &app.Corporation{ID: myCorpID}
	myCorp := &app.EveEntity{ID: myCorpID, Category: app.EveEntityCorporation}
	corpMember := &app.EveEntity{ID: otherID, Category: app.EveEntityCharacter}
	t.Run("expired contract issued for my corporation", func(t *testing.T) {
		o := &app.CorporationContract{
			Contract: app.Contract{
				DateExpired:       now.Add(-time.Hour),
				ForCorporation:    true,
				Issuer:            corpMember,
				IssuerCorporation: myCorp,
				Status:            app.ContractStatusOutstanding,
			},
		}
		r := newContractRow(&o.Contract, corporation.ID, corporation.NameOrZero(), true, myIDs, myCorpIDs)
		assert.Equal(t, app.ContractCategoryRequiresAttention, r.category)
	})
	t.Run("expired personal contract of a corporation member", func(t *testing.T) {
		o := &app.CorporationContract{
			Contract: app.Contract{
				DateExpired:       now.Add(-time.Hour),
				ForCorporation:    false,
				Issuer:            corpMember,
				IssuerCorporation: myCorp,
				Status:            app.ContractStatusOutstanding,
			},
		}
		r := newContractRow(&o.Contract, corporation.ID, corporation.NameOrZero(), true, myIDs, myCorpIDs)
		assert.Equal(t, app.ContractCategoryOther, r.category)
	})
	t.Run("overdue contract accepted for my corporation", func(t *testing.T) {
		o := &app.CorporationContract{
			Contract: app.Contract{
				Acceptor:          optional.New(myCorp),
				DateAccepted:      optional.New(now.AddDate(0, 0, -2)),
				DateExpired:       now.AddDate(0, 0, 10),
				DaysToComplete:    optional.New[int64](1),
				Issuer:            corpMember,
				IssuerCorporation: myCorp,
				Status:            app.ContractStatusInProgress,
				Type:              app.ContractTypeCourier,
			},
		}
		r := newContractRow(&o.Contract, corporation.ID, corporation.NameOrZero(), true, myIDs, myCorpIDs)
		assert.Equal(t, app.ContractCategoryRequiresAttention, r.category)
	})
}

func TestContracts_RefreshTimeDependent(t *testing.T) {
	db, st, _ := testutil.NewDBOnDisk(t)
	defer db.Close()
	now := time.Now()
	rows := []contractRow{
		{contractID: 1, status: app.ContractStatusOutstanding, dateExpired: now.Add(time.Hour)},
		{contractID: 2, status: app.ContractStatusOutstanding, dateExpired: now.AddDate(0, 0, 2)},
	}
	for i := range rows {
		rows[i].init(myID, noAcceptor, myIDs, myCorpIDs, now)
	}
	a := NewContractsForCharacters(testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	}))
	var calls, got int
	a.OnUpdate = func(attention int) {
		calls++
		got = attention
	}
	a.rows = rows
	a.selectStatus.SetSelected(app.ContractCategoryRequiresAttention.Display())
	assert.Empty(t, a.rowsFiltered)

	a.refreshTimeDependent(now.Add(90 * time.Minute))
	assert.Equal(t, 1, calls)
	assert.Equal(t, 1, got)
	assert.Equal(t, []int64{1}, xslices.Map(a.rowsFiltered, func(r contractRow) int64 {
		return r.contractID
	}))

	a.refreshTimeDependent(now.Add(91 * time.Minute))
	assert.Equal(t, 1, calls, "no update without changes")

	a.selectStatus.SetSelected(contractsStatusAll)
	before := a.rowsFiltered[slices.IndexFunc(a.rowsFiltered, func(r contractRow) bool { return r.contractID == 2 })].timeLeftText
	a.refreshTimeDependent(now.Add(5 * time.Hour))
	after := a.rowsFiltered[slices.IndexFunc(a.rowsFiltered, func(r contractRow) bool { return r.contractID == 2 })].timeLeftText
	assert.NotEqual(t, before, after, "time left is updated in the list")
	assert.Equal(t, 1, calls, "badge is not updated when the count is unchanged")
}

func overdueSegments() []widget.RichTextSegment {
	return xwidget.RichTextSegmentsFromText("OVERDUE", widget.RichTextStyle{ColorName: theme.ColorNameError})
}

func TestContractDeadline(t *testing.T) {
	expired := time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)
	accepted := optional.New(time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC))
	days := optional.New[int64](1)
	none := optional.Optional[time.Time]{}
	noDays := optional.Optional[int64]{}
	for _, tc := range []struct {
		name         string
		status       app.ContractStatus
		dateAccepted optional.Optional[time.Time]
		days         optional.Optional[int64]
		wantDeadline optional.Optional[time.Time]
		wantKind     contractDeadlineKind
	}{
		{"outstanding", app.ContractStatusOutstanding, none, days, optional.New(expired), contractDeadlineExpiry},
		{"in progress courier", app.ContractStatusInProgress, accepted, days, optional.New(accepted.MustValue().AddDate(0, 0, 1)), contractDeadlineDelivery},
		{"in progress without delivery data", app.ContractStatusInProgress, none, noDays, none, contractDeadlineNone},
		{"finished", app.ContractStatusFinished, accepted, days, none, contractDeadlineNone},
		{"failed", app.ContractStatusFailed, accepted, days, none, contractDeadlineNone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deadline, kind := contractDeadline(tc.status, expired, tc.dateAccepted, tc.days)
			assert.Equal(t, tc.wantDeadline, deadline)
			assert.Equal(t, tc.wantKind, kind)
		})
	}
}

func TestContractDeadlineKind(t *testing.T) {
	for _, tc := range []struct {
		kind       contractDeadlineKind
		wantLabel  string
		wantPassed string
	}{
		{contractDeadlineNone, "", ""},
		{contractDeadlineExpiry, "Expires", "EXPIRED"},
		{contractDeadlineDelivery, "Due", "OVERDUE"},
	} {
		assert.Equal(t, tc.wantLabel, tc.kind.label())
		assert.Equal(t, tc.wantPassed, tc.kind.passedText())
	}
}

func TestContractTimeLeftText(t *testing.T) {
	now := time.Now()
	t.Run("before deadline", func(t *testing.T) {
		got, isPassed := contractTimeLeftText(optional.New(now.Add(3*time.Hour)), "OVERDUE", now)
		assert.Equal(t, "3h 0m", got)
		assert.False(t, isPassed)
	})
	t.Run("after deadline", func(t *testing.T) {
		got, isPassed := contractTimeLeftText(optional.New(now.Add(-time.Hour)), "OVERDUE", now)
		assert.Equal(t, "OVERDUE", got)
		assert.True(t, isPassed)
	})
	t.Run("no deadline", func(t *testing.T) {
		got, isPassed := contractTimeLeftText(optional.Optional[time.Time]{}, "", now)
		assert.Equal(t, "", got)
		assert.False(t, isPassed)
	})
}

func TestMakeTimeLeftDisplay(t *testing.T) {
	assert.Equal(t, xwidget.RichTextSegmentsFromText("3h"), makeTimeLeftDisplay("3h", false))
	assert.Equal(t, overdueSegments(), makeTimeLeftDisplay("OVERDUE", true))
}

func TestCompareContractDeadlines(t *testing.T) {
	now := time.Now()
	early := optional.New(now)
	late := optional.New(now.Add(time.Hour))
	none := optional.Optional[time.Time]{}
	assert.Equal(t, -1, compareContractDeadlines(early, late))
	assert.Equal(t, 1, compareContractDeadlines(late, early))
	assert.Equal(t, -1, compareContractDeadlines(late, none))
	assert.Equal(t, 1, compareContractDeadlines(none, early))
	assert.Equal(t, 0, compareContractDeadlines(none, none))
}

func TestMakeContractDateString(t *testing.T) {
	d := time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)
	assert.Equal(t, d.Format(app.DateTimeFormat), makeContractDateString(d, ""))
	assert.Equal(t, d.Format(app.DateTimeFormat)+" (OVERDUE)", makeContractDateString(d, makeContractDeadlineNote(d, true, "OVERDUE")))
}

func TestContracts_CorporationReloadsWhenCharactersChange(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	corporation := factory.CreateCorporation()
	c := factory.CreateCharacterFull()
	for _, arg := range []storage.CreateEveEntityParams{
		{ID: corporation.ID, Name: corporation.EveCorporation.Name, Category: app.EveEntityCorporation},
		{ID: c.ID, Name: c.EveCharacter.Name, Category: app.EveEntityCharacter},
	} {
		if _, err := st.GetOrCreateEveEntity(t.Context(), arg); err != nil {
			t.Fatal(err)
		}
	}
	factory.CreateCorporationContract(storage.CreateCorporationContractParams{
		CorporationID: corporation.ID,
		Status:        app.ContractStatusOutstanding,
		DateExpired:   time.Now().Add(-time.Hour),
		IssuerID:      c.ID,
	})
	u := testdouble.NewUIFake(testdouble.UIParams{
		App:     test.NewTempApp(t),
		Storage: st,
	})
	a := NewContractsForCorporation(u)
	a.corporation.Store(corporation)
	var got int
	a.OnUpdate = func(attention int) {
		got = attention
	}
	a.update(t.Context())
	assert.Equal(t, 1, got, "issued by one of the user's characters")

	if err := st.DeleteCharacter(t.Context(), c.ID); err != nil {
		t.Fatal(err)
	}
	u.Signals().CharacterRemoved.Emit(t.Context(), &app.EntityShort{ID: c.ID})
	assert.Equal(t, 0, got, "reloaded after the issuer was removed")
}

func TestContracts_ReloadsWhenCorporationsChange(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	newUI := func(t *testing.T) *testdouble.UIFake {
		return testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
	}
	// addCharacterToCorporation adds a character to a corporation without emitting signals.
	addCharacterToCorporation := func(corporationID int64) {
		x := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corporationID})
		factory.CreateCharacterFull(storage.CreateCharacterParams{ID: x.ID})
	}
	t.Run("characters", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		ecOwn := factory.CreateEveCorporation()
		ec := factory.CreateEveCorporation()
		x := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: ecOwn.ID})
		c := factory.CreateCharacterFull(storage.CreateCharacterParams{ID: x.ID})
		factory.CreateCharacterContract(storage.CreateCharacterContractParams{
			CharacterID:    c.ID,
			Status:         app.ContractStatusInProgress,
			AcceptorID:     ec.ID,
			DateAccepted:   optional.New(time.Now().AddDate(0, 0, -4)),
			DaysToComplete: optional.New[int64](3),
		})
		u := newUI(t)
		a := NewContractsForCharacters(u)
		var got int
		a.OnUpdate = func(attention int) {
			got = attention
		}
		a.update(t.Context())
		assert.Equal(t, 0, got, "accepted for another corporation")

		addCharacterToCorporation(ec.ID)
		u.Signals().CorporationsChanged.Emit(t.Context(), struct{}{})
		assert.Equal(t, 1, got, "reloaded after the acceptor became one of the user's corporations")
	})
	t.Run("corporation", func(t *testing.T) {
		testutil.MustTruncateTables(db)
		ec := factory.CreateEveCorporation()
		corporation := factory.CreateCorporation(ec.ID)
		if _, err := st.GetOrCreateEveEntity(t.Context(), storage.CreateEveEntityParams{
			ID:       ec.ID,
			Name:     ec.Name,
			Category: app.EveEntityCorporation,
		}); err != nil {
			t.Fatal(err)
		}
		member := factory.CreateEveEntityCharacter()
		factory.CreateCorporationContract(storage.CreateCorporationContractParams{
			CorporationID:  corporation.ID,
			Status:         app.ContractStatusOutstanding,
			DateExpired:    time.Now().Add(-time.Hour),
			ForCorporation: true,
			IssuerID:       member.ID,
		})
		u := newUI(t)
		a := NewContractsForCorporation(u)
		a.corporation.Store(corporation)
		var got int
		a.OnUpdate = func(attention int) {
			got = attention
		}
		a.update(t.Context())
		assert.Equal(t, 0, got, "issued for a corporation without the user's characters")

		addCharacterToCorporation(ec.ID)
		u.Signals().CorporationsChanged.Emit(t.Context(), struct{}{})
		assert.Equal(t, 1, got, "reloaded after the issuer became one of the user's corporations")
	})
}

func TestContracts_ErrorDisplay(t *testing.T) {
	newContracts := func(t *testing.T, st *storage.Storage) *Contracts {
		return NewContractsForCharacters(testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		}))
	}
	t.Run("shows error when update fails and keeps it on refresh and filtering", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := newContracts(t, st)
		now := time.Now()
		r := contractRow{contractID: 1, status: app.ContractStatusOutstanding, dateExpired: now.Add(time.Hour)}
		r.init(myID, noAcceptor, myIDs, myCorpIDs, now)
		a.rows = []contractRow{r}
		a.filterRowsAsync("")
		if _, err := db.Exec("DROP TABLE character_contracts"); err != nil { // let the next update fail
			t.Fatal(err)
		}

		a.update(t.Context())
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.errorLabel.Text, "ERROR:")

		a.refreshTimeDependent(now.Add(2 * time.Hour))
		assert.Equal(t, app.ContractCategoryRequiresAttention, a.rows[0].category, "refresh continues with last rows")
		a.selectStatus.SetSelected(contractsStatusAll)
		assert.True(t, a.errorLabel.Visible())
		assert.Contains(t, a.footerLabel.Text, "Showing")
	})
	t.Run("hides error after successful update", func(t *testing.T) {
		db, st, _ := testutil.NewDBOnDisk(t)
		defer db.Close()
		a := newContracts(t, st)
		a.errorLabel.SetText("ERROR: failed")
		a.errorLabel.Show()

		a.update(t.Context())
		assert.False(t, a.errorLabel.Visible())
	})
}
