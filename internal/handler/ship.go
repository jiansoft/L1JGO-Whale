package handler

import (
	"fmt"

	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/net/packet"
)

// shipTicketByDestMap maps destination ship-map ID to the ticket item ID that must
// be consumed. These are protocol-level constants matching the 3.80C client.
// Java: C_Ship.java — switch on shipMapId to determine ticket item.
var shipTicketByDestMap = map[int16]int32{
	5:   40299, // Talking Island → Gludin ship
	6:   40298, // Aden Mainland → Talking Island ship
	83:  40300, // Forgotten Island → Aden ship
	84:  40301, // Aden Mainland → Forgotten Island ship
	446: 40303, // Hidden Dock → Pirate Island ship
	447: 40302, // Pirate Island → Hidden Dock ship
}

// HandleEnterShip processes C_ENTER_SHIP (opcode 231) — board an inter-island ship.
// Java: C_Ship.java — reads destination (mapId, x, y), consumes ticket, teleports.
// Packet format: [H destMapId][H destX][H destY]
func HandleEnterShip(sess *net.Session, r *packet.Reader, deps *Deps) {
	destMapID := int16(r.ReadH())
	destX := int32(r.ReadH())
	destY := int32(r.ReadH())

	player := deps.World.GetBySession(sess.ID)
	if player == nil || player.Dead {
		return
	}

	// Look up the required ticket for this destination
	ticketItemID, ok := shipTicketByDestMap[destMapID]
	if !ok {
		deps.Log.Warn(fmt.Sprintf("船票  角色=%s  未知目的地=%d", player.Name, destMapID))
		return
	}

	// Find and consume the ticket
	ticket := player.Inv.FindByItemID(ticketItemID)
	if ticket == nil {
		// No ticket — silent fail matching Java behavior
		return
	}

	fullyRemoved := player.Inv.RemoveItem(ticket.ObjectID, 1)
	if fullyRemoved {
		sendRemoveInventoryItem(sess, ticket.ObjectID)
	} else {
		sendItemCountUpdate(sess, ticket)
	}
	sendWeightUpdate(sess, player)

	// Cancel any active trade (Java: L1Trade.tradeCancel in teleportation)
	cancelTradeIfActive(player, deps)

	// Teleport to the ship destination (teleportPlayer sends S_Paralysis unlock)
	teleportPlayer(sess, player, destX, destY, destMapID, 5, deps)

	deps.Log.Info(fmt.Sprintf("搭船  角色=%s  目的地=%d  x=%d  y=%d",
		player.Name, destMapID, destX, destY))
}
