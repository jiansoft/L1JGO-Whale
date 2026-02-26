package system

import (
	"time"

	coresys "github.com/l1jgo/server/internal/core/system"
	"github.com/l1jgo/server/internal/handler"
)

// SkillSystem processes queued skill requests in Phase 2.
// Handlers parse packets and call QueueSkill(); this system dispatches
// to handler.ProcessSkill in deterministic order.
type SkillSystem struct {
	deps     *handler.Deps
	requests []handler.SkillRequest
}

func NewSkillSystem(deps *handler.Deps) *SkillSystem {
	return &SkillSystem{deps: deps}
}

func (s *SkillSystem) Phase() coresys.Phase { return coresys.PhaseUpdate }

// QueueSkill implements handler.SkillQueue.
func (s *SkillSystem) QueueSkill(req handler.SkillRequest) {
	s.requests = append(s.requests, req)
}

func (s *SkillSystem) Update(_ time.Duration) {
	for _, req := range s.requests {
		handler.ProcessSkill(req.SessionID, req.SkillID, req.TargetID, s.deps)
	}
	s.requests = s.requests[:0]
}
