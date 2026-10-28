package ratelimit

import (
	"sync"
	"time"
)

type entry struct {
	count   int
	resetAt time.Time
}

// Limiter is an in-memory fixed-window counter keyed by an arbitrary string.
// It is safe for concurrent use. Buckets are pruned lazily as windows expire.
type Limiter struct {
	mu          sync.Mutex
	entries     map[string]entry
	now         func() time.Time
	nextCleanup time.Time
}

func New() *Limiter {
	return &Limiter{
		entries: make(map[string]entry),
		now:     time.Now,
	}
}

// Allow records a request for key and reports whether it is within limit for
// the current window. When the limit is exceeded it returns the duration until
// the window resets. A non-positive limit or window disables limiting.
func (l *Limiter) Allow(key string, limit int, window time.Duration) (time.Duration, bool) {
	if l == nil || limit <= 0 || window <= 0 {
		return 0, true
	}

	const cleanupInterval = time.Minute

	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if !now.Before(l.nextCleanup) {
		for k, e := range l.entries {
			if !now.Before(e.resetAt) {
				delete(l.entries, k)
			}
		}
		l.nextCleanup = now.Add(cleanupInterval)
	}

	e, ok := l.entries[key]
	if !ok || !now.Before(e.resetAt) {
		e = entry{resetAt: now.Add(window)}
	}

	if e.count >= limit {
		return e.resetAt.Sub(now), false
	}

	e.count++
	l.entries[key] = e
	return 0, true
}
