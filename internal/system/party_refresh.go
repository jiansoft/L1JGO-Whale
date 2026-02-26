package system

import (
	"time"

	coresys "github.com/l1jgo/server/internal/core/system"
	"github.com/l1jgo/server/internal/handler"
	"github.com/l1jgo/server/internal/world"
)

// PartyRefreshSystem broadcasts party member positions to all partied players
// at a fixed interval. Phase 3 (PostUpdate).
type PartyRefreshSystem struct {
	world     *world.State
	deps      *handler.Deps
	tickCount int
	interval  int // refresh every N ticks
}

func NewPartyRefreshSystem(ws *world.State, deps *handler.Deps, intervalTicks int) *PartyRefreshSystem {
	return &PartyRefreshSystem{
		world:    ws,
		deps:     deps,
		interval: intervalTicks,
	}
}

func (s *PartyRefreshSystem) Phase() coresys.Phase { return coresys.PhasePostUpdate }

func (s *PartyRefreshSystem) Update(_ time.Duration) {
	s.tickCount++
	if s.tickCount < s.interval {
		return
	}
	s.tickCount = 0
	s.world.AllPlayers(func(p *world.PlayerInfo) {
		if p.PartyID != 0 {
			handler.RefreshPartyPositions(p, s.deps)
		}
	})
}
