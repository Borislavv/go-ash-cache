package rate

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TestNewJitter_CreatesJitter verifies that NewJitter creates a working rate limiter.
func TestNewJitter_CreatesJitter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jitter := NewJitter(ctx, 10) // 10 per second
	require.NotNil(t, jitter)
	require.NotNil(t, jitter.Chan())
}

// TestJitter_Chan_ReceivesSignals verifies that Chan() receives rate-limited signals.
func TestJitter_Chan_ReceivesSignals(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jitter := NewJitter(ctx, 10) // 10 per second

	// Should receive at least one signal within reasonable time
	select {
	case <-jitter.Chan():
		// Success
	case <-time.After(200 * time.Millisecond):
		t.Fatal("jitter should emit signals")
	}
}

// TestJitter_Take_BlocksUntilSignal verifies that Take() blocks until signal.
func TestJitter_Take_BlocksUntilSignal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jitter := NewJitter(ctx, 10) // 10 per second

	// Take should eventually return (not block forever)
	done := make(chan struct{})
	go func() {
		jitter.Take()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Take should not block forever")
	}
}

// TestJitter_StopsOnContextCancel verifies that jitter stops when context is cancelled.
func TestJitter_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	jitter := NewJitter(ctx, 100) // High rate

	// Wait a bit to ensure jitter is running
	time.Sleep(10 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for cleanup (provider goroutine to exit and close channel)
	time.Sleep(200 * time.Millisecond)

	// Channel should be closed (provider closes it on exit)
	// Read any remaining values, then check if closed
	for {
		select {
		case _, ok := <-jitter.Chan():
			if !ok {
				// Channel is closed, test passes
				return
			}
		case <-time.After(50 * time.Millisecond):
			// If we can't read and channel isn't closed, that's also a failure
			// But give it more time
			_, ok := <-jitter.Chan()
			require.False(t, ok, "channel should be closed after context cancel")
			return
		}
	}
}

// TestNewJitter_MinBurst verifies that minimum burst size is enforced.
func TestNewJitter_MinBurst(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Very low limit should still have burst >= 1
	jitter := NewJitter(ctx, 1)
	require.NotNil(t, jitter)

	// Should be able to receive at least one signal
	select {
	case <-jitter.Chan():
		// Success
	case <-time.After(200 * time.Millisecond):
		t.Fatal("jitter should work even with low limit")
	}
}
