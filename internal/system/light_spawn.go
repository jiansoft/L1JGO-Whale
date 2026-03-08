package system

import (
	"log"
	"time"

	coresys "github.com/l1jgo/server/internal/core/system"
	"github.com/l1jgo/server/internal/data"
	"github.com/l1jgo/server/internal/world"
)

// LightSpawnSystem 路燈生成系統 — 夜晚生成光源 NPC，白天移除。
// Phase 3（PostUpdate），與 WeatherSystem 同階段。
type LightSpawnSystem struct {
	state       *world.State
	spawns      []data.LightSpawnEntry
	npcTable    *data.NpcTable
	lastIsNight bool
	initialized bool
	lightNpcIDs []int32 // 已生成的路燈 NPC object ID
}

func NewLightSpawnSystem(ws *world.State, spawns []data.LightSpawnEntry, npcTable *data.NpcTable) *LightSpawnSystem {
	return &LightSpawnSystem{
		state:    ws,
		spawns:   spawns,
		npcTable: npcTable,
	}
}

func (s *LightSpawnSystem) Phase() coresys.Phase { return coresys.PhasePostUpdate }

func (s *LightSpawnSystem) Update(_ time.Duration) {
	isNight := world.GameTimeNow().IsNight()
	if !s.initialized {
		s.initialized = true
		s.lastIsNight = isNight
		if isNight {
			s.spawnLights()
		}
		return
	}
	if isNight == s.lastIsNight {
		return
	}
	s.lastIsNight = isNight
	if isNight {
		s.spawnLights()
	} else {
		s.removeLights()
	}
}

// spawnLights 生成所有路燈 NPC。
func (s *LightSpawnSystem) spawnLights() {
	s.lightNpcIDs = make([]int32, 0, len(s.spawns))
	for i := range s.spawns {
		sp := &s.spawns[i]
		tmpl := s.npcTable.Get(sp.NpcID)
		if tmpl == nil {
			continue
		}
		npc := &world.NpcInfo{
			ID:        world.NextNpcID(),
			NpcID:     tmpl.NpcID,
			Impl:      tmpl.Impl,
			GfxID:     tmpl.GfxID,
			LightSize: byte(tmpl.LightSize),
			Name:      tmpl.Name,
			NameID:    tmpl.NameID,
			X:         sp.X,
			Y:         sp.Y,
			MapID:     sp.MapID,
			MaxHP:     1,
			HP:        1,
		}
		s.state.AddNpc(npc)
		s.lightNpcIDs = append(s.lightNpcIDs, npc.ID)
	}
	log.Printf("[路燈] 生成 %d 盞路燈 NPC", len(s.lightNpcIDs))
}

// removeLights 移除所有路燈 NPC。
// VisibilitySystem 會自動對附近玩家發送 S_RemoveObject。
func (s *LightSpawnSystem) removeLights() {
	log.Printf("[路燈] 移除 %d 盞路燈 NPC", len(s.lightNpcIDs))
	for _, id := range s.lightNpcIDs {
		s.state.RemoveNpc(id)
	}
	s.lightNpcIDs = nil
}
