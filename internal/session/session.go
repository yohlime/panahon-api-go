package session

import (
	"context"
	"errors"
	"sync"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
	"github.com/google/uuid"
)

// cacheTTL bounds how long a session's liveness is cached. Short enough that a
// deleted/blocked session stops authenticating promptly, long enough to avoid a
// database round-trip on every request.
const cacheTTL = 30 * time.Second

type entry struct {
	live    bool
	expires time.Time
}

type cache struct {
	mu sync.Mutex
	m  map[uuid.UUID]entry
}

func newCache() *cache {
	return &cache{m: make(map[uuid.UUID]entry)}
}

func (c *cache) get(id uuid.UUID) (bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[id]
	if !ok || time.Now().After(e.expires) {
		return false, false
	}
	return e.live, true
}

func (c *cache) set(id uuid.UUID, live bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[id] = entry{live: live, expires: time.Now().Add(cacheTTL)}
}

func (c *cache) del(id uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, id)
}

func (c *cache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[uuid.UUID]entry)
}

var liveness = newCache()

// Live reports whether the session still exists and is neither blocked nor
// expired. Results are cached briefly (see cacheTTL). A database error other
// than "not found" fails closed but is not cached, so a transient blip does not
// lock a user out for the whole TTL.
func Live(ctx context.Context, store db.Store, id uuid.UUID) bool {
	if live, ok := liveness.get(id); ok {
		return live
	}

	s, err := store.GetSession(ctx, id)
	switch {
	case err == nil:
		live := !s.IsBlocked && time.Now().Before(s.ExpiresAt.Time)
		liveness.set(id, live)
		return live
	case errors.Is(err, db.ErrRecordNotFound):
		liveness.set(id, false)
		return false
	default:
		return false
	}
}

// Invalidate drops a session from the liveness cache so that a logout or block
// takes effect immediately instead of after the cache TTL.
func Invalidate(id uuid.UUID) {
	liveness.del(id)
}

// Clear drops every cached liveness entry. Used when many sessions are removed
// at once (e.g. all of a user's sessions after a username change).
func Clear() {
	liveness.clear()
}
