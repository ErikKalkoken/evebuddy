package screens

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/evebuddy/internal/xassert"
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
		data     []assetWalletValue
		m        int
		expected []assetWalletValue
	}{
		{
			name: "No reduction needed",
			data: []assetWalletValue{
				{name: "A", assets: 10, wallet: 1},
				{name: "B", assets: 20, wallet: 2},
			},
			m: 5,
			expected: []assetWalletValue{
				{name: "A", assets: 10, wallet: 1},
				{name: "B", assets: 20, wallet: 2},
			},
		},
		{
			name: "Reduces to top M by combined value and aggregates others",
			data: []assetWalletValue{
				{name: "Banana", assets: 8, wallet: 2, contracts: 1},            // combined 11, Top 2
				{name: "Apple", assets: 40, wallet: 10, orders: 5},              // combined 55, Top 1
				{name: "Cherry", assets: 4, wallet: 1, contracts: 1, orders: 1}, // combined 7, Other
				{name: "Date", assets: 1, wallet: 1},                            // combined 2, Other
			},
			m: 2,
			expected: []assetWalletValue{
				{name: "Apple", assets: 40, wallet: 10, orders: 5},              // Sorted alphabetically
				{name: "Banana", assets: 8, wallet: 2, contracts: 1},            // Sorted alphabetically
				{name: "Others", assets: 5, wallet: 2, contracts: 1, orders: 1}, // (4+1), (1+1), (1+0), (0+1)
			},
		},
		{
			name: "M is zero",
			data: []assetWalletValue{
				{name: "A", assets: 10, wallet: 1},
				{name: "B", assets: 20, wallet: 2},
			},
			m: 0,
			expected: []assetWalletValue{
				{name: "Others", assets: 30, wallet: 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We pass a copy to avoid mutating the test case slice if reused
			input := make([]assetWalletValue, len(tt.data))
			copy(input, tt.data)

			actual := reduceAssetWalletValues(input, tt.m)

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
