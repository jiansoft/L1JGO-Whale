package handler

import (
	"fmt"
	"time"

	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/net/packet"
	"github.com/l1jgo/server/internal/world"
)

// --- 城堡管理封包 ---

// HandleTaxRate 處理 C_TaxRate (opcode 19) — 設定城堡稅率。
// Java 參考: C_TaxRate.java
// 封包格式: readD(charID) + readC(taxRate)
func HandleTaxRate(sess *net.Session, r *packet.Reader, deps *Deps) {
	player := deps.World.GetBySession(sess.ID)
	if player == nil || player.Dead {
		return
	}

	_ = r.ReadD() // charID（略過）
	rate := int32(r.ReadC())

	if deps.Castle == nil {
		return
	}

	// 找出玩家所屬城堡
	castleInfo := findPlayerCastle(player, deps)
	if castleInfo == nil {
		return
	}

	deps.Castle.SetTaxRate(sess, player, castleInfo.CastleID, rate)
}

// HandleCastleDeposit 處理 C_Deposit (opcode 56) — 存入城堡寶庫。
// Java 參考: C_Deposit.java
// 封包格式: readD(charID) + readD(amount)
func HandleCastleDeposit(sess *net.Session, r *packet.Reader, deps *Deps) {
	player := deps.World.GetBySession(sess.ID)
	if player == nil || player.Dead {
		return
	}

	_ = r.ReadD() // charID
	amount := r.ReadD()

	if deps.Castle == nil {
		return
	}

	castleInfo := findPlayerCastle(player, deps)
	if castleInfo == nil {
		return
	}

	deps.Castle.Deposit(sess, player, castleInfo.CastleID, amount)
}

// HandleCastleWithdraw 處理 C_Withdraw (opcode 44) — 從城堡寶庫領出。
// Java 參考: C_Drawal.java
// 封包格式: readD(charID) + readD(amount)
func HandleCastleWithdraw(sess *net.Session, r *packet.Reader, deps *Deps) {
	player := deps.World.GetBySession(sess.ID)
	if player == nil || player.Dead {
		return
	}

	_ = r.ReadD() // charID
	amount := r.ReadD()

	if deps.Castle == nil {
		return
	}

	castleInfo := findPlayerCastle(player, deps)
	if castleInfo == nil {
		return
	}

	deps.Castle.Withdraw(sess, player, castleInfo.CastleID, amount)
}

// findPlayerCastle 找出玩家公會持有的城堡。
func findPlayerCastle(player *world.PlayerInfo, deps *Deps) *CastleInfo {
	if player.ClanID == 0 || deps.Castle == nil {
		return nil
	}
	return deps.Castle.GetCastleByOwnerClan(player.ClanID)
}

// --- 城堡封包建構 ---

// SendTaxRateUI 發送稅率設定 UI（S_TaxRate）。
// Java: S_TaxRate — writeC(opcode) + writeD(charID) + writeC(10) + writeC(50)
func SendTaxRateUI(sess *net.Session, charID int32) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_TAXRATE)
	w.WriteD(charID)
	w.WriteC(10) // 最小稅率
	w.WriteC(50) // 最大稅率
	sess.Send(w.Bytes())
}

// SendDepositUI 發送城堡寶庫存入 UI（S_Deposit）。
// Java: S_Deposit — writeC(opcode) + writeD(charID)
func SendDepositUI(sess *net.Session, charID int32) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_DEPOSIT)
	w.WriteD(charID)
	sess.Send(w.Bytes())
}

// SendDrawalUI 發送城堡寶庫領出 UI（S_Drawal）。
// Java: S_Drawal — writeC(opcode) + writeD(charID) + writeD(min(money, 2000000000))
func SendDrawalUI(sess *net.Session, charID int32, publicMoney int64) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_DRAWAL)
	w.WriteD(charID)
	displayMoney := publicMoney
	if displayMoney > 2_000_000_000 {
		displayMoney = 2_000_000_000
	}
	w.WriteD(int32(displayMoney))
	sess.Send(w.Bytes())
}

// SendCastleMaster 發送城主皇冠標誌（S_CastleMaster）。
// Java: S_CastleMaster — writeC(opcode) + writeC(castleID) + writeD(charID)
func SendCastleMaster(sess *net.Session, castleID int32, charID int32) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_CASTLEMASTER)
	w.WriteC(byte(castleID))
	w.WriteD(charID)
	sess.Send(w.Bytes())
}

// BuildCastleMaster 建構城主皇冠標誌封包（用於廣播）。
func BuildCastleMaster(castleID int32, charID int32) []byte {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_CASTLEMASTER)
	w.WriteC(byte(castleID))
	w.WriteD(charID)
	return w.Bytes()
}

// BuildPacketBoxWar 建構攻城戰訊息封包。
// subCode: 0=開始(639), 1=結束(640), 2=進行中(641), 3=掌握主導權(642), 4=佔領(643)
// Java: S_PacketBoxWar — writeC(S_OPCODE_EVENT) + writeC(subCode) + writeC(castleID) + writeH(0)
func BuildPacketBoxWar(subCode byte, castleID int32) []byte {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_EVENT)
	w.WriteC(subCode)
	w.WriteC(byte(castleID))
	w.WriteH(0)
	return w.Bytes()
}

// BroadcastPacketBoxWar 向所有線上玩家廣播攻城戰訊息。
func BroadcastPacketBoxWar(worldState *world.State, subCode byte, castleID int32) {
	data := BuildPacketBoxWar(subCode, castleID)
	worldState.AllPlayers(func(p *world.PlayerInfo) {
		p.Session.Send(data)
	})
}

// SendPacketBoxWarToPlayer 向單一玩家發送攻城戰狀態訊息。
func SendPacketBoxWarToPlayer(sess *net.Session, subCode byte, castleID int32) {
	data := BuildPacketBoxWar(subCode, castleID)
	sess.Send(data)
}

// BuildWarPacket 建構戰爭訊息封包（S_War）。
// Java: S_War — writeC(opcode) + writeC(type) + writeS(clan1) + writeS(clan2)
// type: 1=宣戰(226), 2=投降(228), 3=結束(227), 4=勝利(231), 6=結盟(224), 8=進行中
func BuildWarPacket(warType byte, clan1, clan2 string) []byte {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_WAR)
	w.WriteC(warType)
	w.WriteS(clan1)
	w.WriteS(clan2)
	return w.Bytes()
}

// BroadcastWarPacket 向所有線上玩家廣播戰爭訊息。
func BroadcastWarPacket(worldState *world.State, warType byte, clan1, clan2 string) {
	data := BuildWarPacket(warType, clan1, clan2)
	worldState.AllPlayers(func(p *world.PlayerInfo) {
		p.Session.Send(data)
	})
}

// Itoa 整數轉字串輔助函式（用於 SendServerMessageArgs 參數）。
func Itoa(v int32) string {
	return fmt.Sprintf("%d", v)
}

// HandleWar 處理 C_War (opcode 227) — 宣戰/投降/休戰。
// Java 參考: C_War.java
// 封包格式: readC(type) + readS(targetClanName)
// type: 0=宣戰, 2=投降, 3=休戰
func HandleWar(sess *net.Session, r *packet.Reader, deps *Deps) {
	player := deps.World.GetBySession(sess.ID)
	if player == nil || player.Dead {
		return
	}

	warType := r.ReadC()
	targetClanName := r.ReadS()

	if deps.War == nil {
		return
	}

	switch warType {
	case 0: // 宣戰
		deps.War.DeclareWar(sess, player, targetClanName)
	case 2: // 投降
		deps.War.SurrenderWar(sess, player, targetClanName)
	case 3: // 休戰
		deps.War.CeaseWar(sess, player, targetClanName)
	}
}

// --- NPC 動作處理 ---

// handleCastleInex 查詢城堡資金（Java: "inex" 動作 → S_ServerMessage 309）。
func handleCastleInex(sess *net.Session, player *world.PlayerInfo, deps *Deps) {
	ci := findPlayerCastle(player, deps)
	if ci == nil {
		return
	}
	money := ci.PublicMoney
	if money > 2_000_000_000 {
		money = 2_000_000_000
	}
	sendServerMessageArgs(sess, 309, fmt.Sprintf("%d", money))
}

// handleCastleTax 開啟稅率設定 UI。
func handleCastleTax(sess *net.Session, player *world.PlayerInfo, deps *Deps) {
	ci := findPlayerCastle(player, deps)
	if ci == nil {
		return
	}
	SendTaxRateUI(sess, player.CharID)
}

// handleCastleWithdrawal 開啟寶庫領出視窗。
func handleCastleWithdrawal(sess *net.Session, player *world.PlayerInfo, deps *Deps) {
	ci := findPlayerCastle(player, deps)
	if ci == nil {
		return
	}
	SendDrawalUI(sess, player.CharID, ci.PublicMoney)
}

// handleCastleDeposit 開啟寶庫存入視窗。
func handleCastleDeposit(sess *net.Session, player *world.PlayerInfo, deps *Deps) {
	ci := findPlayerCastle(player, deps)
	if ci == nil {
		return
	}
	SendDepositUI(sess, player.CharID)
}

// handleCastleGateRepair 修復城門（非攻城戰時）。
func handleCastleGateRepair(sess *net.Session, player *world.PlayerInfo, deps *Deps) {
	ci := findPlayerCastle(player, deps)
	if ci == nil || deps.Castle == nil {
		return
	}
	if deps.Castle.IsWarNow(ci.CastleID) {
		return
	}
	if deps.Castles != nil {
		castleCfg := deps.Castles.Get(ci.CastleID)
		if castleCfg != nil {
			doors := deps.World.GetDoorsByMap(castleCfg.WarArea.Map)
			for _, door := range doors {
				if deps.Castles.CheckInWarArea(ci.CastleID, door.X, door.Y, door.MapID) {
					door.RepairGate()
				}
			}
		}
	}
	sendServerMessage(sess, 990)
}

// handleCastleGateOpen 開啟內城門。
func handleCastleGateOpen(sess *net.Session, player *world.PlayerInfo, deps *Deps) {
	ci := findPlayerCastle(player, deps)
	if ci == nil || deps.Castle == nil {
		return
	}
	if !deps.Castle.IsWarNow(ci.CastleID) {
		return
	}
	if deps.Castles != nil {
		castleCfg := deps.Castles.Get(ci.CastleID)
		if castleCfg != nil && castleCfg.InnerMap != 0 {
			doors := deps.World.GetDoorsByMap(castleCfg.InnerMap)
			for _, door := range doors {
				door.Open()
			}
		}
	}
}

// handleCastleGateClose 關閉內城門。
func handleCastleGateClose(sess *net.Session, player *world.PlayerInfo, deps *Deps) {
	ci := findPlayerCastle(player, deps)
	if ci == nil || deps.Castle == nil {
		return
	}
	if !deps.Castle.IsWarNow(ci.CastleID) {
		return
	}
	if deps.Castles != nil {
		castleCfg := deps.Castles.Get(ci.CastleID)
		if castleCfg != nil && castleCfg.InnerMap != 0 {
			doors := deps.World.GetDoorsByMap(castleCfg.InnerMap)
			for _, door := range doors {
				door.Close()
			}
		}
	}
}

// SendWarTime 發送 S_WarTime 封包（opcode 231）— 攻城戰時間設定 UI。
// Java: S_WarTime.java — 1997/01/01 17:00 為基點，每個 time 單位 = 182 分鐘。
func SendWarTime(sess *net.Session, warTime time.Time) {
	// 基點：1997/01/01 17:00 UTC+8
	baseCal := time.Date(1997, 1, 1, 17, 0, 0, 0, warTime.Location())
	diff := warTime.Sub(baseCal)

	// 減去 1200 分鐘的誤差修正（Java: diff -= 1200 * 60 * 1000）
	diff -= 1200 * time.Minute

	// 轉為分鐘，每個 time 單位 = 182 分鐘
	minutes := int32(diff.Minutes())
	timeUnit := minutes / 182

	w := packet.NewWriterWithOpcode(packet.S_OPCODE_WARTIME)
	w.WriteH(6) // 選項數量（最多 6）
	w.WriteS("GMT+8")
	w.WriteC(0)
	w.WriteC(0)
	w.WriteC(0)
	for i := int32(0); i < 6; i++ {
		w.WriteD(timeUnit - i)
		w.WriteC(0)
	}
	sess.Send(w.Bytes())
}
