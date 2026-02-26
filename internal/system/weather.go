package system

import (
	"time"

	coresys "github.com/l1jgo/server/internal/core/system"
	"github.com/l1jgo/server/internal/net/packet"
	"github.com/l1jgo/server/internal/world"
)

// WeatherSystem checks for game-hour changes and randomizes weather on each
// new hour. Broadcasts S_WEATHER to all online players. Phase 3 (PostUpdate).
type WeatherSystem struct {
	world *world.State
}

func NewWeatherSystem(ws *world.State) *WeatherSystem {
	return &WeatherSystem{world: ws}
}

func (s *WeatherSystem) Phase() coresys.Phase { return coresys.PhasePostUpdate }

func (s *WeatherSystem) Update(_ time.Duration) {
	gt := world.GameTimeNow()
	curHour := gt.Hour()
	if s.world.LastHour < 0 {
		// First tick — initialize without broadcast
		s.world.LastHour = curHour
		s.world.RandomizeWeather()
		return
	}
	if curHour != s.world.LastHour {
		s.world.LastHour = curHour
		s.world.RandomizeWeather()
		weather := s.world.Weather
		s.world.AllPlayers(func(p *world.PlayerInfo) {
			w := packet.NewWriterWithOpcode(packet.S_OPCODE_WEATHER)
			w.WriteC(weather)
			p.Session.Send(w.Bytes())
		})
	}
}
