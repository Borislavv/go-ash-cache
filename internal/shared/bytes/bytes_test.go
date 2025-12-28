package bytes

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// TestIsBytesAreEquals_Equal verifies that equal byte slices are correctly identified.
func TestIsBytesAreEquals_Equal(t *testing.T) {
	a := []byte("test data")
	b := []byte("test data")

	require.True(t, IsBytesAreEquals(a, b))
}

// TestIsBytesAreEquals_NotEqual verifies that different byte slices are correctly identified.
func TestIsBytesAreEquals_NotEqual(t *testing.T) {
	a := []byte("test data")
	b := []byte("different data")

	require.False(t, IsBytesAreEquals(a, b))
}

// TestIsBytesAreEquals_DifferentLength verifies that slices of different lengths are not equal.
func TestIsBytesAreEquals_DifferentLength(t *testing.T) {
	a := []byte("short")
	b := []byte("much longer data")

	require.False(t, IsBytesAreEquals(a, b))
}

// TestIsBytesAreEquals_LargeSlices verifies hash-based comparison for large slices.
func TestIsBytesAreEquals_LargeSlices(t *testing.T) {
	// Create large slices (> 32 bytes to trigger hash comparison)
	a := make([]byte, 100)
	b := make([]byte, 100)
	for i := range a {
		a[i] = byte(i % 256)
		b[i] = byte(i % 256)
	}

	require.True(t, IsBytesAreEquals(a, b))

	// Modify one byte
	b[50] = 255
	require.False(t, IsBytesAreEquals(a, b))
}

// TestFmtMem_FormatsCorrectly verifies memory formatting for different sizes.
func TestFmtMem_FormatsCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{"bytes", 512, "512B"},
		{"kilobytes", 5 * 1024, "5KB 0B"},
		{"megabytes", 10 * 1024 * 1024, "10MB 0KB"},
		{"gigabytes", 2 * 1024 * 1024 * 1024, "2GB 0MB"},
		{"terabytes", 1 * 1024 * 1024 * 1024 * 1024, "1TB 0GB"},
		{"mixed KB", 1536, "1KB 512B"},
		{"mixed MB", 10*1024*1024 + 512*1024, "10MB 512KB"},
		{"mixed GB", 2*1024*1024*1024 + 100*1024*1024, "2GB 100MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FmtMem(tt.bytes)
			require.Equal(t, tt.expected, result)
		})
	}
}
