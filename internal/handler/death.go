package handler

import (
	"fmt"

	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/net/packet"
	"github.com/l1jgo/server/internal/world"
)

// HandleRestart processes C_RESTART (opcode 177).
// When a dead player clicks restart, resurrect them at the nearest town.
func HandleRestart(sess *net.Session, _ *packet.Reader, deps *Deps) {
	player := deps.World.GetBySession(sess.ID)
	if player == nil {
		return
	}
	if !player.Dead {
		return
	}

	// Resurrect
	player.Dead = false
	player.LastMoveTime = 0 // reset speed validation
	player.HP = int16(player.Level) // Java: setCurrentHp(getLevel())
	if player.HP < 1 {
		player.HP = 1
	}
	if player.HP > player.MaxHP {
		player.HP = player.MaxHP
	}
	player.MP = int16(player.Level / 2)
	if player.MP > player.MaxMP {
		player.MP = player.MaxMP
	}
	player.Food = 40 // Java: C_Restart sets food = 40

	// Get respawn location based on current map (Lua: scripts/world/respawn.lua)
	rx, ry, rmap := getBackLocation(player.MapID, deps)

	// Clear old tile (for NPC pathfinding)
	if deps.MapData != nil {
		deps.MapData.SetImpassable(player.MapID, player.X, player.Y, false)
	}

	// Broadcast removal from old position + unblock entity collision
	nearby := deps.World.GetNearbyPlayers(player.X, player.Y, player.MapID, sess.ID)
	for _, other := range nearby {
		sendRemoveObject(other.Session, player.CharID)
		SendEntityUnblock(other.Session, player.X, player.Y)
	}

	// Move to respawn point
	deps.World.UpdatePosition(sess.ID, rx, ry, rmap, 0)

	// Mark new tile as impassable (for NPC pathfinding)
	if deps.MapData != nil {
		deps.MapData.SetImpassable(rmap, rx, ry, true)
	}

	// Send map ID (in case map changed)
	sendMapID(sess, uint16(rmap), false)

	// Send own char pack at new position
	sendPutObject(sess, player)

	// Send status update
	sendPlayerStatus(sess, player)

	// Send to nearby players at new location + entity collision
	newNearby := deps.World.GetNearbyPlayers(rx, ry, rmap, sess.ID)
	for _, other := range newNearby {
		sendPutObject(other.Session, player)
		sendPutObject(sess, other)
		// Mutual entity collision
		SendEntityBlock(other.Session, rx, ry, rmap, deps.World)
		SendEntityBlock(sess, other.X, other.Y, rmap, deps.World)
	}

	// Send nearby NPCs + entity collision
	nearbyNpcs := deps.World.GetNearbyNpcs(rx, ry, rmap)
	for _, npc := range nearbyNpcs {
		sendNpcPack(sess, npc)
		if !npc.Dead {
			SendEntityBlock(sess, npc.X, npc.Y, rmap, deps.World)
		}
	}

	// Send nearby ground items
	nearbyGnd := deps.World.GetNearbyGroundItems(rx, ry, rmap)
	for _, g := range nearbyGnd {
		sendDropItem(sess, g)
	}

	// Send weather
	sendWeather(sess, 0)

	deps.Log.Info(fmt.Sprintf("玩家重新開始  角色=%s  x=%d  y=%d  地圖=%d", player.Name, rx, ry, rmap))
}

// KillPlayer handles player death: set dead, broadcast animation, apply exp penalty.
func KillPlayer(player *world.PlayerInfo, deps *Deps) {
	if player.Dead {
		return
	}

	player.Dead = true
	player.HP = 0

	// Dead player no longer occupies the tile
	deps.World.VacateEntity(player.MapID, player.X, player.Y, player.CharID)

	// Broadcast death animation to self + nearby, unblock entity collision (dead = passable)
	nearby := deps.World.GetNearbyPlayersAt(player.X, player.Y, player.MapID)
	for _, viewer := range nearby {
		sendActionGfx(viewer.Session, player.CharID, 8) // ACTION_Die = 8
		if viewer.CharID != player.CharID {
			SendEntityUnblock(viewer.Session, player.X, player.Y)
		}
	}
	sendActionGfx(player.Session, player.CharID, 8)

	// Clear ALL buffs on death (good and bad, no exceptions)
	clearAllBuffsOnDeath(player, deps)

	// Send HP update (0)
	sendHpUpdate(player.Session, player)

	// Exp penalty via Lua (scripts/core/levelup.lua): 5% of level exp range
	applyDeathExpPenalty(player, deps)
	sendExpUpdate(player.Session, player.Level, player.Exp)

	deps.Log.Info(fmt.Sprintf("玩家死亡  角色=%s  x=%d  y=%d", player.Name, player.X, player.Y))
}

// applyDeathExpPenalty subtracts exp on death via Lua (scripts/core/levelup.lua).
func applyDeathExpPenalty(player *world.PlayerInfo, deps *Deps) {
	penalty := deps.Scripting.CalcDeathExpPenalty(int(player.Level), int(player.Exp))
	if penalty > 0 {
		player.Exp -= int32(penalty)
	}
}

// clearAllBuffsOnDeath removes ALL buffs (good and bad) unconditionally.
// Unlike cancelAllBuffs, this ignores the non-cancellable list.
func clearAllBuffsOnDeath(player *world.PlayerInfo, deps *Deps) {
	if player.ActiveBuffs == nil {
		return
	}
	for skillID, buff := range player.ActiveBuffs {
		revertBuffStats(player, buff)
		delete(player.ActiveBuffs, skillID)
		cancelBuffIcon(player, skillID)

		if skillID == SkillShapeChange {
			UndoPoly(player, deps)
		}
		if buff.SetMoveSpeed > 0 {
			player.MoveSpeed = 0
			player.HasteTicks = 0
			sendSpeedToAll(player, deps, 0, 0)
		}
		if buff.SetBraveSpeed > 0 {
			player.BraveSpeed = 0
			sendSpeedToAll(player, deps, 0, 0)
		}
	}
	sendPlayerStatus(player.Session, player)
}

// getBackLocation returns respawn coordinates via Lua (scripts/world/respawn.lua).
func getBackLocation(mapID int16, deps *Deps) (int32, int32, int16) {
	loc := deps.Scripting.GetRespawnLocation(int(mapID))
	if loc != nil {
		return int32(loc.X), int32(loc.Y), int16(loc.Map)
	}
	return 33084, 33391, 4
}
