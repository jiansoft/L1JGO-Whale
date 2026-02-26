package system

import (
	"time"

	coresys "github.com/l1jgo/server/internal/core/system"
	"github.com/l1jgo/server/internal/handler"
)

// CombatSystem processes queued attack requests in Phase 2.
// Handlers parse packets and call QueueAttack(); this system dispatches
// to handler.ProcessMeleeAttack / ProcessRangedAttack in deterministic order.
// Event emission (EntityKilled) is handled inside handleNpcDeath via Deps.Bus.
type CombatSystem struct {
	deps     *handler.Deps
	requests []handler.AttackRequest
}

func NewCombatSystem(deps *handler.Deps) *CombatSystem {
	return &CombatSystem{deps: deps}
}

func (s *CombatSystem) Phase() coresys.Phase { return coresys.PhaseUpdate }

// QueueAttack implements handler.CombatQueue.
func (s *CombatSystem) QueueAttack(req handler.AttackRequest) {
	s.requests = append(s.requests, req)
}

func (s *CombatSystem) Update(_ time.Duration) {
	for _, req := range s.requests {
		if req.IsMelee {
			handler.ProcessMeleeAttack(req.AttackerSessionID, req.TargetID, s.deps)
		} else {
			handler.ProcessRangedAttack(req.AttackerSessionID, req.TargetID, s.deps)
		}
	}
	s.requests = s.requests[:0]
}
