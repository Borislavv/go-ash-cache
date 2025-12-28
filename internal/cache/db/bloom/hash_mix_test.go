package bloom

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// TestNextPow2_CalculatesCorrectly verifies nextPow2 calculates next power of two.
func TestNextPow2_CalculatesCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"zero", 0, 1},
		{"one", 1, 1},
		{"two", 2, 2},
		{"three", 3, 4},
		{"four", 4, 4},
		{"five", 5, 8},
		{"seven", 7, 8},
		{"eight", 8, 8},
		{"nine", 9, 16},
		{"fifteen", 15, 16},
		{"sixteen", 16, 16},
		{"large", 1000, 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nextPow2(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

// TestMix64_ProducesDifferentValues verifies mix64 produces diverse outputs.
func TestMix64_ProducesDifferentValues(t *testing.T) {
	values := make(map[uint64]bool)
	for i := uint64(0); i < 100; i++ {
		mixed := mix64(i)
		values[mixed] = true
	}

	// Should have high diversity
	require.Greater(t, len(values), 90, "mix64 should produce diverse values")
}

// TestMix64_Deterministic verifies mix64 is deterministic.
func TestMix64_Deterministic(t *testing.T) {
	input := uint64(12345)
	result1 := mix64(input)
	result2 := mix64(input)

	require.Equal(t, result1, result2, "mix64 should be deterministic")
}

// TestMix64_NonZero verifies mix64 produces non-zero for non-zero input.
func TestMix64_NonZero(t *testing.T) {
	result := mix64(1)
	require.NotEqual(t, uint64(0), result, "mix64 should produce non-zero for non-zero input")
}
