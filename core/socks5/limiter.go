package socks5

import (
	"sync"
	"time"
)

const (
	defaultMaxFailures = 5
	defaultFailWindow  = time.Minute
	defaultLockout     = 5 * time.Minute
)

// AuthLimiter throttles failed authentication attempts per remote host.
//
// A host is locked out after MaxFailures failures inside Window. The lockout
// lasts Lockout, and a successful authentication clears the record. The zero
// value is not usable; call NewAuthLimiter.
type AuthLimiter struct {
	maxFailures int
	window      time.Duration
	lockout     time.Duration

	mu        sync.Mutex
	records   map[string]*authRecord
	lastSweep time.Time
	now       func() time.Time
}

type authRecord struct {
	failures     int
	firstFailure time.Time
	blockedUntil time.Time
}

// NewAuthLimiter returns a limiter. Non-positive values fall back to 5
// failures per minute and a 5 minute lockout.
func NewAuthLimiter(maxFailures int, window, lockout time.Duration) *AuthLimiter {
	if maxFailures <= 0 {
		maxFailures = defaultMaxFailures
	}
	if window <= 0 {
		window = defaultFailWindow
	}
	if lockout <= 0 {
		lockout = defaultLockout
	}
	return &AuthLimiter{
		maxFailures: maxFailures,
		window:      window,
		lockout:     lockout,
		records:     make(map[string]*authRecord),
		now:         time.Now,
	}
}

// Allow reports whether remote may attempt authentication now.
func (l *AuthLimiter) Allow(remote string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	record, ok := l.records[remote]
	if !ok {
		return true
	}
	return !l.now().Before(record.blockedUntil)
}

// Failure records a failed attempt and starts a lockout once the threshold is
// reached.
func (l *AuthLimiter) Failure(remote string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweepLocked(now)

	record, ok := l.records[remote]
	if !ok {
		l.records[remote] = &authRecord{failures: 1, firstFailure: now}
		return
	}
	if now.Sub(record.firstFailure) > l.window {
		record.failures = 0
		record.firstFailure = now
		record.blockedUntil = time.Time{}
	}
	record.failures++
	if record.failures >= l.maxFailures {
		record.blockedUntil = now.Add(l.lockout)
	}
}

// Success clears the failure record for remote.
func (l *AuthLimiter) Success(remote string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.records, remote)
}

// sweepLocked drops records that no longer affect decisions. It runs at most
// once per window to keep the map bounded under scanning.
func (l *AuthLimiter) sweepLocked(now time.Time) {
	if now.Sub(l.lastSweep) < l.window {
		return
	}
	l.lastSweep = now
	for remote, record := range l.records {
		if now.Before(record.blockedUntil) {
			continue
		}
		if now.Sub(record.firstFailure) > l.window {
			delete(l.records, remote)
		}
	}
}
