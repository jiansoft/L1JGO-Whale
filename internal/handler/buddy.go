package handler

import (
	"context"
	"strings"
	"time"

	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/net/packet"
	"github.com/l1jgo/server/internal/world"
	"go.uber.org/zap"
)

// HandleQueryBuddy processes C_QUERY_BUDDY (opcode 4) — request buddy list.
// Java: C_Buddy.java → responds with S_Buddy (S_OPCODE_HYPERTEXT window "buddy").
func HandleQueryBuddy(sess *net.Session, _ *packet.Reader, deps *Deps) {
	player := deps.World.GetBySession(sess.ID)
	if player == nil {
		return
	}

	sendBuddyList(sess, player, deps)
}

// HandleAddBuddy processes C_ADD_BUDDY (opcode 207) — add a buddy.
// Java: C_AddBuddy.java → validates name, inserts into DB + memory.
func HandleAddBuddy(sess *net.Session, r *packet.Reader, deps *Deps) {
	name := r.ReadS()
	if name == "" {
		return
	}

	player := deps.World.GetBySession(sess.ID)
	if player == nil {
		return
	}

	// Cannot add yourself
	if strings.EqualFold(name, player.Name) {
		return
	}

	// Check if already in buddy list
	for _, b := range player.Buddies {
		if strings.EqualFold(b.Name, name) {
			return // duplicate
		}
	}

	// Verify target character exists in DB
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	target, err := deps.CharRepo.LoadByName(ctx, name)
	if err != nil {
		deps.Log.Error("好友查詢失敗", zap.String("name", name), zap.Error(err))
		return
	}
	if target == nil {
		sendServerMessage(sess, 109) // "Character not found"
		return
	}

	// Add to DB
	if err := deps.BuddyRepo.Add(ctx, player.CharID, target.ID, target.Name); err != nil {
		deps.Log.Error("新增好友失敗", zap.String("name", name), zap.Error(err))
		return
	}

	// Add to in-memory list
	player.Buddies = append(player.Buddies, world.BuddyEntry{
		CharID: target.ID,
		Name:   target.Name,
	})
}

// HandleRemoveBuddy processes C_REMOVE_BUDDY (opcode 202) — remove a buddy.
// Java: C_DelBuddy.java → removes from DB + memory by name (case-insensitive).
func HandleRemoveBuddy(sess *net.Session, r *packet.Reader, deps *Deps) {
	name := r.ReadS()
	if name == "" {
		return
	}

	player := deps.World.GetBySession(sess.ID)
	if player == nil {
		return
	}

	// Find and remove from in-memory list
	found := false
	for i, b := range player.Buddies {
		if strings.EqualFold(b.Name, name) {
			player.Buddies = append(player.Buddies[:i], player.Buddies[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return
	}

	// Remove from DB
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := deps.BuddyRepo.Remove(ctx, player.CharID, name); err != nil {
		deps.Log.Error("刪除好友失敗", zap.String("name", name), zap.Error(err))
	}
}

// sendBuddyList sends S_Buddy (S_OPCODE_HYPERTEXT, window "buddy") — buddy list with online status.
// Java: S_Buddy.java → [D objID][S "buddy"][H 2][H 2][S allNames][S onlineNames]
func sendBuddyList(sess *net.Session, player *world.PlayerInfo, deps *Deps) {
	var allNames, onlineNames strings.Builder

	for _, b := range player.Buddies {
		allNames.WriteString(b.Name)
		allNames.WriteByte(' ')

		if deps.World.GetByName(b.Name) != nil {
			onlineNames.WriteString(b.Name)
			onlineNames.WriteByte(' ')
		}
	}

	w := packet.NewWriterWithOpcode(packet.S_OPCODE_HYPERTEXT)
	w.WriteD(player.CharID)
	w.WriteS("buddy")
	w.WriteH(2)
	w.WriteH(2)
	w.WriteS(allNames.String())
	w.WriteS(onlineNames.String())
	sess.Send(w.Bytes())
}
