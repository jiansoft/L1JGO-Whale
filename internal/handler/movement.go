package handler

import (
	"time"

	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/net/packet"
	"github.com/l1jgo/server/internal/world"
)

// Direction deltas indexed by heading (0-7).
var headingDX = [8]int32{0, 1, 1, 1, 0, -1, -1, -1}
var headingDY = [8]int32{-1, -1, 0, 1, 1, 1, 0, -1}

// HandleMove processes C_MOVE (opcode 29).
// Taiwan 3.80C client: heading XOR'd with 0x49, sends current X/Y.
// Java (Taiwan, CLIENT_LANGUAGE == 3) IGNORES client X/Y and always uses
// server-tracked position: locx = pc.getX(); locy = pc.getY();
// We do the same — InQueue is blocking so no packets are dropped.
func HandleMove(sess *net.Session, r *packet.Reader, deps *Deps) {
	_ = r.ReadH() // client X (ignored — Taiwan client offset differs from server)
	_ = r.ReadH() // client Y (ignored)
	rawHeading := r.ReadC()

	// Taiwan client: heading XOR'd with 0x49
	heading := int16(rawHeading ^ 0x49)

	if heading < 0 || heading > 7 {
		return
	}

	ws := deps.World
	player := ws.GetBySession(sess.ID)
	if player == nil {
		return
	}

	// 麻痺/暈眩/凍結/睡眠時無法移動（客戶端已鎖定，這裡做伺服器端防護）
	if player.Paralyzed || player.Sleeped {
		return
	}

	// --- 移動速度驗證（反加速外掛） ---
	// 一般走路 ~200ms，加速 ~133ms。套用 80% 容許值。
	now := time.Now().UnixNano()
	minInterval := int64(160_000_000) // 200ms * 80% = 160ms
	if player.MoveSpeed == 1 {
		minInterval = 106_000_000 // 133ms * 80% = 106ms
	}
	if player.LastMoveTime > 0 && (now-player.LastMoveTime) < minInterval {
		rejectMove(sess, player, ws, deps)
		return
	}
	player.LastMoveTime = now

	// 永遠使用伺服器端座標（與 Java 台版行為一致）
	curX := player.X
	curY := player.Y

	// 從目前位置 + 朝向計算目的地
	destX := curX + headingDX[heading]
	destY := curY + headingDY[heading]

	// 地形通行性檢查（含 tileImpassable 0x80 動態標記）
	if deps.MapData != nil && !deps.MapData.IsPassable(player.MapID, curX, curY, int(heading)) {
		rejectMove(sess, player, ws, deps)
		return
	}
	// EntityGrid 備援檢查（地圖資料不可用時的安全網）
	if ws.IsOccupied(destX, destY, player.MapID, player.CharID) {
		rejectMove(sess, player, ws, deps)
		return
	}

	// Java C_MoveChar: marks old tile passable, new tile impassable (bit 0x80).
	// This prevents NPCs from walking into player tiles (NPC checks isPassable).
	if deps.MapData != nil {
		deps.MapData.SetImpassable(player.MapID, curX, curY, false)
	}

	// Get old nearby set BEFORE moving
	oldNearby := ws.GetNearbyPlayers(curX, curY, player.MapID, sess.ID)

	// Update position to DESTINATION
	ws.UpdatePosition(sess.ID, destX, destY, player.MapID, heading)

	// Mark new position as impassable (for NPC pathfinding)
	if deps.MapData != nil {
		deps.MapData.SetImpassable(player.MapID, destX, destY, true)
	}

	// Auto-cancel trade if partner is too far (> 15 tiles or different map)
	if player.TradePartnerID != 0 {
		partner := deps.World.GetByCharID(player.TradePartnerID)
		if partner != nil {
			tdx := destX - partner.X
			tdy := destY - partner.Y
			if tdx < 0 {
				tdx = -tdx
			}
			if tdy < 0 {
				tdy = -tdy
			}
			dist := tdx
			if tdy > dist {
				dist = tdy
			}
			if dist > 15 || player.MapID != partner.MapID {
				cancelTradeIfActive(player, deps)
			}
		} else {
			cancelTradeIfActive(player, deps)
		}
	}

	// Get new nearby set AFTER moving
	newNearby := ws.GetNearbyPlayers(destX, destY, player.MapID, sess.ID)

	// Build lookup sets for diffing
	oldSet := make(map[uint64]struct{}, len(oldNearby))
	for _, p := range oldNearby {
		oldSet[p.SessionID] = struct{}{}
	}
	newSet := make(map[uint64]struct{}, len(newNearby))
	for _, p := range newNearby {
		newSet[p.SessionID] = struct{}{}
	}

	// 1. 雙方都在視野內：發送移動封包 + 更新格子封鎖
	for _, other := range newNearby {
		if _, wasOld := oldSet[other.SessionID]; wasOld {
			sendMoveObject(other.Session, player.CharID, curX, curY, heading)
			// 對方看我：解鎖舊格 + 封鎖新格
			SendEntityTileUnblock(other.Session, curX, curY)
			SendEntityTileBlock(other.Session, destX, destY)
			// 我看對方：對方位置沒變，進入視野時已封鎖 → 不需處理
		}
	}

	// 2. 新進入視野：互相出現 + 封鎖格子
	for _, other := range newNearby {
		if _, wasOld := oldSet[other.SessionID]; !wasOld {
			sendPutObject(sess, other)
			SendEntityTileBlock(sess, other.X, other.Y)
			sendPutObject(other.Session, player)
			SendEntityTileBlock(other.Session, destX, destY)
		}
	}

	// 3. 離開視野：互相移除 + 解鎖格子
	for _, other := range oldNearby {
		if _, isNew := newSet[other.SessionID]; !isNew {
			sendRemoveObject(sess, other.CharID)
			SendEntityTileUnblock(sess, other.X, other.Y)
			sendRemoveObject(other.Session, player.CharID)
			SendEntityTileUnblock(other.Session, curX, curY)
		}
	}

	// --- NPC AOI ---
	oldNpcs := ws.GetNearbyNpcs(curX, curY, player.MapID)
	newNpcs := ws.GetNearbyNpcs(destX, destY, player.MapID)

	oldNpcSet := make(map[int32]struct{}, len(oldNpcs))
	for _, n := range oldNpcs {
		oldNpcSet[n.ID] = struct{}{}
	}
	newNpcSet := make(map[int32]struct{}, len(newNpcs))
	for _, n := range newNpcs {
		newNpcSet[n.ID] = struct{}{}
	}

	for _, n := range newNpcs {
		if _, wasOld := oldNpcSet[n.ID]; !wasOld {
			// 新進入視野：顯示 + 封鎖格子
			sendNpcPack(sess, n)
			SendEntityTileBlock(sess, n.X, n.Y)
		}
		// 持續在視野內：進入視野時已封鎖 → 不需處理
	}
	for _, n := range oldNpcs {
		if _, isNew := newNpcSet[n.ID]; !isNew {
			sendRemoveObject(sess, n.ID)
			SendEntityTileUnblock(sess, n.X, n.Y)
		}
	}

	// --- 召喚獸 AOI ---
	oldSummons := ws.GetNearbySummons(curX, curY, player.MapID)
	newSummons := ws.GetNearbySummons(destX, destY, player.MapID)
	oldSumSet := make(map[int32]struct{}, len(oldSummons))
	for _, s := range oldSummons {
		oldSumSet[s.ID] = struct{}{}
	}
	for _, s := range newSummons {
		if _, wasOld := oldSumSet[s.ID]; !wasOld {
			isOwner := s.OwnerCharID == player.CharID
			masterName := ""
			if m := ws.GetByCharID(s.OwnerCharID); m != nil {
				masterName = m.Name
			}
			sendSummonPack(sess, s, isOwner, masterName)
			SendEntityTileBlock(sess, s.X, s.Y)
		}
		// 持續在視野內：進入視野時已封鎖 → 不需處理
	}
	newSumSet := make(map[int32]struct{}, len(newSummons))
	for _, s := range newSummons {
		newSumSet[s.ID] = struct{}{}
	}
	for _, s := range oldSummons {
		if _, isNew := newSumSet[s.ID]; !isNew {
			sendRemoveObject(sess, s.ID)
			SendEntityTileUnblock(sess, s.X, s.Y)
		}
	}

	// --- 魔法娃娃 AOI ---
	oldDolls := ws.GetNearbyDolls(curX, curY, player.MapID)
	newDolls := ws.GetNearbyDolls(destX, destY, player.MapID)
	oldDollSet := make(map[int32]struct{}, len(oldDolls))
	for _, d := range oldDolls {
		oldDollSet[d.ID] = struct{}{}
	}
	for _, d := range newDolls {
		if _, wasOld := oldDollSet[d.ID]; !wasOld {
			masterName := ""
			if m := ws.GetByCharID(d.OwnerCharID); m != nil {
				masterName = m.Name
			}
			sendDollPack(sess, d, masterName)
			SendEntityTileBlock(sess, d.X, d.Y)
		}
		// 持續在視野內：進入視野時已封鎖 → 不需處理
	}
	newDollSet := make(map[int32]struct{}, len(newDolls))
	for _, d := range newDolls {
		newDollSet[d.ID] = struct{}{}
	}
	for _, d := range oldDolls {
		if _, isNew := newDollSet[d.ID]; !isNew {
			sendRemoveObject(sess, d.ID)
			SendEntityTileUnblock(sess, d.X, d.Y)
		}
	}

	// --- 隨從 AOI ---
	oldFollow := ws.GetNearbyFollowers(curX, curY, player.MapID)
	newFollow := ws.GetNearbyFollowers(destX, destY, player.MapID)
	oldFollowSet := make(map[int32]struct{}, len(oldFollow))
	for _, f := range oldFollow {
		oldFollowSet[f.ID] = struct{}{}
	}
	for _, f := range newFollow {
		if _, wasOld := oldFollowSet[f.ID]; !wasOld {
			sendFollowerPack(sess, f)
			SendEntityTileBlock(sess, f.X, f.Y)
		}
		// 持續在視野內：進入視野時已封鎖 → 不需處理
	}
	newFollowSet := make(map[int32]struct{}, len(newFollow))
	for _, f := range newFollow {
		newFollowSet[f.ID] = struct{}{}
	}
	for _, f := range oldFollow {
		if _, isNew := newFollowSet[f.ID]; !isNew {
			sendRemoveObject(sess, f.ID)
			SendEntityTileUnblock(sess, f.X, f.Y)
		}
	}

	// --- 寵物 AOI ---
	oldPets := ws.GetNearbyPets(curX, curY, player.MapID)
	newPets := ws.GetNearbyPets(destX, destY, player.MapID)
	oldPetSet := make(map[int32]struct{}, len(oldPets))
	for _, p := range oldPets {
		oldPetSet[p.ID] = struct{}{}
	}
	for _, p := range newPets {
		if _, wasOld := oldPetSet[p.ID]; !wasOld {
			isOwner := p.OwnerCharID == player.CharID
			masterName := ""
			if m := ws.GetByCharID(p.OwnerCharID); m != nil {
				masterName = m.Name
			}
			sendPetPack(sess, p, isOwner, masterName)
			SendEntityTileBlock(sess, p.X, p.Y)
		}
		// 持續在視野內：進入視野時已封鎖 → 不需處理
	}
	newPetSet := make(map[int32]struct{}, len(newPets))
	for _, p := range newPets {
		newPetSet[p.ID] = struct{}{}
	}
	for _, p := range oldPets {
		if _, isNew := newPetSet[p.ID]; !isNew {
			sendRemoveObject(sess, p.ID)
			SendEntityTileUnblock(sess, p.X, p.Y)
		}
	}

	// --- Ground item AOI: show/hide ground items as player moves ---
	oldGnd := ws.GetNearbyGroundItems(curX, curY, player.MapID)
	newGnd := ws.GetNearbyGroundItems(destX, destY, player.MapID)

	oldGndSet := make(map[int32]struct{}, len(oldGnd))
	for _, g := range oldGnd {
		oldGndSet[g.ID] = struct{}{}
	}
	newGndSet := make(map[int32]struct{}, len(newGnd))
	for _, g := range newGnd {
		newGndSet[g.ID] = struct{}{}
	}

	for _, g := range newGnd {
		if _, wasOld := oldGndSet[g.ID]; !wasOld {
			sendDropItem(sess, g)
		}
	}
	for _, g := range oldGnd {
		if _, isNew := newGndSet[g.ID]; !isNew {
			sendRemoveObject(sess, g.ID)
		}
	}

	// --- Door AOI: show/hide doors as player moves ---
	oldDoors := ws.GetNearbyDoors(curX, curY, player.MapID)
	newDoors := ws.GetNearbyDoors(destX, destY, player.MapID)

	oldDoorSet := make(map[int32]struct{}, len(oldDoors))
	for _, d := range oldDoors {
		oldDoorSet[d.ID] = struct{}{}
	}
	newDoorSet := make(map[int32]struct{}, len(newDoors))
	for _, d := range newDoors {
		newDoorSet[d.ID] = struct{}{}
	}

	// Doors newly visible
	for _, d := range newDoors {
		if _, wasOld := oldDoorSet[d.ID]; !wasOld {
			SendDoorPerceive(sess, d)
		}
	}
	// Doors no longer visible
	for _, d := range oldDoors {
		if _, isNew := newDoorSet[d.ID]; !isNew {
			sendRemoveObject(sess, d.ID)
		}
	}

}

// rejectMove 碰撞拒絕：回彈玩家位置 + 重發所有附近實體。
// 對應 Java L1PcUnlock.Pc_Unlock() 流程：
//   S_OwnCharPack → removeAllKnownObjects → updateObject → S_CharVisualUpdate
// 只發 S_OwnCharPack 會讓客戶端清除附近物件渲染，必須立即重發所有可見實體。
func rejectMove(sess *net.Session, player *world.PlayerInfo, ws *world.State, deps *Deps) {
	// 1. 回彈：告知客戶端正確座標
	sendOwnCharPackPlayer(sess, player)

	px, py := player.X, player.Y

	// 2. 重發所有附近玩家 + 封鎖格子
	nearbyPlayers := ws.GetNearbyPlayers(px, py, player.MapID, sess.ID)
	for _, other := range nearbyPlayers {
		sendPutObject(sess, other)
		SendEntityTileBlock(sess, other.X, other.Y)
	}

	// 3. 重發所有附近 NPC + 封鎖格子
	nearbyNpcs := ws.GetNearbyNpcs(px, py, player.MapID)
	for _, n := range nearbyNpcs {
		sendNpcPack(sess, n)
		SendEntityTileBlock(sess, n.X, n.Y)
	}

	// 4. 重發所有附近召喚獸 + 封鎖格子
	nearbySummons := ws.GetNearbySummons(px, py, player.MapID)
	for _, s := range nearbySummons {
		isOwner := s.OwnerCharID == player.CharID
		masterName := ""
		if m := ws.GetByCharID(s.OwnerCharID); m != nil {
			masterName = m.Name
		}
		sendSummonPack(sess, s, isOwner, masterName)
		SendEntityTileBlock(sess, s.X, s.Y)
	}

	// 5. 重發所有附近魔法娃娃 + 封鎖格子
	nearbyDolls := ws.GetNearbyDolls(px, py, player.MapID)
	for _, d := range nearbyDolls {
		masterName := ""
		if m := ws.GetByCharID(d.OwnerCharID); m != nil {
			masterName = m.Name
		}
		sendDollPack(sess, d, masterName)
		SendEntityTileBlock(sess, d.X, d.Y)
	}

	// 6. 重發所有附近隨從 + 封鎖格子
	nearbyFollowers := ws.GetNearbyFollowers(px, py, player.MapID)
	for _, f := range nearbyFollowers {
		sendFollowerPack(sess, f)
		SendEntityTileBlock(sess, f.X, f.Y)
	}

	// 7. 重發所有附近寵物 + 封鎖格子
	nearbyPets := ws.GetNearbyPets(px, py, player.MapID)
	for _, p := range nearbyPets {
		isOwner := p.OwnerCharID == player.CharID
		masterName := ""
		if m := ws.GetByCharID(p.OwnerCharID); m != nil {
			masterName = m.Name
		}
		sendPetPack(sess, p, isOwner, masterName)
		SendEntityTileBlock(sess, p.X, p.Y)
	}

	// 8. 重發所有附近地面物品（地面物品不佔格子，不需封鎖）
	nearbyGnd := ws.GetNearbyGroundItems(player.X, player.Y, player.MapID)
	for _, g := range nearbyGnd {
		sendDropItem(sess, g)
	}

	// 9. 重發所有附近門
	nearbyDoors := ws.GetNearbyDoors(player.X, player.Y, player.MapID)
	for _, d := range nearbyDoors {
		SendDoorPerceive(sess, d)
	}

	// 10. 武器/變身視覺更新（Java 流程最後一步）
	sendCharVisualUpdate(sess, player)
}

// HandleChangeDirection processes C_CHANGE_DIRECTION (opcode 225).
// NOTE: Unlike C_MOVE, C_ChangeHeading does NOT XOR heading with 0x49 — raw value.
func HandleChangeDirection(sess *net.Session, r *packet.Reader, deps *Deps) {
	heading := int16(r.ReadC())
	if heading < 0 || heading > 7 {
		return
	}

	player := deps.World.GetBySession(sess.ID)
	if player == nil {
		return
	}
	player.Heading = heading

	// Broadcast direction change to nearby players
	nearby := deps.World.GetNearbyPlayers(player.X, player.Y, player.MapID, sess.ID)
	for _, other := range nearby {
		sendChangeHeading(other.Session, player.CharID, heading)
	}
}
