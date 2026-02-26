package system

import (
	"time"

	coresys "github.com/l1jgo/server/internal/core/system"
	"github.com/l1jgo/server/internal/net"
)

// OutputSystem flushes buffered output packets for all sessions.
// Phase 4 (Output) — runs after all game logic, before persistence.
//
// During Phases 0-3, handlers and systems call sess.Send() which appends
// packets to a per-session buffer. OutputSystem drains these buffers into
// the OutQueue channels, where writeLoop goroutines pick them up for
// encryption and TCP transmission.
//
// Benefits:
//   - Single flush point = predictable network I/O timing
//   - Multiple packets per tick are batched into fewer channel operations
//   - Compliant with CLAUDE.md Phase 4 architecture
type OutputSystem struct {
	store *net.SessionStore
}

func NewOutputSystem(store *net.SessionStore) *OutputSystem {
	return &OutputSystem{store: store}
}

func (s *OutputSystem) Phase() coresys.Phase { return coresys.PhaseOutput }

func (s *OutputSystem) Update(_ time.Duration) {
	s.store.ForEach(func(sess *net.Session) {
		sess.FlushOutput()
	})
}
