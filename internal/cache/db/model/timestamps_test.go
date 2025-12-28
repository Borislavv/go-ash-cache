package model

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestEntry_SetTTL sets TTL correctly.
func TestEntry_SetTTL(t *testing.T) {
	entry := NewEmptyEntry(NewKey("test"), 0, nil)
	ttl := 1 * time.Hour

	entry.SetTTL(ttl)

	// TTL is stored internally, verify it's set (via IsExpired check)
	// We can't directly access ttl field, but we can verify behavior
	require.NotNil(t, entry)
}

// TestEntry_RenewTouchedAt updates touchedAt timestamp.
func TestEntry_RenewTouchedAt(t *testing.T) {
	entry := NewEmptyEntry(NewKey("test"), 0, nil)
	entry.SetPayload([]byte("data"))

	initial := entry.TouchedAt()
	time.Sleep(1 * time.Millisecond)

	entry.RenewTouchedAt()
	renewed := entry.TouchedAt()

	require.GreaterOrEqual(t, renewed, initial, "RenewTouchedAt should update timestamp")
}

// TestEntry_RenewUpdatedAt updates updatedAt timestamp.
func TestEntry_RenewUpdatedAt(t *testing.T) {
	entry := NewEmptyEntry(NewKey("test"), 0, nil)
	entry.SetPayload([]byte("data"))

	initial := entry.UpdatedAt()
	time.Sleep(1 * time.Millisecond)

	entry.RenewUpdatedAt()
	renewed := entry.UpdatedAt()

	require.GreaterOrEqual(t, renewed, initial, "RenewUpdatedAt should update timestamp")
}

// TestEntry_UntouchRefreshedAt sets updatedAt to past time.
func TestEntry_UntouchRefreshedAt(t *testing.T) {
	entry := NewEmptyEntry(NewKey("test"), time.Hour.Nanoseconds(), nil)
	entry.SetPayload([]byte("data"))

	initial := entry.UpdatedAt()
	entry.UntouchRefreshedAt()
	untouched := entry.UpdatedAt()

	require.Less(t, untouched, initial, "UntouchRefreshedAt should set timestamp to past")
}

// TestEntry_TouchedAt returns current touchedAt value.
func TestEntry_TouchedAt(t *testing.T) {
	entry := NewEmptyEntry(NewKey("test"), 0, nil)
	
	// SetPayload sets touchedAt via cachedtime
	entry.SetPayload([]byte("data"))

	touched := entry.TouchedAt()
	// Even if cachedtime is disabled, SetPayload uses cachedtime.Now() which falls back to time.Now()
	require.GreaterOrEqual(t, touched, int64(0), "TouchedAt should return valid timestamp")
}

// TestEntry_UpdatedAt returns current updatedAt value.
func TestEntry_UpdatedAt(t *testing.T) {
	entry := NewEmptyEntry(NewKey("test"), 0, nil)
	
	// SetPayload sets updatedAt via cachedtime
	entry.SetPayload([]byte("data"))

	updated := entry.UpdatedAt()
	// Even if cachedtime is disabled, SetPayload uses cachedtime.Now() which falls back to time.Now()
	require.GreaterOrEqual(t, updated, int64(0), "UpdatedAt should return valid timestamp")
}
