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
		result := reduceSliceValues(input, 0.1)

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

		result := reduceSliceValues(input, 0.1)

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

	t.Run("aggregates everything into 'Others' if no share reaches minShare", func(t *testing.T) {
		input := []namedValue{
			{name: "A", value: 10},
			{name: "B", value: 20},
		}
		// Neither share (33%/67%) reaches 90%, so both become 'Others'
		result := reduceSliceValues(input, 0.9)

		assert.Len(t, result, 1)
		xassert.Equal(t, "Others", result[0].name)
		xassert.Equal(t, float64(30), result[0].value)
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
			name: "Reduces to top M by combined assets+wallet and aggregates others",
			data: []assetWalletValue{
				{name: "Banana", assets: 8, wallet: 2},  // combined 10, Top 2
				{name: "Apple", assets: 40, wallet: 10}, // combined 50, Top 1
				{name: "Cherry", assets: 4, wallet: 1},  // combined 5, Other
				{name: "Date", assets: 1, wallet: 1},    // combined 2, Other
			},
			m: 2,
			expected: []assetWalletValue{
				{name: "Apple", assets: 40, wallet: 10}, // Sorted alphabetically
				{name: "Banana", assets: 8, wallet: 2},  // Sorted alphabetically
				{name: "Others", assets: 5, wallet: 2},  // (4+1), (1+1)
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

func TestNiceCeil(t *testing.T) {
	cases := []struct {
		value    float64
		expected float64
	}{
		{0, 1},
		{-5, 1},
		{1, 1},
		{1.5, 2},
		{4, 5},
		{9, 10},
		{12, 20},
		{37.4, 50},
		{100, 100},
		{101, 200},
	}
	for _, tt := range cases {
		xassert.Equal(t, tt.expected, niceCeil(tt.value))
	}
}
