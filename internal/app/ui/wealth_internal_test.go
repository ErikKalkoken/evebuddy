package ui

import (
	"testing"

	chartData "github.com/s-daehling/fyne-charts/pkg/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReduceChartPoints(t *testing.T) {
	t.Run("returns original slice if length <= m", func(t *testing.T) {
		input := []chartData.ProportionalPoint{
			{C: "A", Val: 10},
			{C: "B", Val: 20},
		}
		result := reduceProportionalPoints(input, 5)

		assert.Len(t, result, 2)
		assert.Equal(t, input, result)
	})

	t.Run("reduces and aggregates 'Others' correctly", func(t *testing.T) {
		// We provide 4 points, m = 2.
		// Top 2 should stay, bottom 2 should sum to "Others" (15 + 5 = 20)
		input := []chartData.ProportionalPoint{
			{C: "Zebra", Val: 100}, // Top 1
			{C: "Apple", Val: 50},  // Top 2
			{C: "Banana", Val: 15}, // Should be reduced
			{C: "Cherry", Val: 5},  // Should be reduced
		}
		m := 2

		result := reduceProportionalPoints(input, m)

		// Result should have m + 1 (Others) = 3 elements
		require.Len(t, result, 3)

		// Check 'Others' aggregation
		// 'Others' is appended last
		others := result[2]
		assert.Equal(t, "Others", others.C)
		assert.Equal(t, float64(20), others.Val)

		// Check alphabetical sorting of the remaining top items
		// "Apple" (50) and "Zebra" (100) are top 2.
		// Alphabetically, Apple comes before Zebra.
		assert.Equal(t, "Apple", result[0].C)
		assert.Equal(t, "Zebra", result[1].C)
	})

	t.Run("handles m=0", func(t *testing.T) {
		input := []chartData.ProportionalPoint{
			{C: "A", Val: 10},
			{C: "B", Val: 20},
		}
		// If m=0, all elements become 'Others'
		result := reduceProportionalPoints(input, 0)

		assert.Len(t, result, 1)
		assert.Equal(t, "Others", result[0].C)
		assert.Equal(t, float64(30), result[0].Val)
	})
}

func TestReduceCategoricalPoints(t *testing.T) {
	tests := []struct {
		name     string
		data     []chartData.CategoricalPoint
		m        int
		expected []chartData.CategoricalPoint
	}{
		{
			name: "No reduction needed",
			data: []chartData.CategoricalPoint{
				{C: "A", Val: 10},
				{C: "B", Val: 20},
			},
			m: 5,
			expected: []chartData.CategoricalPoint{
				{C: "A", Val: 10},
				{C: "B", Val: 20},
			},
		},
		{
			name: "Reduces to top M and aggregates others",
			data: []chartData.CategoricalPoint{
				{C: "Banana", Val: 10}, // Top 2
				{C: "Apple", Val: 50},  // Top 1
				{C: "Cherry", Val: 5},  // Other
				{C: "Date", Val: 2},    // Other
			},
			m: 2,
			expected: []chartData.CategoricalPoint{
				{C: "Apple", Val: 50},  // Sorted alphabetically
				{C: "Banana", Val: 10}, // Sorted alphabetically
				{C: "Others", Val: 7},  // 5 + 2
			},
		},
		{
			name: "M is zero",
			data: []chartData.CategoricalPoint{
				{C: "A", Val: 10},
				{C: "B", Val: 20},
			},
			m: 0,
			expected: []chartData.CategoricalPoint{
				{C: "Others", Val: 30},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We pass a copy to avoid mutating the test case slice if reused
			input := make([]chartData.CategoricalPoint, len(tt.data))
			copy(input, tt.data)

			actual := reduceCategoricalPoints(input, tt.m)

			assert.Equal(t, tt.expected, actual, "The reduced slice should match expected output")
		})
	}
}

func TestTruncateWithSuffix(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		limit     int
		suffixLen int
		expected  string
	}{
		{
			name:      "Standard truncation",
			input:     "november",
			limit:     7,
			suffixLen: 1,
			expected:  "novem...r",
		},
		{
			name:      "Trailing space removal",
			input:     "november ",
			limit:     8,
			suffixLen: 1,
			expected:  "november",
		},
		{
			name:      "Suffix ends in space",
			input:     "open space ",
			limit:     9,
			suffixLen: 2,             // "e "
			expected:  "open s...ce", // Space trimmed
		},
		{
			name:      "String within limit",
			input:     "hello",
			limit:     10,
			suffixLen: 2,
			expected:  "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateWithSuffix(tt.input, tt.limit, tt.suffixLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}
