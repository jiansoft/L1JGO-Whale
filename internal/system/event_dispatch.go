package system

import (
	"time"

	"github.com/l1jgo/server/internal/core/event"
	coresys "github.com/l1jgo/server/internal/core/system"
)

// EventDispatchSystem swaps the event bus double-buffer and dispatches
// all events from the previous tick. Phase 1 (PreUpdate).
type EventDispatchSystem struct {
	bus *event.Bus
}

func NewEventDispatchSystem(bus *event.Bus) *EventDispatchSystem {
	return &EventDispatchSystem{bus: bus}
}

func (s *EventDispatchSystem) Phase() coresys.Phase { return coresys.PhasePreUpdate }

func (s *EventDispatchSystem) Update(_ time.Duration) {
	s.bus.SwapBuffers()
	s.bus.DispatchAll()
}
