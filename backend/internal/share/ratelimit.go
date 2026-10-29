package share

import (
	"sync"
	"time"
)

// Authentication rate limiting for public share links.
//
// A public share link is reachable by anyone who knows (or guesses) the
// share_id, and its password is the only secret protecting the content. Unlike
// the interactive login endpoint there is no per-account lockout hiding behind
// it, so the handler must bound online guessing itself: without a limit an
// attacker can try every entry of a wordlist at network speed.
//
// The policy implemented here is deliberately simple and stateless from the
// caller's point of view:
//
//   - Failures are counted per (share_id, client IP) inside a rolling window.
//   - Reaching the threshold locks the key for a lockout period; while locked
//     the handler answers 429 without touching the database.
//   - Every failure also adds a small, capped delay so even below the threshold
//     a guessing loop is slowed to a crawl.
//   - A successful authentication clears the counter for that key.
//
// A separate per-share counter (independent of the IP) stops a distributed
// attacker from rotating source addresses: the password can only be guessed
// so many times per window no matter how many hosts are used.

const (
	// authWindow is the rolling window failures are counted in.
	authWindow = 10 * time.Minute
	// authMaxFailuresPerKey is the number of wrong passwords tolerated for one
	// (share, IP) pair before the pair is locked out.
	authMaxFailuresPerKey = 5
	// authMaxFailuresPerShare caps failures for a share across all IPs. It is
	// higher than the per-key limit so that a handful of legitimate typos from
	// different people cannot lock a share for everybody, while a distributed
	// brute force still runs out of attempts.
	authMaxFailuresPerShare = 30
	// authLockout is how long a key that exceeded its budget stays blocked.
	authLockout = 15 * time.Minute
	// authPenalty is added to the response time of every failed authentication.
	authPenalty = 250 * time.Millisecond
	// authMaxPenalty caps the accumulated penalty applied before answering.
	authMaxPenalty = 2 * time.Second
	// authMaxKeys bounds the number of tracked keys so that an attacker cannot
	// grow memory without bound by spraying random share ids or source IPs.
	authMaxKeys = 20000
	// authMaxConcurrentDelays bounds how many requests may sleep in the penalty
	// at the same time, so the mitigation itself cannot exhaust goroutines.
	authMaxConcurrentDelays = 64
)

// AuthLimiter tracks failed public-share authentication attempts.
//
// The zero value is not usable; build one with NewAuthLimiter. All methods are
// safe for concurrent use.
type AuthLimiter struct {
	mu sync.Mutex

	window      time.Duration
	maxPerKey   int
	maxPerShare int
	lockout     time.Duration
	penalty     time.Duration
	maxPenalty  time.Duration
	maxKeys     int

	// Now and Sleep are injectable so tests can run without real delays.
	Now   func() time.Time
	Sleep func(time.Duration)

	keys         map[string]*authCounter
	shares       map[string]*authCounter
	lastCleanup  time.Time
	cleanupEvery time.Duration

	// delaySlots bounds how many request goroutines may sit in the response
	// penalty at once. Sleeping is a deliberate cost, but an attacker must not
	// be able to turn it into a memory/goroutine exhaustion vector by flooding
	// failures: when every slot is busy the delay is skipped, and the failure
	// is still counted so the lockout arrives sooner.
	delaySlots chan struct{}
}

type authCounter struct {
	failures    int
	windowStart time.Time
	lockedUntil time.Time
}

// NewAuthLimiter returns a limiter using the package policy.
func NewAuthLimiter() *AuthLimiter {
	return &AuthLimiter{
		window:       authWindow,
		maxPerKey:    authMaxFailuresPerKey,
		maxPerShare:  authMaxFailuresPerShare,
		lockout:      authLockout,
		penalty:      authPenalty,
		maxPenalty:   authMaxPenalty,
		maxKeys:      authMaxKeys,
		Now:          time.Now,
		Sleep:        time.Sleep,
		keys:         make(map[string]*authCounter),
		shares:       make(map[string]*authCounter),
		cleanupEvery: time.Minute,
		delaySlots:   make(chan struct{}, authMaxConcurrentDelays),
	}
}

// SetPolicy overrides the limiter policy. Tests and operators that need a
// different trade-off between usability and resistance can call it before the
// limiter is used.
func (l *AuthLimiter) SetPolicy(window time.Duration, maxPerKey, maxPerShare int, lockout, penalty, maxPenalty time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.window = window
	l.maxPerKey = maxPerKey
	l.maxPerShare = maxPerShare
	l.lockout = lockout
	l.penalty = penalty
	l.maxPenalty = maxPenalty
}

// Blocked reports whether key is currently locked out and how long the caller
// should wait before retrying.
func (l *AuthLimiter) Blocked(key, shareID string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.Now()
	l.cleanupLocked(now)
	if until, ok := l.lockedUntilLocked(l.keys, key, now); ok {
		return true, until
	}
	if until, ok := l.lockedUntilLocked(l.shares, shareID, now); ok {
		return true, until
	}
	return false, 0
}

func (l *AuthLimiter) lockedUntilLocked(set map[string]*authCounter, key string, now time.Time) (time.Duration, bool) {
	c, ok := set[key]
	if !ok || c.lockedUntil.IsZero() {
		return 0, false
	}
	if now.Before(c.lockedUntil) {
		return c.lockedUntil.Sub(now), true
	}
	// Lockout expired: start from a clean slate.
	delete(set, key)
	return 0, false
}

// Failure records one wrong password for key/shareID and returns the delay the
// handler should apply before answering so that timing does not reveal whether
// the share exists. It also reports whether the attempt just triggered a
// lockout.
func (l *AuthLimiter) Failure(key, shareID string) (delay time.Duration, locked bool) {
	l.mu.Lock()
	now := l.Now()
	l.cleanupLocked(now)
	keyLocked := l.recordLocked(l.keys, key, now, l.maxPerKey, true)
	shareLocked := l.recordLocked(l.shares, shareID, now, l.maxPerShare, false)
	delay = l.penalty
	if c, ok := l.keys[key]; ok {
		delay = time.Duration(c.failures) * l.penalty
	}
	l.mu.Unlock()

	if delay > l.maxPenalty {
		delay = l.maxPenalty
	}
	return delay, keyLocked || shareLocked
}

func (l *AuthLimiter) recordLocked(set map[string]*authCounter, key string, now time.Time, max int, enforceCap bool) bool {
	if enforceCap && len(set) >= l.maxKeys {
		l.evictLocked()
	}
	c, ok := set[key]
	if !ok {
		c = &authCounter{windowStart: now}
		set[key] = c
	}
	if now.Sub(c.windowStart) > l.window {
		c.failures = 0
		c.windowStart = now
	}
	c.failures++
	if c.failures >= max {
		c.lockedUntil = now.Add(l.lockout)
		return true
	}
	return false
}

// Success clears the failure counter for key. The per-share counter is left
// alone: a single successful login must not reset a distributed attack that is
// still in flight.
func (l *AuthLimiter) Success(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.keys, key)
}

// ApplyDelay sleeps for d, using the injectable sleep function.
//
// At most authMaxConcurrentDelays requests may be sleeping at once. Under a
// flood the extra requests return immediately instead of piling up goroutines;
// their failures were already counted, so the lockout still tightens.
func (l *AuthLimiter) ApplyDelay(d time.Duration) {
	if d <= 0 || l.delaySlots == nil {
		return
	}
	select {
	case l.delaySlots <- struct{}{}:
		defer func() { <-l.delaySlots }()
	default:
		return
	}
	sleep := l.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	sleep(d)
}

// cleanupLocked drops expired counters. It is amortised: the maps are only
// walked once per cleanupEvery interval, so the common path stays O(1).
func (l *AuthLimiter) cleanupLocked(now time.Time) {
	if !l.lastCleanup.IsZero() && now.Sub(l.lastCleanup) < l.cleanupEvery {
		return
	}
	l.lastCleanup = now
	l.pruneLocked(l.keys, now)
	l.pruneLocked(l.shares, now)
}

func (l *AuthLimiter) pruneLocked(set map[string]*authCounter, now time.Time) {
	for k, c := range set {
		lockedActive := !c.lockedUntil.IsZero() && now.Before(c.lockedUntil)
		if lockedActive {
			continue
		}
		if now.Sub(c.windowStart) > l.window {
			delete(set, k)
		}
	}
}

// evictLocked makes room in the key map when an attacker floods it with random
// keys. Dropping the whole map is acceptable: it only happens under abuse, and
// the per-share counter still bounds guessing for any individual share.
func (l *AuthLimiter) evictLocked() {
	now := l.Now()
	for k, c := range l.keys {
		if c.lockedUntil.IsZero() || !now.Before(c.lockedUntil) {
			delete(l.keys, k)
		}
	}
	if len(l.keys) >= l.maxKeys {
		l.keys = make(map[string]*authCounter)
	}
}
