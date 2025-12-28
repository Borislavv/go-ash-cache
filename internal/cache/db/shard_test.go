package db

import (
	"context"
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

// TestShard_Set_Insert verifies Set inserts new entries correctly.
func TestShard_Set_Insert(t *testing.T) {
	sh := NewShard(0)
	key := uint64(123)
	entry := model.NewEmptyEntry(model.NewKey("test"), 0, nil)
	entry.SetPayload([]byte("data"))

	bytesDelta, lenDelta := sh.Set(key, entry)

	require.Equal(t, int64(1), lenDelta, "should increment length")
	require.Greater(t, bytesDelta, int64(0), "should add memory")
	require.Equal(t, int64(1), sh.Len())
	require.Greater(t, sh.Weight(), int64(0))
}

// TestShard_Set_Update verifies Set updates existing entries.
func TestShard_Set_Update(t *testing.T) {
	sh := NewShard(0)
	key := uint64(123)
	entry1 := model.NewEmptyEntry(model.NewKey("test1"), 0, nil)
	entry1.SetPayload([]byte("small"))
	entry2 := model.NewEmptyEntry(model.NewKey("test2"), 0, nil)
	entry2.SetPayload(make([]byte, 1024)) // larger payload

	sh.Set(key, entry1)
	bytesDelta, lenDelta := sh.Set(key, entry2)

	require.Equal(t, int64(0), lenDelta, "should not change length on update")
	require.Greater(t, bytesDelta, int64(0), "should reflect memory increase")
	require.Equal(t, int64(1), sh.Len())
}

// TestShard_Get_Exists returns entry when key exists.
func TestShard_Get_Exists(t *testing.T) {
	sh := NewShard(0)
	key := uint64(123)
	entry := model.NewEmptyEntry(model.NewKey("test"), 0, nil)
	entry.SetPayload([]byte("data"))

	sh.Set(key, entry)
	retrieved, found := sh.Get(key)

	require.True(t, found)
	require.Equal(t, entry.PayloadBytes(), retrieved.PayloadBytes())
}

// TestShard_Get_NotExists returns false when key doesn't exist.
func TestShard_Get_NotExists(t *testing.T) {
	sh := NewShard(0)
	retrieved, found := sh.Get(999)

	require.False(t, found)
	require.Nil(t, retrieved)
}

// TestShard_Remove_Exists removes entry and returns freed bytes.
func TestShard_Remove_Exists(t *testing.T) {
	sh := NewShard(0)
	key := uint64(123)
	entry := model.NewEmptyEntry(model.NewKey("test"), 0, nil)
	entry.SetPayload([]byte("data"))

	sh.Set(key, entry)
	initialMem := sh.Weight()
	freedBytes, hit := sh.Remove(key)

	require.True(t, hit)
	require.Equal(t, initialMem, freedBytes)
	require.Equal(t, int64(0), sh.Len())
	require.Equal(t, int64(0), sh.Weight())
}

// TestShard_Remove_NotExists returns false when key doesn't exist.
func TestShard_Remove_NotExists(t *testing.T) {
	sh := NewShard(0)
	freedBytes, hit := sh.Remove(999)

	require.False(t, hit)
	require.Equal(t, int64(0), freedBytes)
}

// TestShard_Clear_RemovesAllEntries clears all entries and returns totals.
func TestShard_Clear_RemovesAllEntries(t *testing.T) {
	sh := NewShard(0)
	for i := 0; i < 10; i++ {
		entry := model.NewEmptyEntry(model.NewKey("test"), 0, nil)
		entry.SetPayload([]byte("data"))
		sh.Set(uint64(i), entry)
	}

	freedBytes, items := sh.Clear()

	require.Equal(t, int64(10), items)
	require.Greater(t, freedBytes, int64(0))
	require.Equal(t, int64(0), sh.Len())
	require.Equal(t, int64(0), sh.Weight())
}

// TestShard_ConcurrentSetGet verifies thread-safety of Set and Get.
func TestShard_ConcurrentSetGet(t *testing.T) {
	sh := NewShard(0)
	const numGoroutines = 10
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := uint64(id*opsPerGoroutine + j)
				entry := model.NewEmptyEntry(model.NewKey("test"), 0, nil)
				entry.SetPayload([]byte("data"))
				sh.Set(key, entry)
			}
		}(i)
	}

	// Readers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				key := uint64(id*opsPerGoroutine + j)
				_, _ = sh.Get(key)
			}
		}(i)
	}

	wg.Wait()
	require.Equal(t, int64(numGoroutines*opsPerGoroutine), sh.Len())
}

// TestShard_WalkR_IteratesAllEntries iterates all entries under read lock.
func TestShard_WalkR_IteratesAllEntries(t *testing.T) {
	sh := NewShard(0)
	expected := make(map[uint64]bool)
	for i := 0; i < 10; i++ {
		key := uint64(i)
		entry := model.NewEmptyEntry(model.NewKey("test"), 0, nil)
		entry.SetPayload([]byte("data"))
		sh.Set(key, entry)
		expected[key] = true
	}

	seen := make(map[uint64]bool)
	ctx := context.Background()
	sh.WalkR(ctx, func(k uint64, v *model.Entry) bool {
		seen[k] = true
		return true
	})

	require.Equal(t, len(expected), len(seen))
	for k := range expected {
		require.True(t, seen[k], "key %d should be seen", k)
	}
}

// TestShard_WalkR_RespectsContextCancel stops iteration on context cancel.
func TestShard_WalkR_RespectsContextCancel(t *testing.T) {
	sh := NewShard(0)
	for i := 0; i < 100; i++ {
		entry := model.NewEmptyEntry(model.NewKey("test"), 0, nil)
		entry.SetPayload([]byte("data"))
		sh.Set(uint64(i), entry)
	}

	ctx, cancel := context.WithCancel(context.Background())
	seen := 0
	sh.WalkR(ctx, func(k uint64, v *model.Entry) bool {
		seen++
		if seen == 5 {
			cancel()
		}
		return true
	})

	require.LessOrEqual(t, seen, 10, "should stop iteration after cancel")
}

// TestShard_WalkR_StopsOnFalse stops iteration when callback returns false.
func TestShard_WalkR_StopsOnFalse(t *testing.T) {
	sh := NewShard(0)
	for i := 0; i < 10; i++ {
		entry := model.NewEmptyEntry(model.NewKey("test"), 0, nil)
		entry.SetPayload([]byte("data"))
		sh.Set(uint64(i), entry)
	}

	seen := 0
	ctx := context.Background()
	sh.WalkR(ctx, func(k uint64, v *model.Entry) bool {
		seen++
		return seen < 3 // Stop after 3
	})

	require.Equal(t, 3, seen)
}

// TestShard_EnqueueRefresh_DequeueExpired manages refresh queue correctly.
func TestShard_EnqueueRefresh_DequeueExpired(t *testing.T) {
	sh := NewShard(0)
	key := uint64(123)

	require.True(t, sh.EnqueueRefresh(key))
	retrieved, ok := sh.DequeueExpired()

	require.True(t, ok)
	require.Equal(t, key, retrieved)
}

// TestShard_EnqueueRefresh_Full returns false when queue is full.
func TestShard_EnqueueRefresh_Full(t *testing.T) {
	sh := NewShard(0)
	// Fill queue (capacity is 4096)
	for i := 0; i < 4096; i++ {
		sh.EnqueueRefresh(uint64(i))
	}

	// Next should fail
	require.False(t, sh.EnqueueRefresh(9999))
}

// TestShard_DequeueExpired_Empty returns false when queue is empty.
func TestShard_DequeueExpired_Empty(t *testing.T) {
	sh := NewShard(0)
	_, ok := sh.DequeueExpired()

	require.False(t, ok)
}

// TestShard_AddMem_UpdatesMemory updates memory atomically.
func TestShard_AddMem_UpdatesMemory(t *testing.T) {
	sh := NewShard(0)
	initial := sh.Weight()

	sh.AddMem(1024)
	require.Equal(t, initial+1024, sh.Weight())

	sh.AddMem(-512)
	require.Equal(t, initial+512, sh.Weight())
}

// TestShard_ConcurrentRemove verifies thread-safety of Remove.
func TestShard_ConcurrentRemove(t *testing.T) {
	sh := NewShard(0)
	const numKeys = 100

	// Insert keys
	for i := 0; i < numKeys; i++ {
		entry := model.NewEmptyEntry(model.NewKey("test"), 0, nil)
		entry.SetPayload([]byte("data"))
		sh.Set(uint64(i), entry)
	}

	// Remove concurrently
	var wg sync.WaitGroup
	wg.Add(numKeys)
	for i := 0; i < numKeys; i++ {
		go func(key uint64) {
			defer wg.Done()
			sh.Remove(key)
		}(uint64(i))
	}

	wg.Wait()
	require.Equal(t, int64(0), sh.Len())
	require.Equal(t, int64(0), sh.Weight())
}
