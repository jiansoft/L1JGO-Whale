package handler

import (
	"time"

	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/net/packet"
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

	// --- Move speed validation (anti speed-hack) ---
	// Normal walk ~200ms, haste ~133ms. Apply 80% tolerance.
	now := time.Now().UnixNano()
	minInterval := int64(160_000_000) // 200ms * 80% = 160ms
	if player.MoveSpeed == 1 {
		minInterval = 106_000_000 // 133ms * 80% = 106ms
	}
	if player.LastMoveTime > 0 && (now-player.LastMoveTime) < minInterval {
		sendOwnCharPackPlayer(sess, player) // rubber-band speed hacker
		return
	}
	player.LastMoveTime = now

	// Always use server-tracked position (matching Java Taiwan behavior).
	curX := player.X
	curY := player.Y

	// Calculate destination from base position + heading
	destX := curX + headingDX[heading]
	destY := curY + headingDY[heading]

	// Entity collision: S_CHANGE_ATTR blocks 4 cardinal directions client-side,
	// but diagonal movement (NE/NW/SE/SW) bypasses edge checks.
	// Server-side IsOccupied catches diagonal bypass attempts.
	// No rubber-band packet (sendOwnCharPackPlayer causes NPC render suppression).
	// Instead, silently reject the move — server keeps old position, client auto-corrects
	// on next move since we always use server-tracked position.
	if ws.IsOccupied(destX, destY, player.MapID, player.CharID) {
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

	// 1. Players in BOTH old and new: send movement
	for _, other := range newNearby {
		if _, wasOld := oldSet[other.SessionID]; wasOld {
			sendMoveObject(other.Session, player.CharID, curX, curY, heading)
			// Proximity-based: unblock old, conditionally block new
			SendEntityUnblock(other.Session, curX, curY)
			if ChebyshevDist(destX, destY, other.X, other.Y) <= entityBlockRange {
				SendEntityBlock(other.Session, destX, destY, player.MapID, ws)
			}
			// Update their blocking for us based on proximity
			if ChebyshevDist(destX, destY, other.X, other.Y) <= entityBlockRange {
				SendEntityBlock(sess, other.X, other.Y, player.MapID, ws)
			} else {
				SendEntityUnblock(sess, other.X, other.Y)
			}
		}
	}

	// 2. Players in NEW but not OLD: they just entered our view
	for _, other := range newNearby {
		if _, wasOld := oldSet[other.SessionID]; !wasOld {
			sendPutObject(sess, other)          // We see them appear
			sendPutObject(other.Session, player) // They see us appear
			// Only block if within proximity
			if ChebyshevDist(destX, destY, other.X, other.Y) <= entityBlockRange {
				SendEntityBlock(sess, other.X, other.Y, player.MapID, ws)
				SendEntityBlock(other.Session, destX, destY, player.MapID, ws)
			}
		}
	}

	// 3. Players in OLD but not NEW: they left our view
	for _, other := range oldNearby {
		if _, isNew := newSet[other.SessionID]; !isNew {
			sendRemoveObject(sess, other.CharID)
			sendRemoveObject(other.Session, player.CharID)
			SendEntityUnblock(sess, other.X, other.Y)
			SendEntityUnblock(other.Session, curX, curY)
		}
	}

	// --- NPC AOI: show/hide NPCs as player moves ---
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

	// NPCs newly visible — send appearance (no blocking yet, proximity triggers it)
	for _, n := range newNpcs {
		if _, wasOld := oldNpcSet[n.ID]; !wasOld {
			sendNpcPack(sess, n)
		}
	}
	// NPCs no longer visible — remove + unblock
	for _, n := range oldNpcs {
		if _, isNew := newNpcSet[n.ID]; !isNew {
			sendRemoveObject(sess, n.ID)
			SendEntityUnblock(sess, n.X, n.Y)
		}
	}

	// Proximity-based entity collision: block/unblock NPCs based on distance
	for _, n := range newNpcs {
		if n.Dead {
			continue
		}
		dist := ChebyshevDist(destX, destY, n.X, n.Y)
		if dist <= entityBlockRange {
			SendEntityBlock(sess, n.X, n.Y, player.MapID, ws)
		} else {
			SendEntityUnblock(sess, n.X, n.Y)
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
