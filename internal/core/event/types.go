package event

import "github.com/l1jgo/server/internal/core/ecs"

// --- Session lifecycle events ---

type PlayerLoggedIn struct {
	EntityID    ecs.EntityID
	AccountName string
}

type PlayerDisconnected struct {
	EntityID  ecs.EntityID
	SessionID uint64
}

// --- Combat events (emitted by CombatSystem, readable next tick) ---

// EntityKilled is emitted when an NPC dies.
// Subscribers: QuestSystem (kill count), future AchievementSystem, etc.
type EntityKilled struct {
	KillerSessionID uint64
	KillerCharID    int32
	NpcID           int32 // world NPC object ID
	NpcTemplateID   int32 // NPC template ID from spawn data
	ExpGained       int32
	MapID           int16
	X, Y            int32
}

// PlayerDied is emitted when any player dies (PvE or PvP).
// Subscribers: respawn system, death penalty, quest failure checks.
type PlayerDied struct {
	CharID int32
	MapID  int16
	X, Y   int32
}

// PlayerKilled is emitted when a player is killed by another player (PK).
// Emitted in addition to PlayerDied — subscribe to this for PK-specific logic.
type PlayerKilled struct {
	KillerCharID int32
	VictimCharID int32
	MapID        int16
	X, Y         int32
}
