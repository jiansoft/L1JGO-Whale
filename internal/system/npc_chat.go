package system

import (
	"time"

	coresys "github.com/l1jgo/server/internal/core/system"
	"github.com/l1jgo/server/internal/data"
	"github.com/l1jgo/server/internal/handler"
	"github.com/l1jgo/server/internal/net/packet"
	"github.com/l1jgo/server/internal/world"
)

// NpcChatSystem 處理 NPC 定時聊天（Phase 3 後置更新）。
// Java: L1NpcChatTimer — 依 tick 驅動，非獨立線程。
type NpcChatSystem struct {
	world *world.State
	deps  *handler.Deps
}

func NewNpcChatSystem(ws *world.State, deps *handler.Deps) *NpcChatSystem {
	return &NpcChatSystem{world: ws, deps: deps}
}

func (s *NpcChatSystem) Phase() coresys.Phase { return coresys.PhasePostUpdate }

// msToTicks 將毫秒轉換為 tick 數（200ms/tick）。
func msToTicks(ms int) int {
	t := ms / 200
	if t < 1 && ms > 0 {
		t = 1
	}
	return t
}

// StartNpcChat 啟動 NPC 聊天計時器（指定時機）。
// 在 NPC 出現/死亡/被攻擊時呼叫。
func StartNpcChat(npc *world.NpcInfo, timing int, chatTable *data.NpcChatTable) {
	if chatTable == nil {
		return
	}
	chat := chatTable.Get(npc.NpcID, timing)
	if chat == nil {
		return
	}
	// UNDERATTACK 只觸發一次（首次被攻擊）
	if timing == data.ChatTimingUnderAttack {
		if npc.ChatFirstAttack {
			return
		}
		npc.ChatFirstAttack = true
	}

	ids := chat.ChatIDs()
	if len(ids) == 0 {
		return
	}

	npc.ChatActive = true
	npc.ChatTiming = timing
	npc.ChatStep = 0
	npc.ChatIntervalTicks = 0
	npc.ChatRepeatTicks = 0

	if chat.StartDelayTime > 0 {
		npc.ChatDelayTicks = msToTicks(chat.StartDelayTime)
	} else {
		npc.ChatDelayTicks = 0
	}
}

// StopNpcChat 停止 NPC 聊天計時器。
func StopNpcChat(npc *world.NpcInfo) {
	npc.ChatActive = false
	npc.ChatStep = 0
	npc.ChatDelayTicks = 0
	npc.ChatIntervalTicks = 0
	npc.ChatRepeatTicks = 0
}

func (s *NpcChatSystem) Update(_ time.Duration) {
	if s.deps.NpcChats == nil {
		return
	}
	for _, npc := range s.world.NpcList() {
		// 自動啟動出現聊天 — 活著且尚未啟動聊天計時器的 NPC
		if !npc.Dead && !npc.ChatActive && !npc.ChatAppearStarted {
			npc.ChatAppearStarted = true
			StartNpcChat(npc, data.ChatTimingAppearance, s.deps.NpcChats)
		}
		if !npc.ChatActive {
			continue
		}
		s.tickNpcChat(npc)
	}
}

func (s *NpcChatSystem) tickNpcChat(npc *world.NpcInfo) {
	// 使用目前啟用的聊天時機查找配置
	chat := s.deps.NpcChats.Get(npc.NpcID, npc.ChatTiming)
	if chat == nil {
		npc.ChatActive = false
		return
	}

	ids := chat.ChatIDs()
	if len(ids) == 0 {
		npc.ChatActive = false
		return
	}

	// 首次延遲倒數
	if npc.ChatDelayTicks > 0 {
		npc.ChatDelayTicks--
		return
	}

	// 重複間隔倒數（序列播完後等待下次）
	if npc.ChatRepeatTicks > 0 {
		npc.ChatRepeatTicks--
		return
	}

	// 對話間隔倒數
	if npc.ChatIntervalTicks > 0 {
		npc.ChatIntervalTicks--
		return
	}

	// 播放當前 step 的對話
	if npc.ChatStep < len(ids) {
		chatID := ids[npc.ChatStep]
		s.broadcastNpcChat(npc, chat, chatID)
		npc.ChatStep++

		// 還有下一段 → 設定間隔計時
		if npc.ChatStep < len(ids) && chat.ChatInterval > 0 {
			npc.ChatIntervalTicks = msToTicks(chat.ChatInterval)
		}
	}

	// 序列播完
	if npc.ChatStep >= len(ids) {
		if chat.IsRepeat && chat.RepeatInterval > 0 {
			// 重置序列，設定重複間隔
			npc.ChatStep = 0
			npc.ChatRepeatTicks = msToTicks(chat.RepeatInterval)
		} else {
			// 不重複 → 停止
			npc.ChatActive = false
		}
	}
}

// broadcastNpcChat 廣播 NPC 聊天封包。
// Java: S_NpcChat（type=0）、S_NpcChatShouting（type=2）、S_NpcChatGlobal（type=3）
// 格式：writeC(161) + writeC(type) + writeD(npcID) + writeS(text)
func (s *NpcChatSystem) broadcastNpcChat(npc *world.NpcInfo, chat *data.NpcChat, chatID string) {
	nameID := npc.NameID
	if nameID == "" {
		nameID = npc.Name
	}

	if chat.IsWorldChat {
		// 世界聊天 — 發送給所有線上玩家
		text := "[" + nameID + "] " + chatID
		pkt := buildNpcChatPacket(npc.ID, 3, text)
		s.world.AllPlayers(func(p *world.PlayerInfo) {
			if p.Session != nil {
				p.Session.Send(pkt)
			}
		})
	}

	if chat.IsShout {
		// 大喊 — 廣播到附近（較大範圍）
		text := "<" + nameID + "> " + chatID
		pkt := buildNpcChatPacket(npc.ID, 2, text)
		nearby := s.world.GetNearbyPlayersAt(npc.X, npc.Y, npc.MapID)
		for _, p := range nearby {
			if p.Session != nil {
				p.Session.Send(pkt)
			}
		}
	} else if !chat.IsWorldChat {
		// 一般聊天 — 廣播到附近
		text := nameID + ": " + chatID
		pkt := buildNpcChatPacket(npc.ID, 0, text)
		nearby := s.world.GetNearbyPlayersAt(npc.X, npc.Y, npc.MapID)
		for _, p := range nearby {
			if p.Session != nil {
				p.Session.Send(pkt)
			}
		}
	}
}

// buildNpcChatPacket 建構 NPC 聊天封包。
// 格式：writeC(S_OPCODE_NPCSHOUT=161) + writeC(type) + writeD(npcID) + writeS(text)
func buildNpcChatPacket(npcID int32, chatType byte, text string) []byte {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_NPCSHOUT)
	w.WriteC(chatType)
	w.WriteD(npcID)
	w.WriteS(text)
	return w.Bytes()
}
