package db

import (
	"github.com/Borislavv/go-ash-cache/internal/cache/db/model"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestShard_EnableLRU_InitializesStructures initializes LRU structures.
func TestShard_EnableLRU_InitializesStructures(t *testing.T) {
	sh := NewShard(0)
	sh.enableLRU()

	require.True(t, sh.lruOn)
	require.NotNil(t, sh.lru)
	require.NotNil(t, sh.lidx)
}

// TestShard_EnableLRU_WithExistingEntries adds existing entries to LRU.
func TestShard_EnableLRU_WithExistingEntries(t *testing.T) {
	sh := NewShard(0)
	// Add entries before enabling LRU
	for i := 0; i < 5; i++ {
		entry := model.NewEntry(model.NewKey("test"), 0, false)
		entry.SetPayload([]byte("data"))
		sh.Set(uint64(i), entry)
	}

	sh.enableLRU()

	require.True(t, sh.lruOn)
	require.Equal(t, 5, sh.lru.Len(), "LRU should contain all existing entries")
	require.Equal(t, 5, len(sh.lidx))
}

// TestShard_DisableLRU_ClearsStructures clears LRU structures.
func TestShard_DisableLRU_ClearsStructures(t *testing.T) {
	sh := NewShard(0)
	sh.enableLRU()
	sh.disableLRU()

	require.False(t, sh.lruOn)
	require.Nil(t, sh.lru)
	require.Nil(t, sh.lidx)
}

// TestShard_LRUOnInsert_AddsToFront adds new entries to front of LRU.
func TestShard_LRUOnInsert_AddsToFront(t *testing.T) {
	sh := NewShard(0)
	sh.enableLRU()

	entry1 := model.NewEntry(model.NewKey("test1"), 0, false)
	entry1.SetPayload([]byte("data1"))
	entry2 := model.NewEntry(model.NewKey("test2"), 0, false)
	entry2.SetPayload([]byte("data2"))

	sh.Lock()
	sh.items[1] = entry1
	sh.lruOnInsertUnlocked(1)
	sh.items[2] = entry2
	sh.lruOnInsertUnlocked(2)
	sh.Unlock()

	require.Equal(t, 2, sh.lru.Len())
	// Front should be the last inserted (2)
	frontKey := sh.lru.Front().Value.(uint64)
	require.Equal(t, uint64(2), frontKey)
}

// TestShard_LRUOnAccess_MovesToFront moves accessed entries to front.
func TestShard_LRUOnAccess_MovesToFront(t *testing.T) {
	sh := NewShard(0)
	sh.enableLRU()

	entry1 := model.NewEntry(model.NewKey("test1"), 0, false)
	entry1.SetPayload([]byte("data1"))
	entry2 := model.NewEntry(model.NewKey("test2"), 0, false)
	entry2.SetPayload([]byte("data2"))

	sh.Lock()
	sh.items[1] = entry1
	sh.lruOnInsertUnlocked(1)
	sh.items[2] = entry2
	sh.lruOnInsertUnlocked(2)
	// Now front is 2, back is 1
	sh.lruOnAccessUnlocked(1) // Access 1, should move to front
	sh.Unlock()

	frontKey := sh.lru.Front().Value.(uint64)
	require.Equal(t, uint64(1), frontKey, "accessed key should be at front")
	backKey := sh.lru.Back().Value.(uint64)
	require.Equal(t, uint64(2), backKey, "other key should be at back")
}

// TestShard_LRUOnDelete_RemovesFromList removes entries from LRU on delete.
func TestShard_LRUOnDelete_RemovesFromList(t *testing.T) {
	sh := NewShard(0)
	sh.enableLRU()

	entry1 := model.NewEntry(model.NewKey("test1"), 0, false)
	entry1.SetPayload([]byte("data1"))
	entry2 := model.NewEntry(model.NewKey("test2"), 0, false)
	entry2.SetPayload([]byte("data2"))

	sh.Lock()
	sh.items[1] = entry1
	sh.lruOnInsertUnlocked(1)
	sh.items[2] = entry2
	sh.lruOnInsertUnlocked(2)
	sh.lruOnDeleteUnlocked(1)
	sh.Unlock()

	require.Equal(t, 1, sh.lru.Len())
	require.Nil(t, sh.lidx[1])
	require.NotNil(t, sh.lidx[2])
}

// TestShard_LRUPeekTail_ReturnsLeastRecent returns least recently used entry.
func TestShard_LRUPeekTail_ReturnsLeastRecent(t *testing.T) {
	sh := NewShard(0)
	sh.enableLRU()

	entry1 := model.NewEntry(model.NewKey("test1"), 0, false)
	entry1.SetPayload([]byte("data1"))
	entry2 := model.NewEntry(model.NewKey("test2"), 0, false)
	entry2.SetPayload([]byte("data2"))
	entry3 := model.NewEntry(model.NewKey("test3"), 0, false)
	entry3.SetPayload([]byte("data3"))

	sh.Lock()
	sh.items[1] = entry1
	sh.lruOnInsertUnlocked(1)
	sh.items[2] = entry2
	sh.lruOnInsertUnlocked(2)
	sh.items[3] = entry3
	sh.lruOnInsertUnlocked(3)
	// Order: 3 (front) -> 2 -> 1 (back)
	sh.Unlock()

	key, val, ok := sh.lruPeekTail()
	require.True(t, ok)
	require.Equal(t, uint64(1), key, "should return least recent")
	require.Equal(t, entry1.PayloadBytes(), val.PayloadBytes())
}

// TestShard_LRUPeekTail_Empty returns false when LRU is empty.
func TestShard_LRUPeekTail_Empty(t *testing.T) {
	sh := NewShard(0)
	sh.enableLRU()

	_, _, ok := sh.lruPeekTail()
	require.False(t, ok)
}

// TestShard_LRUPeekTail_NoLRU returns false when LRU is disabled.
func TestShard_LRUPeekTail_NoLRU(t *testing.T) {
	sh := NewShard(0)
	// Don't enable LRU

	_, _, ok := sh.lruPeekTail()
	require.False(t, ok)
}

// TestShard_LRUPopTail_RemovesAndReturns removes entry from tail.
func TestShard_LRUPopTail_RemovesAndReturns(t *testing.T) {
	sh := NewShard(0)
	sh.enableLRU()

	entry1 := model.NewEntry(model.NewKey("test1"), 0, false)
	entry1.SetPayload([]byte("data1"))
	entry2 := model.NewEntry(model.NewKey("test2"), 0, false)
	entry2.SetPayload([]byte("data2"))

	sh.Lock()
	sh.items[1] = entry1
	sh.lruOnInsertUnlocked(1)
	sh.items[2] = entry2
	sh.lruOnInsertUnlocked(2)
	sh.Unlock()

	initialLen := sh.Len()
	key, val, ok := sh.lruPopTail()

	require.True(t, ok)
	require.Equal(t, uint64(1), key)
	require.Equal(t, entry1.PayloadBytes(), val.PayloadBytes())
	require.Equal(t, initialLen-1, sh.Len(), "should decrement length")
	require.Equal(t, 1, sh.lru.Len(), "should remove from LRU list")
	require.Nil(t, sh.lidx[1], "should remove from index")
}

// TestShard_LRUPeekHead_ReturnsMostRecent returns most recently used entry.
func TestShard_LRUPeekHead_ReturnsMostRecent(t *testing.T) {
	sh := NewShard(0)
	sh.enableLRU()

	entry1 := model.NewEntry(model.NewKey("test1"), 0, false)
	entry1.SetPayload([]byte("data1"))
	entry2 := model.NewEntry(model.NewKey("test2"), 0, false)
	entry2.SetPayload([]byte("data2"))

	sh.Lock()
	sh.items[1] = entry1
	sh.lruOnInsertUnlocked(1)
	sh.items[2] = entry2
	sh.lruOnInsertUnlocked(2)
	// Order: 2 (front) -> 1 (back)
	sh.Unlock()

	key, val, ok := sh.lruPeekHead()
	require.True(t, ok)
	require.Equal(t, uint64(2), key, "should return most recent")
	require.Equal(t, entry2.PayloadBytes(), val.PayloadBytes())
}

// TestShard_TouchLRU_MovesToFront moves entry to front on touch.
func TestShard_TouchLRU_MovesToFront(t *testing.T) {
	sh := NewShard(0)
	sh.enableLRU()

	entry1 := model.NewEntry(model.NewKey("test1"), 0, false)
	entry1.SetPayload([]byte("data1"))
	entry2 := model.NewEntry(model.NewKey("test2"), 0, false)
	entry2.SetPayload([]byte("data2"))

	sh.Lock()
	sh.items[1] = entry1
	sh.lruOnInsertUnlocked(1)
	sh.items[2] = entry2
	sh.lruOnInsertUnlocked(2)
	sh.Unlock()

	// Touch 1 (currently at back)
	sh.touchLRU(1)

	sh.RLock()
	frontKey := sh.lru.Front().Value.(uint64)
	sh.RUnlock()
	require.Equal(t, uint64(1), frontKey, "touched key should be at front")
}

// TestShard_LRUPeekHeadK_SelectsFirstMatch selects first matching entry from head.
func TestShard_LRUPeekHeadK_SelectsFirstMatch(t *testing.T) {
	sh := NewShard(0)
	sh.enableLRU()

	for i := 0; i < 5; i++ {
		entry := model.NewEntry(model.NewKey("test"), 0, false)
		entry.SetPayload([]byte("data"))
		sh.Lock()
		sh.items[uint64(i)] = entry
		sh.lruOnInsertUnlocked(uint64(i))
		sh.Unlock()
	}

	// Find entry with key 2 (should be in first 3 from head)
	// Note: We need to find by the map key (uint64), not by GetKey().Value() which is a hash
	var targetEntry *model.Entry
	sh.RLock()
	targetEntry = sh.items[2]
	sh.RUnlock()
	require.NotNil(t, targetEntry, "entry with key 2 should exist")

	val, ok := sh.lruPeekHeadK(3, func(e *model.Entry) bool {
		return e == targetEntry
	})

	require.True(t, ok)
	require.Equal(t, targetEntry, val)
}

// TestShard_LRUPeekTailK_SelectsFirstMatch selects first matching entry from tail.
func TestShard_LRUPeekTailK_SelectsFirstMatch(t *testing.T) {
	sh := NewShard(0)
	sh.enableLRU()

	for i := 0; i < 5; i++ {
		entry := model.NewEntry(model.NewKey("test"), 0, false)
		entry.SetPayload([]byte("data"))
		sh.Lock()
		sh.items[uint64(i)] = entry
		sh.lruOnInsertUnlocked(uint64(i))
		sh.Unlock()
	}

	// Find entry with key 2 (should be in first 3 from tail)
	// Note: We need to find by the map key (uint64), not by GetKey().Value() which is a hash
	var targetEntry *model.Entry
	sh.RLock()
	targetEntry = sh.items[2]
	sh.RUnlock()
	require.NotNil(t, targetEntry, "entry with key 2 should exist")

	val, ok := sh.lruPeekTailK(3, func(e *model.Entry) bool {
		return e == targetEntry
	})

	require.True(t, ok)
	require.Equal(t, targetEntry, val)
}
