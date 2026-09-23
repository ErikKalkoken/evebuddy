package screens

import (
	"testing"

	"github.com/ErikKalkoken/go-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
)

func TestReduceSliceValues(t *testing.T) {
	t.Run("returns all entries unchanged if all shares are above minShare", func(t *testing.T) {
		input := []namedValue{
			{name: "A", value: 10},
			{name: "B", value: 20},
		}
		result := reduceSliceValues(input, 0.1, 0)

		assert.Len(t, result, 2)
		xassert.Equal(t, input, result)
	})

	t.Run("reduces and aggregates 'Others' correctly", func(t *testing.T) {
		// Total is 170. With minShare = 0.1 (10%), Banana (8.8%) and
		// Cherry (2.9%) fall below the threshold and are aggregated.
		input := []namedValue{
			{name: "Zebra", value: 100}, // 58.8%, kept
			{name: "Apple", value: 50},  // 29.4%, kept
			{name: "Banana", value: 15}, // 8.8%, reduced
			{name: "Cherry", value: 5},  // 2.9%, reduced
		}

		result := reduceSliceValues(input, 0.1, 0)

		// Result should have 2 kept entries + 1 "Others" entry
		require.Len(t, result, 3)

		// Check 'Others' aggregation
		// 'Others' is appended last
		others := result[2]
		xassert.Equal(t, "Others", others.name)
		xassert.Equal(t, float64(20), others.value)

		// Check alphabetical sorting of the remaining kept items
		xassert.Equal(t, "Apple", result[0].name)
		xassert.Equal(t, "Zebra", result[1].name)
	})

	t.Run("aggregates everything into 'Others' if no share reaches minShare and there is no floor", func(t *testing.T) {
		input := []namedValue{
			{name: "A", value: 10},
			{name: "B", value: 20},
		}
		// Neither share (33%/67%) reaches 90%, so both become 'Others'
		result := reduceSliceValues(input, 0.9, 0)

		assert.Len(t, result, 1)
		xassert.Equal(t, "Others", result[0].name)
		xassert.Equal(t, float64(30), result[0].value)
	})

	t.Run("keeps the top minCount entries even if no share reaches minShare", func(t *testing.T) {
		// Total is 150. With minShare = 0.5 (50%), no single entry reaches
		// the threshold on its own, so without a floor everything would
		// collapse into a single 100% 'Others' slice.
		input := []namedValue{
			{name: "A", value: 50},
			{name: "B", value: 40},
			{name: "C", value: 30},
			{name: "D", value: 20},
			{name: "E", value: 10},
		}

		result := reduceSliceValues(input, 0.5, 3)

		// The top 3 by value (A, B, C) are kept regardless of share;
		// D and E (20+10=30) are aggregated into 'Others'.
		require.Len(t, result, 4)
		xassert.Equal(t, "A", result[0].name)
		xassert.Equal(t, "B", result[1].name)
		xassert.Equal(t, "C", result[2].name)
		xassert.Equal(t, "Others", result[3].name)
		xassert.Equal(t, float64(30), result[3].value)
	})
}

func TestReduceAssetWalletValues(t *testing.T) {
	tests := []struct {
		name     string
		data     []characterWealthValue
		m        int
		expected []characterWealthValue
	}{
		{
			name: "No reduction needed",
			data: []characterWealthValue{
				{name: "A", assets: 10, wallet: 1},
				{name: "B", assets: 20, wallet: 2},
			},
			m: 5,
			expected: []characterWealthValue{
				{name: "A", assets: 10, wallet: 1},
				{name: "B", assets: 20, wallet: 2},
			},
		},
		{
			name: "Reduces to top M by combined value and aggregates others",
			data: []characterWealthValue{
				{name: "Banana", assets: 8, wallet: 2, contracts: 1},            // combined 11, Top 2
				{name: "Apple", assets: 40, wallet: 10, orders: 5},              // combined 55, Top 1
				{name: "Cherry", assets: 4, wallet: 1, contracts: 1, orders: 1}, // combined 7, Other
				{name: "Date", assets: 1, wallet: 1},                            // combined 2, Other
			},
			m: 2,
			expected: []characterWealthValue{
				{name: "Apple", assets: 40, wallet: 10, orders: 5},              // Sorted alphabetically
				{name: "Banana", assets: 8, wallet: 2, contracts: 1},            // Sorted alphabetically
				{name: "Others", assets: 5, wallet: 2, contracts: 1, orders: 1}, // (4+1), (1+1), (1+0), (0+1)
			},
		},
		{
			name: "M is zero",
			data: []characterWealthValue{
				{name: "A", assets: 10, wallet: 1},
				{name: "B", assets: 20, wallet: 2},
			},
			m: 0,
			expected: []characterWealthValue{
				{name: "Others", assets: 30, wallet: 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We pass a copy to avoid mutating the test case slice if reused
			input := make([]characterWealthValue, len(tt.data))
			copy(input, tt.data)

			actual := reduceCharacterWealthValues(input, tt.m)

			xassert.Equal(t, tt.expected, actual)
		})
	}
}

func TestNiceStep(t *testing.T) {
	cases := []struct {
		value    float64
		expected float64
	}{
		{1, 1},
		{1.5, 2},
		{4, 5},
		{9, 10},
		{12, 20},
		{20, 20},
	}
	for _, tt := range cases {
		xassert.Equal(t, tt.expected, niceStep(tt.value))
	}
}

func TestNiceAxisBounds(t *testing.T) {
	cases := []struct {
		value             float64
		expectedAxisMax   float64
		expectedTickCount int
	}{
		{0, 1, 2},
		{-5, 1, 2},
		{9, 10, 6},
		{12, 15, 4},
		{37.4, 40, 5},
		// Regression: a max of 74 used to round up to 100 with ticks by 20,
		// leaving 80-100 empty. It should now round to 80 instead.
		{74, 80, 5},
		{100, 100, 6},
		{101, 150, 4},
	}
	for _, tt := range cases {
		axisMax, tickCount := niceAxisBounds(tt.value, 5)
		xassert.Equal(t, tt.expectedAxisMax, axisMax)
		xassert.Equal(t, tt.expectedTickCount, tickCount)
	}
}

func TestNewWealthDetailsRow(t *testing.T) {
	t.Run("converts row with all values", func(t *testing.T) {
		r := newWealthDetailsRow(characterWealthRow{
			characterID:     42,
			characterName:   "Bruce Wayne",
			combinedAssets:  optional.New(1_000_000.0),
			contractsEscrow: optional.New(2_000.0),
			ordersEscrow:    optional.New(3_000.0),
			skillPoints:     optional.New(4_000.0),
			tags:            set.Of("Zeta", "Alpha"),
			total:           optional.New(1_005_000.0),
			walletBalance:   optional.New(0.0),
		})
		xassert.Equal(t, int64(42), r.characterID)
		xassert.Equal(t, "Bruce Wayne", r.characterName)
		xassert.Equal(t, "bruce wayne", r.searchTarget)
		xassert.Equal(t, "Alpha, Zeta", r.tagsDisplay)
		xassert.Equal(t, "1,000,000", r.combinedAssetsDisplay)
		xassert.Equal(t, "2,000", r.contractsEscrowDisplay)
		xassert.Equal(t, "3,000", r.ordersEscrowDisplay)
		xassert.Equal(t, "4,000", r.skillPointsDisplay)
		xassert.Equal(t, "1,005,000", r.totalNetWorthDisplay)
		xassert.Equal(t, "0", r.walletDisplay)
		xassert.Equal(t, optional.New(1_000_000.0), r.combinedAssetsValue)
		assert.False(t, r.isTotal)
	})
	t.Run("shows missing values as ?", func(t *testing.T) {
		r := newWealthDetailsRow(characterWealthRow{
			characterName: "Bruce Wayne",
			tags:          set.Of[string](),
		})
		xassert.Equal(t, "", r.tagsDisplay)
		xassert.Equal(t, "?", r.combinedAssetsDisplay)
		xassert.Equal(t, "?", r.totalNetWorthDisplay)
		xassert.Equal(t, "?", r.walletDisplay)
		assert.True(t, r.totalNetWorth.IsEmpty())
	})
}

func TestFilterWealthRows(t *testing.T) {
	corpA := &app.EveEntity{ID: 1, Name: "Corp A", Category: app.EveEntityCorporation}
	corpB := &app.EveEntity{ID: 2, Name: "Corp B", Category: app.EveEntityCorporation}
	allianceX := &app.EveEntity{ID: 3, Name: "Alliance X", Category: app.EveEntityAlliance}
	rows := []characterWealthRow{
		{characterName: "Alpha", corporation: corpA, alliance: optional.New(allianceX), tags: set.Of("Main")},
		{characterName: "Bravo", corporation: corpA, alliance: optional.New(allianceX), tags: set.Of("Alt")},
		{characterName: "Charlie", corporation: corpB, tags: set.Of("Alt", "Industry")},
	}
	names := func(rows []characterWealthRow) []string {
		return xslices.Map(rows, func(r characterWealthRow) string {
			return r.characterName
		})
	}
	cases := []struct {
		name        string
		tag         string
		corporation string
		alliance    string
		want        []string
	}{
		{"no filter", "", "", "", []string{"Alpha", "Bravo", "Charlie"}},
		{"tag", "Alt", "", "", []string{"Bravo", "Charlie"}},
		{"corporation", "", "Corp B", "", []string{"Charlie"}},
		{"alliance", "", "", "Alliance X", []string{"Alpha", "Bravo"}},
		{"combined", "Alt", "Corp A", "", []string{"Bravo"}},
		{"no match", "Main", "Corp B", "", []string{}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := filterWealthRows(rows, tt.tag, tt.corporation, tt.alliance)
			xassert.Equal(t, tt.want, names(got))
		})
	}
	t.Run("does not modify input", func(t *testing.T) {
		filterWealthRows(rows, "Main", "", "")
		xassert.Equal(t, []string{"Alpha", "Bravo", "Charlie"}, names(rows))
	})
}

func TestWealthFilterOptions(t *testing.T) {
	corpA := &app.EveEntity{ID: 1, Name: "Corp A", Category: app.EveEntityCorporation}
	corpB := &app.EveEntity{ID: 2, Name: "Corp B", Category: app.EveEntityCorporation}
	allianceX := &app.EveEntity{ID: 3, Name: "Alliance X", Category: app.EveEntityAlliance}
	rows := []characterWealthRow{
		{corporation: corpB, alliance: optional.New(allianceX), tags: set.Of("Main")},
		{corporation: corpA, alliance: optional.New(allianceX), tags: set.Of("Alt")},
		{corporation: corpA, tags: set.Of("Alt", "Industry")},
	}
	tags, corporations, alliances := wealthFilterOptions(rows)
	xassert.Equal(t, []string{"Alt", "Industry", "Main"}, tags)
	xassert.Equal(t, []string{"Corp A", "Corp B"}, corporations)
	xassert.Equal(t, []string{"Alliance X"}, alliances)
}

func TestWealthEmptyText(t *testing.T) {
	cases := []struct {
		name          string
		isFiltered    bool
		filteredCount int
		chartCount    int
		want          string
	}{
		{"has data", false, 3, 2, ""},
		{"has filtered data", true, 1, 1, ""},
		{"no characters", false, 0, 0, "No characters"},
		{"no filter match", true, 0, 0, "No characters match the filter"},
		{"no wealth data", false, 2, 0, "No wealth data yet"},
		{"no wealth data for filtered", true, 2, 0, "No wealth data yet"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			xassert.Equal(t, tt.want, wealthEmptyText(tt.isFiltered, tt.filteredCount, tt.chartCount))
		})
	}
}
