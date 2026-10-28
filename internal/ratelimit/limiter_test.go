package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLimiterAllow(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	l := New()
	l.now = func() time.Time { return now }

	retryAfter, ok := l.Allow("login:192.0.2.1", 2, time.Minute)
	require.True(t, ok)
	require.Zero(t, retryAfter)

	_, ok = l.Allow("login:192.0.2.1", 2, time.Minute)
	require.True(t, ok)

	retryAfter, ok = l.Allow("login:192.0.2.1", 2, time.Minute)
	require.False(t, ok)
	require.Greater(t, retryAfter, time.Duration(0))

	// Another key has its own bucket.
	_, ok = l.Allow("login:192.0.2.2", 2, time.Minute)
	require.True(t, ok)

	// The window resets after its duration.
	now = now.Add(time.Minute)
	_, ok = l.Allow("login:192.0.2.1", 2, time.Minute)
	require.True(t, ok)
}

func TestLimiterDisabled(t *testing.T) {
	l := New()

	for range 5 {
		_, ok := l.Allow("key", 0, time.Minute)
		require.True(t, ok)
	}

	var nilLimiter *Limiter
	_, ok := nilLimiter.Allow("key", 10, time.Minute)
	require.True(t, ok)
}
