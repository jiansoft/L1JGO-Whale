package system

import (
	"strings"
	"time"

	"github.com/l1jgo/server/internal/handler"
	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/world"
)

// WarSystem 處理戰爭邏輯（宣戰/投降/休戰/攻城勝利）。
// 實作 handler.WarManager 介面。
type WarSystem struct {
	deps *handler.Deps
	wars []*handler.ActiveWar // 進行中的戰爭
}

// NewWarSystem 建立戰爭系統。
func NewWarSystem(deps *handler.Deps) *WarSystem {
	return &WarSystem{
		deps: deps,
		wars: make([]*handler.ActiveWar, 0),
	}
}

// DeclareWar 宣戰（Java: C_War type=0）。
func (s *WarSystem) DeclareWar(sess *net.Session, player *world.PlayerInfo, targetClanName string) {
	// 驗證：必須是君主
	if player.ClassType != 0 {
		handler.SendServerMessage(sess, 478) // 只有君主才能宣戰
		return
	}
	// 驗證：必須有血盟且是盟主
	if player.ClanID == 0 {
		handler.SendServerMessage(sess, 272) // 沒有血盟
		return
	}
	attackClan := s.deps.World.Clans.GetClan(player.ClanID)
	if attackClan == nil || attackClan.LeaderID != player.CharID {
		handler.SendServerMessage(sess, 518) // 必須是盟主
		return
	}

	// 驗證：需要盟徽
	if attackClan.EmblemStatus == 0 {
		handler.SendServerMessage(sess, 1041) // 需要設定盟徽
		return
	}

	// 查找目標公會
	defenceClan := s.deps.World.Clans.GetClanByName(targetClanName)
	if defenceClan == nil {
		handler.SendServerMessage(sess, 218) // 找不到該血盟
		return
	}

	// 不能對自己宣戰
	if defenceClan.ClanID == attackClan.ClanID {
		return
	}

	// 判斷是攻城戰還是血盟戰
	if defenceClan.HasCastle > 0 {
		s.declareCastleWar(sess, player, attackClan, defenceClan)
	} else {
		s.declareSimWar(sess, player, attackClan, defenceClan)
	}
}

// declareCastleWar 宣布攻城戰。
func (s *WarSystem) declareCastleWar(sess *net.Session, player *world.PlayerInfo, attackClan, defenceClan *world.ClanInfo) {
	castleID := defenceClan.HasCastle

	// 驗證：君主等級 >= 25
	if player.Level < 25 {
		handler.SendServerMessage(sess, 478)
		return
	}

	// 驗證：攻擊方不能已有城堡
	if attackClan.HasCastle > 0 {
		handler.SendServerMessage(sess, 474) // 已擁有城堡
		return
	}

	// 驗證：必須在攻城戰期間
	if s.deps.Castle == nil || !s.deps.Castle.IsWarNow(castleID) {
		handler.SendServerMessage(sess, 182) // 攻城戰未開始
		return
	}

	// 檢查是否已經在該城堡的戰爭中
	for _, w := range s.wars {
		if w.CastleID == castleID && w.AttackClans[attackClan.ClanName] {
			handler.SendServerMessage(sess, 235) // 已在戰爭中
			return
		}
	}

	// 查找現有的攻城戰爭或建立新的
	var existingWar *handler.ActiveWar
	for _, w := range s.wars {
		if w.WarType == 1 && w.CastleID == castleID && strings.EqualFold(w.DefenceClan, defenceClan.ClanName) {
			existingWar = w
			break
		}
	}

	if existingWar != nil {
		// 加入攻擊方
		existingWar.AttackClans[attackClan.ClanName] = true
	} else {
		// 建立新攻城戰
		w := &handler.ActiveWar{
			WarType:     1,
			DefenceClan: defenceClan.ClanName,
			AttackClans: map[string]bool{attackClan.ClanName: true},
			CastleID:    castleID,
			StartTime:   time.Now(),
		}
		s.wars = append(s.wars, w)
	}

	// 廣播宣戰訊息（Java: S_War type=1 → msgID 226）
	handler.BroadcastWarPacket(s.deps.World, 1, attackClan.ClanName, defenceClan.ClanName)
}

// declareSimWar 宣布模擬戰（血盟戰）。
func (s *WarSystem) declareSimWar(sess *net.Session, player *world.PlayerInfo, attackClan, defenceClan *world.ClanInfo) {
	// 檢查是否已在戰爭中
	if s.IsWar(attackClan.ClanName, defenceClan.ClanName) {
		handler.SendServerMessage(sess, 235) // 已在戰爭中
		return
	}

	w := &handler.ActiveWar{
		WarType:     2,
		DefenceClan: defenceClan.ClanName,
		AttackClans: map[string]bool{attackClan.ClanName: true},
		CastleID:    0,
		StartTime:   time.Now(),
	}
	s.wars = append(s.wars, w)

	// 廣播宣戰訊息
	handler.BroadcastWarPacket(s.deps.World, 1, attackClan.ClanName, defenceClan.ClanName)
}

// SurrenderWar 投降（Java: C_War type=2）。
func (s *WarSystem) SurrenderWar(sess *net.Session, player *world.PlayerInfo, targetClanName string) {
	if player.ClanID == 0 {
		return
	}
	attackClan := s.deps.World.Clans.GetClan(player.ClanID)
	if attackClan == nil || attackClan.LeaderID != player.CharID {
		return
	}

	for i, w := range s.wars {
		if w.AttackClans[attackClan.ClanName] && strings.EqualFold(w.DefenceClan, targetClanName) {
			// 從攻擊方移除
			delete(w.AttackClans, attackClan.ClanName)

			// 廣播投降訊息（Java: S_War type=2 → msgID 228）
			handler.BroadcastWarPacket(s.deps.World, 2, attackClan.ClanName, targetClanName)

			// 如果沒有攻擊方了，移除戰爭
			if len(w.AttackClans) == 0 {
				s.wars = append(s.wars[:i], s.wars[i+1:]...)
			}
			return
		}
	}
}

// CeaseWar 休戰（Java: C_War type=3）。
func (s *WarSystem) CeaseWar(sess *net.Session, player *world.PlayerInfo, targetClanName string) {
	if player.ClanID == 0 {
		return
	}
	clan := s.deps.World.Clans.GetClan(player.ClanID)
	if clan == nil || clan.LeaderID != player.CharID {
		return
	}

	for i, w := range s.wars {
		clanInWar := w.AttackClans[clan.ClanName] || strings.EqualFold(w.DefenceClan, clan.ClanName)
		targetInWar := w.AttackClans[targetClanName] || strings.EqualFold(w.DefenceClan, targetClanName)

		if clanInWar && targetInWar {
			// 攻城戰不可休戰
			if w.WarType == 1 {
				handler.SendServerMessage(sess, 182)
				return
			}

			// 廣播休戰訊息（Java: S_War type=3 → msgID 227）
			handler.BroadcastWarPacket(s.deps.World, 3, clan.ClanName, targetClanName)

			// 移除戰爭
			s.wars = append(s.wars[:i], s.wars[i+1:]...)
			return
		}
	}
}

// WinCastleWar 攻城勝利（王冠被取得）。
func (s *WarSystem) WinCastleWar(winnerClanName string, castleID int32) {
	// 廣播勝利訊息（Java: S_War type=4 → msgID 231）
	for i := len(s.wars) - 1; i >= 0; i-- {
		w := s.wars[i]
		if w.WarType == 1 && w.CastleID == castleID {
			// 對所有參戰方廣播結束
			for attackClan := range w.AttackClans {
				handler.BroadcastWarPacket(s.deps.World, 3, attackClan, w.DefenceClan)
			}
			s.wars = append(s.wars[:i], s.wars[i+1:]...)
		}
	}

	// 廣播佔領城堡訊息
	handler.BroadcastPacketBoxWar(s.deps.World, 4, castleID) // MSG_WAR_OCCUPY
}

// CeaseCastleWar 攻城戰時間到（防禦方勝利）。
func (s *WarSystem) CeaseCastleWar(castleID int32) {
	for i := len(s.wars) - 1; i >= 0; i-- {
		w := s.wars[i]
		if w.WarType == 1 && w.CastleID == castleID {
			for attackClan := range w.AttackClans {
				handler.BroadcastWarPacket(s.deps.World, 3, attackClan, w.DefenceClan)
			}
			s.wars = append(s.wars[:i], s.wars[i+1:]...)
		}
	}
}

// IsWar 兩個公會是否在戰爭中。
func (s *WarSystem) IsWar(clan1, clan2 string) bool {
	for _, w := range s.wars {
		c1InWar := w.AttackClans[clan1] || strings.EqualFold(w.DefenceClan, clan1)
		c2InWar := w.AttackClans[clan2] || strings.EqualFold(w.DefenceClan, clan2)
		if c1InWar && c2InWar {
			return true
		}
	}
	return false
}

// IsClanInWar 公會是否在任何戰爭中。
func (s *WarSystem) IsClanInWar(clanName string) bool {
	for _, w := range s.wars {
		if w.AttackClans[clanName] || strings.EqualFold(w.DefenceClan, clanName) {
			return true
		}
	}
	return false
}

// GetActiveWars 取得所有進行中的戰爭。
func (s *WarSystem) GetActiveWars() []*handler.ActiveWar {
	return s.wars
}

// CheckCastleWar 登入時通知攻城戰狀態（Java: ServerWarExecutor.checkCastleWar）。
func (s *WarSystem) CheckCastleWar(sess *net.Session) {
	if s.deps.Castle == nil {
		return
	}
	for i := int32(1); i <= 8; i++ {
		if s.deps.Castle.IsWarNow(i) {
			// MSG_WAR_GOING = 2（Java: S_PacketBoxWar(MSG_WAR_GOING, castleId)）
			handler.SendPacketBoxWarToPlayer(sess, 2, i)
		}
	}
}
