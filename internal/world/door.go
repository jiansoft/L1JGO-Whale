package world

import "sync/atomic"

// Door action codes (Java ActionCodes.java)
const (
	DoorActionOpen  byte = 28 // door is open, passable
	DoorActionClose byte = 29 // door is closed, blocked

	DoorActionDmg1 byte = 32 // 5/6 HP remaining
	DoorActionDmg2 byte = 33 // 4/6 HP
	DoorActionDmg3 byte = 34 // 3/6 HP
	DoorActionDmg4 byte = 35 // 2/6 HP
	DoorActionDmg5 byte = 36 // 1/6 HP
	DoorActionDie  byte = 37 // 0 HP, destroyed
)

// doorIDCounter generates unique door object IDs.
// Starts at 300_000_000 to avoid collision with char IDs, NPC IDs (200M), item IDs (500M).
var doorIDCounter atomic.Int32

func init() {
	doorIDCounter.Store(300_000_000)
}

// NextDoorID returns a unique object ID for a door instance.
func NextDoorID() int32 {
	return doorIDCounter.Add(1)
}

// DoorInfo holds runtime data for a door currently in-world.
// Accessed only from the game loop goroutine — no locks.
type DoorInfo struct {
	ID        int32 // unique object ID (from NextDoorID)
	DoorID    int32 // spawn template ID
	GfxID     int32 // sprite GFX ID
	X         int32
	Y         int32
	MapID     int16
	MaxHP     int32 // 0 = indestructible
	HP        int32
	KeeperID  int32 // clan keeper NPC ID (0 = public door)
	Direction int   // 0 = "/" (NE-SW), 1 = "\" (NW-SE)

	// Multi-tile edge locations (absolute coordinates, not offsets)
	LeftEdge  int32 // left edge coordinate (X if dir=0, Y if dir=1)
	RightEdge int32 // right edge coordinate

	// Runtime state
	OpenStatus byte // DoorActionOpen or DoorActionClose
	DmgStatus  byte // 0 = healthy, DoorActionDmg1-5, DoorActionDie
	Dead       bool
}

// IsPassable returns true if the door allows passage (open or destroyed).
func (d *DoorInfo) IsPassable() bool {
	return d.Dead || d.OpenStatus == DoorActionOpen
}

// EntranceX returns the X coordinate where the door blocks passage.
func (d *DoorInfo) EntranceX() int32 {
	if d.Direction == 0 {
		return d.X // "/" direction
	}
	return d.X - 1 // "\" direction
}

// EntranceY returns the Y coordinate where the door blocks passage.
func (d *DoorInfo) EntranceY() int32 {
	if d.Direction == 0 {
		return d.Y + 1 // "/" direction
	}
	return d.Y // "\" direction
}

// PackStatus returns the status byte for S_DoorPack.
// Priority: dead > open > damaged > closed.
func (d *DoorInfo) PackStatus() byte {
	if d.Dead {
		return DoorActionDie
	}
	if d.OpenStatus == DoorActionOpen {
		return DoorActionOpen
	}
	if d.MaxHP > 1 && d.DmgStatus != 0 {
		return d.DmgStatus
	}
	return d.OpenStatus
}

// Open transitions the door to open state.
func (d *DoorInfo) Open() bool {
	if d.Dead || d.OpenStatus == DoorActionOpen {
		return false
	}
	d.OpenStatus = DoorActionOpen
	return true
}

// Close transitions the door to closed state.
func (d *DoorInfo) Close() bool {
	if d.Dead || d.OpenStatus == DoorActionClose {
		return false
	}
	d.OpenStatus = DoorActionClose
	return true
}

// ReceiveDamage applies damage and returns true if door just died.
func (d *DoorInfo) ReceiveDamage(damage int32) bool {
	if d.MaxHP == 0 || d.HP <= 0 || d.Dead {
		return false
	}
	d.HP -= damage
	if d.HP <= 0 {
		d.HP = 0
		d.Dead = true
		d.DmgStatus = DoorActionDie
		d.OpenStatus = DoorActionOpen // dead doors are passable
		return true
	}
	d.updateDmgStatus()
	return false
}

// updateDmgStatus maps current HP to damage visual stage.
func (d *DoorInfo) updateDmgStatus() {
	if d.MaxHP <= 0 {
		return
	}
	var newStatus byte
	switch {
	case d.HP*6 <= d.MaxHP*1:
		newStatus = DoorActionDmg5
	case d.HP*6 <= d.MaxHP*2:
		newStatus = DoorActionDmg4
	case d.HP*6 <= d.MaxHP*3:
		newStatus = DoorActionDmg3
	case d.HP*6 <= d.MaxHP*4:
		newStatus = DoorActionDmg2
	case d.HP*6 <= d.MaxHP*5:
		newStatus = DoorActionDmg1
	default:
		newStatus = 0
	}
	d.DmgStatus = newStatus
}

// RepairGate resets the door to full HP and closed state (clan house repair).
func (d *DoorInfo) RepairGate() {
	if d.MaxHP <= 1 {
		return
	}
	d.Dead = false
	d.HP = d.MaxHP
	d.DmgStatus = 0
	d.OpenStatus = DoorActionClose
}
