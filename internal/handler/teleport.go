package handler

import (
	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/net/packet"
)

// HandleTeleport processes C_TELEPORT (opcode 52) — teleport confirmation.
// Used when SEND_PACKET_BEFORE_TELEPORT=true (server sends S_Teleport, client responds).
// On failure: send S_Paralysis(TYPE_TELEPORT_UNLOCK=7) to unfreeze client.
// On success: client unfreezes automatically when it receives S_MapID + S_OwnCharPack.
func HandleTeleport(sess *net.Session, _ *packet.Reader, deps *Deps) {
	player := deps.World.GetBySession(sess.ID)
	if player == nil || player.Dead {
		sendTeleportUnlock(sess)
		return
	}

	// Check for pre-stored teleport destination
	if !player.HasTeleport {
		sendTeleportUnlock(sess)
		return
	}

	// Auto-cancel trade when teleporting
	cancelTradeIfActive(player, deps)

	player.HasTeleport = false

	teleportPlayer(sess, player,
		player.TeleportX, player.TeleportY,
		player.TeleportMapID, player.TeleportHeading, deps)
}
