package share

import (
	"sync"
	"testing"
	"time"
)

// testClock returns a controllable clock and a sleep recorder.
type testClock struct {
	mu    sync.Mutex
	now   time.Time
	slept []time.Duration
}

func newTestClock() *testClock {
	return &testClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func (c *testClock) Sleep(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.slept = append(c.slept, d)
}

func (c *testClock) Slept() []time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]time.Duration, len(c.slept))
	copy(out, c.slept)
	return out
}

func newTestLimiter() (*AuthLimiter, *testClock) {
	clock := newTestClock()
	l := NewAuthLimiter()
	l.Now = clock.Now
	l.Sleep = clock.Sleep
	return l, clock
}

func TestAuthLimiterLocksAfterThreshold(t *testing.T) {
	l, _ := newTestLimiter()
	const key, shareID = "1.2.3.4|abc", "abc"

	for i := 0; i < authMaxFailuresPerKey; i++ {
		if blocked, _ := l.Blocked(key, shareID); blocked {
			t.Fatalf("key blocked before reaching the threshold (attempt %d)", i+1)
		}
		l.Failure(key, shareID)
	}

	blocked, wait := l.Blocked(key, shareID)
	if !blocked {
		t.Fatal("key should be locked out after the failure threshold")
	}
	if wait != authLockout {
		t.Fatalf("lockout wait = %v, want %v", wait, authLockout)
	}
}

func TestAuthLimiterLockoutExpires(t *testing.T) {
	l, clock := newTestLimiter()
	const key, shareID = "1.2.3.4|abc", "abc"

	for i := 0; i < authMaxFailuresPerKey; i++ {
		l.Failure(key, shareID)
	}
	if blocked, _ := l.Blocked(key, shareID); !blocked {
		t.Fatal("expected lockout")
	}

	clock.Advance(authLockout + time.Second)

	if blocked, _ := l.Blocked(key, shareID); blocked {
		t.Fatal("lockout should have expired")
	}
}

func TestAuthLimiterSuccessResetsKey(t *testing.T) {
	l, _ := newTestLimiter()
	const key, shareID = "1.2.3.4|abc", "abc"

	for i := 0; i < authMaxFailuresPerKey-1; i++ {
		l.Failure(key, shareID)
	}
	l.Success(key)
	if blocked, _ := l.Blocked(key, shareID); blocked {
		t.Fatal("a successful auth must clear the key counter")
	}
	// A fresh budget must be available again.
	for i := 0; i < authMaxFailuresPerKey-1; i++ {
		l.Failure(key, shareID)
	}
	if blocked, _ := l.Blocked(key, shareID); blocked {
		t.Fatal("counter should have restarted from zero")
	}
}

func TestAuthLimiterShareWideCapAcrossIPs(t *testing.T) {
	l, _ := newTestLimiter()
	const shareID = "abc"

	// Spread failures over distinct IPs. The per-key limit must never trigger,
	// but the per-share cap must.
	perIP := authMaxFailuresPerKey - 1
	attempts := 0
	for attempts < authMaxFailuresPerShare {
		key := "ip-" + string(rune('a'+attempts%26)) + string(rune('a'+(attempts/26)%26))
		for i := 0; i < perIP && attempts < authMaxFailuresPerShare; i++ {
			l.Failure(key, shareID)
			attempts++
		}
	}

	blocked, _ := l.Blocked("brand-new-ip", shareID)
	if !blocked {
		t.Fatalf("share %q should be locked out for everyone after %d failures", shareID, authMaxFailuresPerShare)
	}
}

func TestAuthLimiterPenaltyGrowsAndIsCapped(t *testing.T) {
	l, _ := newTestLimiter()
	const key, shareID = "1.2.3.4|abc", "abc"

	first, _ := l.Failure(key, shareID)
	if first != authPenalty {
		t.Fatalf("first penalty = %v, want %v", first, authPenalty)
	}
	second, _ := l.Failure(key, shareID)
	if second <= first {
		t.Fatalf("penalty should grow: %v -> %v", first, second)
	}

	for i := 0; i < 100; i++ {
		delay, _ := l.Failure("other|"+shareID, shareID)
		if delay > authMaxPenalty {
			t.Fatalf("penalty %v exceeds cap %v", delay, authMaxPenalty)
		}
	}
}

func TestAuthLimiterApplyDelayUsesInjectedSleep(t *testing.T) {
	l, clock := newTestLimiter()
	l.ApplyDelay(150 * time.Millisecond)
	slept := clock.Slept()
	if len(slept) != 1 || slept[0] != 150*time.Millisecond {
		t.Fatalf("slept = %v, want one 150ms sleep", slept)
	}
	// Zero and negative delays must not sleep.
	l.ApplyDelay(0)
	l.ApplyDelay(-time.Second)
	if len(clock.Slept()) != 1 {
		t.Fatalf("non-positive delays should not sleep, got %v", clock.Slept())
	}
}

func TestAuthLimiterIsolatesKeysAndShares(t *testing.T) {
	l, _ := newTestLimiter()

	for i := 0; i < authMaxFailuresPerKey; i++ {
		l.Failure("1.1.1.1|abc", "abc")
	}
	if blocked, _ := l.Blocked("1.1.1.1|abc", "abc"); !blocked {
		t.Fatal("offending key should be blocked")
	}
	if blocked, _ := l.Blocked("2.2.2.2|abc", "abc"); blocked {
		t.Fatal("a different IP must not be blocked by another IP's failures")
	}
	if blocked, _ := l.Blocked("1.1.1.1|def", "def"); blocked {
		t.Fatal("a different share must not be blocked")
	}
}

func TestAuthLimiterBoundsKeyMap(t *testing.T) {
	l, _ := newTestLimiter()
	l.mu.Lock()
	l.maxKeys = 10
	l.mu.Unlock()

	for i := 0; i < 200; i++ {
		l.Failure("key-"+string(rune('0'+i%10))+string(rune('a'+i/10%26)), "share-"+string(rune('a'+i%26)))
	}
	l.mu.Lock()
	size := len(l.keys)
	l.mu.Unlock()
	if size > l.maxKeys {
		t.Fatalf("key map grew to %d, want <= %d", size, l.maxKeys)
	}
}

func TestAuthLimiterBoundsConcurrentDelays(t *testing.T) {
	l, _ := newTestLimiter()

	// Occupy every delay slot with a sleep that blocks until released, then
	// assert that additional delays return without sleeping.
	release := make(chan struct{})
	l.Sleep = func(time.Duration) { <-release }
	var wg sync.WaitGroup
	for i := 0; i < authMaxConcurrentDelays; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.ApplyDelay(time.Hour)
		}()
	}
	// Give the goroutines a moment to acquire their slots.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(l.delaySlots) == authMaxConcurrentDelays {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if len(l.delaySlots) != authMaxConcurrentDelays {
		t.Fatalf("delay slots = %d, want %d", len(l.delaySlots), authMaxConcurrentDelays)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		l.ApplyDelay(time.Hour) // must return at once: no free slot
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ApplyDelay blocked even though every slot was busy")
	}

	close(release)
}
