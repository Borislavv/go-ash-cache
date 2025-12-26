package queue

import "sync"

type Queue struct {
	mu         sync.Mutex
	buf        []uint64
	head, tail int
}

func (q *Queue) Init(size int) {
	if size < 2 {
		size = 2
	}
	q.buf = make([]uint64, size)
	q.head, q.tail = 0, 0
}

func (q *Queue) TryPush(k uint64) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	next := (q.head + 1) % len(q.buf)
	if next == q.tail { // full
		return false
	}
	q.buf[q.head] = k
	q.head = next
	return true
}

func (q *Queue) TryPop() (uint64, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.head == q.tail {
		return 0, false
	}
	v := q.buf[q.tail]
	q.tail = (q.tail + 1) % len(q.buf)
	return v, true
}
