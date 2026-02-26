package handler

import (
	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/net/packet"
	"github.com/l1jgo/server/internal/world"
)

// HandleOpen processes C_OPEN (opcode 41) — player clicks a door to open/close.
// Packet: [H padding][H padding][D objectID]
func HandleOpen(sess *net.Session, r *packet.Reader, deps *Deps) {
	_ = r.ReadH() // skip
	_ = r.ReadH() // skip
	objectID := r.ReadD()

	player := deps.World.GetBySession(sess.ID)
	if player == nil {
		return
	}

	door := deps.World.GetDoor(objectID)
	if door == nil {
		return
	}

	// Dead doors can't be toggled
	if door.Dead {
		return
	}

	// Keeper check: if door has a keeper, only the clan owning that house can toggle.
	// For now, keeper doors with non-zero KeeperID are restricted.
	// TODO: Implement full clan house keeper lookup when house system is added.
	if door.KeeperID != 0 {
		// Block interaction — requires clan house ownership check
		return
	}

	// Toggle door state
	if door.OpenStatus == world.DoorActionOpen {
		door.Close()
		broadcastDoorClose(door, deps)
	} else if door.OpenStatus == world.DoorActionClose {
		door.Open()
		broadcastDoorOpen(door, deps)
	}
}

// broadcastDoorOpen sends open state to all nearby players and updates tile passability.
func broadcastDoorOpen(door *world.DoorInfo, deps *Deps) {
	nearby := deps.World.GetNearbyPlayersAt(door.X, door.Y, door.MapID)
	for _, viewer := range nearby {
		sendDoorPack(viewer.Session, door)
		sendDoorAction(viewer.Session, door.ID, world.DoorActionOpen)
	}
	// Update passability: open = passable
	sendDoorTilesAll(door, deps)
}

// broadcastDoorClose sends close state to all nearby players and updates tile passability.
func broadcastDoorClose(door *world.DoorInfo, deps *Deps) {
	nearby := deps.World.GetNearbyPlayersAt(door.X, door.Y, door.MapID)
	for _, viewer := range nearby {
		sendDoorPack(viewer.Session, door)
		sendDoorAction(viewer.Session, door.ID, world.DoorActionClose)
	}
	// Update passability: close = blocked
	sendDoorTilesAll(door, deps)
}

// sendDoorPack sends S_DoorPack (opcode 87 = S_PUT_OBJECT) — door appearance.
// Same opcode as S_CharPack but with door-specific status byte.
func sendDoorPack(viewer *net.Session, door *world.DoorInfo) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_PUT_OBJECT)
	w.WriteH(uint16(door.X))
	w.WriteH(uint16(door.Y))
	w.WriteD(door.ID)
	w.WriteH(uint16(door.GfxID))
	w.WriteC(door.PackStatus()) // door state: 28=open, 29=close, 32-37=damage
	w.WriteC(0)                 // heading
	w.WriteC(0)                 // light
	w.WriteC(0)                 // speed
	w.WriteD(1)                 // always 1 (Java S_DoorPack)
	w.WriteH(0)                 // lawful
	w.WriteS("")                // name (null)
	w.WriteS("")                // title (null)
	w.WriteC(0x00)              // status flags (not a PC)
	w.WriteD(0)                 // reserved
	w.WriteS("")                // clan (null)
	w.WriteS("")                // master (null)
	w.WriteC(0x00)              // hidden
	w.WriteC(0xFF)              // HP% (full)
	w.WriteC(0x00)              // reserved
	w.WriteC(0x00)              // level (0 for door)
	w.WriteC(0xFF)              // reserved
	w.WriteC(0xFF)              // reserved
	w.WriteC(0x00)              // reserved
	viewer.Send(w.Bytes())
}

// sendDoorAttr sends S_Door (opcode 209 = S_CHANGE_ATTR) — tile passability.
// Format: [H x][H y][C direction][C passable]
// Java S_Door.java: PASS = 0, NOT_PASS = 1
// Client interprets: 0 = can walk through, 1 = blocked
func sendDoorAttr(viewer *net.Session, x, y int32, direction int, passable bool) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_CHANGE_ATTR)
	w.WriteH(uint16(x))
	w.WriteH(uint16(y))
	w.WriteC(byte(direction))
	if passable {
		w.WriteC(0) // PASS (Java: PASS = 0)
	} else {
		w.WriteC(1) // NOT_PASS (Java: NOT_PASS = 1)
	}
	viewer.Send(w.Bytes())
}

// sendDoorAction sends S_DoActionGFX (opcode 158) — door open/close/damage animation.
func sendDoorAction(viewer *net.Session, doorID int32, actionCode byte) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_ACTION)
	w.WriteD(doorID)
	w.WriteC(actionCode)
	viewer.Send(w.Bytes())
}

// SendDoorPerceive sends door appearance + passability tiles to a single player.
// Called when a player first sees a door (enter world, teleport, movement AOI).
func SendDoorPerceive(sess *net.Session, door *world.DoorInfo) {
	sendDoorPack(sess, door)
	// Only send tile blockers if door is closed (Java optimization)
	if door.OpenStatus == world.DoorActionClose && !door.Dead {
		sendDoorTilesToPlayer(sess, door)
	}
}

// sendDoorTilesToPlayer sends S_Door passability packets for all tiles of a door to one player.
func sendDoorTilesToPlayer(sess *net.Session, door *world.DoorInfo) {
	passable := door.IsPassable()
	entranceX := door.EntranceX()
	entranceY := door.EntranceY()
	left := door.LeftEdge
	right := door.RightEdge

	if left == right {
		// Single-tile door
		sendDoorAttr(sess, entranceX, entranceY, door.Direction, passable)
		return
	}

	// Multi-tile door
	if door.Direction == 0 {
		// "/" direction: iterate X, fixed Y
		for x := left; x <= right; x++ {
			sendDoorAttr(sess, x, entranceY, door.Direction, passable)
		}
	} else {
		// "\" direction: iterate Y, fixed X
		for y := left; y <= right; y++ {
			sendDoorAttr(sess, entranceX, y, door.Direction, passable)
		}
	}
}

// sendDoorTilesAll broadcasts S_Door passability to all nearby players.
func sendDoorTilesAll(door *world.DoorInfo, deps *Deps) {
	nearby := deps.World.GetNearbyPlayersAt(door.X, door.Y, door.MapID)
	for _, viewer := range nearby {
		sendDoorTilesToPlayer(viewer.Session, door)
	}
}

// ==================== Entity Collision (proximity-based S_CHANGE_ATTR) ====================
//
// Approach: no blocking by default. Only when a player is within proximity (≤3 tiles)
// of an entity, dynamically send S_CHANGE_ATTR to block all 4 edges.
//
// Own-tile S_CHANGE_ATTR (dir=0 + dir=1) does NOT cause render suppression — safe always.
// Neighbor-tile S_CHANGE_ATTR (south/west edges) skips tiles occupied by other entities.

const entityBlockRange = 3 // Chebyshev distance to trigger blocking

type entityDoorKey struct {
	viewerID uint64
	x, y     int32
}

type entityDoorEntry struct {
	selfDir0     bool // NOT_PASS at (x, y) dir=0 — blocks north edge
	selfDir1     bool // NOT_PASS at (x, y) dir=1 — blocks east edge
	southBlocked bool // NOT_PASS at (x, y+1) dir=0 — blocks south edge
	westBlocked  bool // NOT_PASS at (x-1, y) dir=1 — blocks west edge
}

var (
	entityDoorMap       = make(map[entityDoorKey]entityDoorEntry)
	entityDoorsByViewer = make(map[uint64]map[entityDoorKey]struct{})
)

// SendEntityBlock sets S_CHANGE_ATTR on all 4 edges around (x, y).
// Own-tile edges (north+east) always sent. Neighbor edges (south+west) skip occupied tiles.
func SendEntityBlock(viewer *net.Session, x, y int32, mapID int16, ws *world.State) {
	key := entityDoorKey{viewerID: viewer.ID, x: x, y: y}
	if _, ok := entityDoorMap[key]; ok {
		return
	}

	var entry entityDoorEntry

	// Own tile dir=0: blocks "/" edge between (x,y) and (x,y-1) — north
	sendDoorAttr(viewer, x, y, 0, false)
	entry.selfDir0 = true

	// Own tile dir=1: blocks "\" edge between (x,y) and (x+1,y) — east
	sendDoorAttr(viewer, x, y, 1, false)
	entry.selfDir1 = true

	// South neighbor (x, y+1) dir=0: blocks edge between (x,y+1) and (x,y) — south
	sendDoorAttr(viewer, x, y+1, 0, false)
	entry.southBlocked = true

	// West neighbor (x-1, y) dir=1: blocks edge between (x-1,y) and (x,y) — west
	sendDoorAttr(viewer, x-1, y, 1, false)
	entry.westBlocked = true

	entityDoorMap[key] = entry
	if entityDoorsByViewer[viewer.ID] == nil {
		entityDoorsByViewer[viewer.ID] = make(map[entityDoorKey]struct{})
	}
	entityDoorsByViewer[viewer.ID][key] = struct{}{}
}

// SendEntityUnblock restores passability for all directions that were blocked.
func SendEntityUnblock(viewer *net.Session, x, y int32) {
	key := entityDoorKey{viewerID: viewer.ID, x: x, y: y}
	entry, ok := entityDoorMap[key]
	if !ok {
		return
	}
	if entry.selfDir0 {
		sendDoorAttr(viewer, x, y, 0, true)
	}
	if entry.selfDir1 {
		sendDoorAttr(viewer, x, y, 1, true)
	}
	if entry.southBlocked {
		sendDoorAttr(viewer, x, y+1, 0, true)
	}
	if entry.westBlocked {
		sendDoorAttr(viewer, x-1, y, 1, true)
	}
	delete(entityDoorMap, key)
	if viewerKeys := entityDoorsByViewer[viewer.ID]; viewerKeys != nil {
		delete(viewerKeys, key)
	}
}

// CleanupViewerEntityDoors removes all tracking for a disconnecting viewer.
func CleanupViewerEntityDoors(viewerID uint64) {
	keys := entityDoorsByViewer[viewerID]
	for key := range keys {
		delete(entityDoorMap, key)
	}
	delete(entityDoorsByViewer, viewerID)
}

// ChebyshevDist returns Chebyshev (chessboard) distance between two points.
func ChebyshevDist(x1, y1, x2, y2 int32) int32 {
	dx := x1 - x2
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y2
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

