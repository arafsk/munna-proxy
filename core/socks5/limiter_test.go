package socks5

import (
	"testing"
	"time"
)

func TestAuthLimiterLockoutAndReset(t *testing.T) {
	limiter := NewAuthLimiter(3, time.Minute, time.Minute)
	now := time.Unix(1_700_000_000, 0)
	limiter.now = func() time.Time { return now }
	const remote = "192.0.2.10"

	for i := 0; i < 3; i++ {
		if !limiter.Allow(remote) {
			t.Fatalf("attempt %d: unexpectedly blocked", i+1)
		}
		limiter.Failure(remote)
	}
	if limiter.Allow(remote) {
		t.Fatal("expected lockout after threshold")
	}

	now = now.Add(2 * time.Minute)
	if !limiter.Allow(remote) {
		t.Fatal("expected lockout to expire")
	}
	limiter.Success(remote)
	if !limiter.Allow(remote) {
		t.Fatal("expected success to clear the record")
	}
}

func TestAuthLimiterWindowResetsFailures(t *testing.T) {
	limiter := NewAuthLimiter(2, time.Minute, time.Minute)
	now := time.Unix(1_700_000_000, 0)
	limiter.now = func() time.Time { return now }
	const remote = "192.0.2.11"

	limiter.Failure(remote)
	now = now.Add(2 * time.Minute)
	limiter.Failure(remote)
	if !limiter.Allow(remote) {
		t.Fatal("failures outside the window must not accumulate")
	}
}

func TestAuthLimiterIsolatesRemotes(t *testing.T) {
	limiter := NewAuthLimiter(1, time.Minute, time.Minute)
	limiter.Failure("192.0.2.12")
	if !limiter.Allow("192.0.2.13") {
		t.Fatal("one remote must not block another")
	}
}

func TestAuthLimiterSweepsExpiredRecords(t *testing.T) {
	limiter := NewAuthLimiter(1, time.Minute, time.Minute)
	now := time.Unix(1_700_000_000, 0)
	limiter.now = func() time.Time { return now }
	limiter.Failure("192.0.2.14")

	now = now.Add(10 * time.Minute)
	limiter.Failure("192.0.2.15")

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if _, ok := limiter.records["192.0.2.14"]; ok {
		t.Fatal("expired record was not swept")
	}
}

func TestAuthLimiterDefaults(t *testing.T) {
	limiter := NewAuthLimiter(0, 0, 0)
	if limiter.maxFailures != defaultMaxFailures {
		t.Fatalf("maxFailures = %d, want %d", limiter.maxFailures, defaultMaxFailures)
	}
	if limiter.window != defaultFailWindow {
		t.Fatalf("window = %v, want %v", limiter.window, defaultFailWindow)
	}
	if limiter.lockout != defaultLockout {
		t.Fatalf("lockout = %v, want %v", limiter.lockout, defaultLockout)
	}
}
