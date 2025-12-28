package queue

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// TestQueue_Init verifies queue initialization.
func TestQueue_Init(t *testing.T) {
	var q Queue
	q.Init(10)

	require.NotNil(t, q.buf)
	require.Equal(t, 10, len(q.buf))
	require.Equal(t, 0, q.head)
	require.Equal(t, 0, q.tail)
}

// TestQueue_Init_MinSize verifies that Init enforces minimum size.
func TestQueue_Init_MinSize(t *testing.T) {
	var q Queue
	q.Init(1) // Should be rounded up to 2

	require.GreaterOrEqual(t, len(q.buf), 2)
}

// TestQueue_TryPushTryPop verifies basic push/pop operations.
func TestQueue_TryPushTryPop(t *testing.T) {
	var q Queue
	q.Init(10)

	// Push values
	require.True(t, q.TryPush(1))
	require.True(t, q.TryPush(2))
	require.True(t, q.TryPush(3))

	// Pop values
	val, ok := q.TryPop()
	require.True(t, ok)
	require.Equal(t, uint64(1), val)

	val, ok = q.TryPop()
	require.True(t, ok)
	require.Equal(t, uint64(2), val)

	val, ok = q.TryPop()
	require.True(t, ok)
	require.Equal(t, uint64(3), val)

	// Queue should be empty
	_, ok = q.TryPop()
	require.False(t, ok)
}

// TestQueue_Full verifies that TryPush returns false when queue is full.
func TestQueue_Full(t *testing.T) {
	var q Queue
	q.Init(3) // Can hold 2 elements (head+1 == tail means full)

	require.True(t, q.TryPush(1))
	require.True(t, q.TryPush(2))
	require.False(t, q.TryPush(3)) // Queue is full, can't push more
}

// TestQueue_Empty verifies that TryPop returns false when queue is empty.
func TestQueue_Empty(t *testing.T) {
	var q Queue
	q.Init(10)

	_, ok := q.TryPop()
	require.False(t, ok)
}

// TestQueue_WrapAround verifies circular buffer behavior.
func TestQueue_WrapAround(t *testing.T) {
	var q Queue
	q.Init(4) // Can hold 3 elements

	// Fill and empty partially to test wrap-around
	require.True(t, q.TryPush(1))
	require.True(t, q.TryPush(2))
	val, _ := q.TryPop() // Pop 1
	require.Equal(t, uint64(1), val)

	require.True(t, q.TryPush(3))
	require.True(t, q.TryPush(4))

	// Should get 2, 3, 4 in order
	val, _ = q.TryPop()
	require.Equal(t, uint64(2), val)
	val, _ = q.TryPop()
	require.Equal(t, uint64(3), val)
	val, _ = q.TryPop()
	require.Equal(t, uint64(4), val)
}
