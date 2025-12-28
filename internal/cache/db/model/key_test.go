package model

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// TestNewKey creates a key from string.
func TestNewKey(t *testing.T) {
	key := NewKey("test key")
	require.NotNil(t, key)
	require.Greater(t, key.Value(), uint64(0), "key value should be non-zero")
}

// TestKey_Value returns the hash value.
func TestKey_Value(t *testing.T) {
	key1 := NewKey("test")
	key2 := NewKey("test")

	// Same string should produce same hash
	require.Equal(t, key1.Value(), key2.Value())
}

// TestKey_IsTheSame verifies key comparison.
func TestKey_IsTheSame(t *testing.T) {
	key1 := NewKey("test")
	key2 := NewKey("test")
	key3 := NewKey("different")

	require.True(t, key1.IsTheSame(key2), "same strings should produce same keys")
	require.False(t, key1.IsTheSame(key3), "different strings should produce different keys")
}

// TestKey_IsTheSame_Self returns true for same key.
func TestKey_IsTheSame_Self(t *testing.T) {
	key := NewKey("test")
	require.True(t, key.IsTheSame(key))
}
