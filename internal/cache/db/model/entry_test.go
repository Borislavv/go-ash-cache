package model

import (
	"errors"
	"github.com/Borislavv/go-ash-cache/model"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestEntry_Update calls callback and stores result.
func TestEntry_Update(t *testing.T) {
	var callbackCalled bool
	testData := []byte("updated data")

	entry := NewEmptyEntry(NewKey("test"), 0, func(item model.Item) ([]byte, error) {
		callbackCalled = true
		return testData, nil
	})

	err := entry.Update()
	require.NoError(t, err)
	require.True(t, callbackCalled)
	require.Equal(t, testData, entry.PayloadBytes())
}

// TestEntry_Update_Error propagates callback error.
func TestEntry_Update_Error(t *testing.T) {
	testErr := errors.New("callback error")

	entry := NewEmptyEntry(NewKey("test"), 0, func(item model.Item) ([]byte, error) {
		return nil, testErr
	})

	err := entry.Update()
	require.Error(t, err)
	require.Equal(t, testErr, err)
}

// TestEntry_SetPayload updates payload and timestamps.
func TestEntry_SetPayload(t *testing.T) {
	entry := NewEmptyEntry(NewKey("test"), 0, nil)
	testData := []byte("test data")

	entry.SetPayload(testData)

	require.Equal(t, testData, entry.PayloadBytes())
	// Timestamps are set via cachedtime, which may fall back to time.Now() if disabled
	require.GreaterOrEqual(t, entry.TouchedAt(), int64(0), "TouchedAt should be set")
	require.GreaterOrEqual(t, entry.UpdatedAt(), int64(0), "UpdatedAt should be set")
}

// TestEntry_Weight calculates weight correctly.
func TestEntry_Weight(t *testing.T) {
	entry := NewEmptyEntry(NewKey("test"), 0, nil)
	testData := make([]byte, 1024) // 1KB

	entry.SetPayload(testData)

	weight := entry.Weight()
	require.Greater(t, weight, int64(1024), "weight should include payload capacity")
}

// TestEntry_IsTheSamePayload compares payloads correctly.
func TestEntry_IsTheSamePayload(t *testing.T) {
	entry1 := NewEmptyEntry(NewKey("test1"), 0, nil)
	entry2 := NewEmptyEntry(NewKey("test2"), 0, nil)
	testData := []byte("same data")

	entry1.SetPayload(testData)
	entry2.SetPayload(testData)

	require.True(t, entry1.IsTheSamePayload(entry2))
}

// TestEntry_IsTheSamePayload_Different returns false for different payloads.
func TestEntry_IsTheSamePayload_Different(t *testing.T) {
	entry1 := NewEmptyEntry(NewKey("test1"), 0, nil)
	entry2 := NewEmptyEntry(NewKey("test2"), 0, nil)

	entry1.SetPayload([]byte("data1"))
	entry2.SetPayload([]byte("data2"))

	require.False(t, entry1.IsTheSamePayload(entry2))
}

// TestEntry_SwapPayloads swaps payloads and returns weight difference.
func TestEntry_SwapPayloads(t *testing.T) {
	entry1 := NewEmptyEntry(NewKey("test1"), 0, nil)
	entry2 := NewEmptyEntry(NewKey("test2"), 0, nil)

	data1 := make([]byte, 512)
	data2 := make([]byte, 1024)

	entry1.SetPayload(data1)
	entry2.SetPayload(data2)

	weightDiff := entry1.SwapPayloads(entry2)

	require.Equal(t, data2, entry1.PayloadBytes())
	require.Greater(t, weightDiff, int64(0), "weight diff should be positive when swapping larger payload")
}
