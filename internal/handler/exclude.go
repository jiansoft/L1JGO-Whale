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

// HandleExclude processes C_EXCLUDE (opcode 171) — toggle exclude/block.
// If already excluded, remove; otherwise add (max 16). Persisted to DB.
func HandleExclude(sess *net.Session, r *packet.Reader, deps *Deps) {
	name := r.ReadS()
	if name == "" {
		return
	}

	player := deps.World.GetBySession(sess.ID)
	if player == nil {
		return
	}

	// Check if already excluded → toggle off
	for i, ex := range player.ExcludeList {
		if strings.EqualFold(ex, name) {
			player.ExcludeList = append(player.ExcludeList[:i], player.ExcludeList[i+1:]...)
			sendExcludeRemove(sess, name)

			// Persist removal
			if deps.ExcludeRepo != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				if err := deps.ExcludeRepo.Remove(ctx, player.CharID, name); err != nil {
					deps.Log.Error("刪除黑名單失敗", zap.String("name", name), zap.Error(err))
				}
			}
			return
		}
	}

	// Not excluded → add
	if len(player.ExcludeList) >= deps.Config.Gameplay.MaxExcludeList {
		sendServerMessage(sess, 472) // "被拒絕的玩家太多。" (Reject list is full)
		return
	}

	player.ExcludeList = append(player.ExcludeList, name)
	sendExcludeAdd(sess, name)

	// Persist addition
	if deps.ExcludeRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := deps.ExcludeRepo.Add(ctx, player.CharID, name); err != nil {
			deps.Log.Error("新增黑名單失敗", zap.String("name", name), zap.Error(err))
		}
	}
}

// IsExcluded checks if the listener has blocked the sender (case-insensitive).
func IsExcluded(listener *world.PlayerInfo, senderName string) bool {
	for _, ex := range listener.ExcludeList {
		if strings.EqualFold(ex, senderName) {
			return true
		}
	}
	return false
}

// sendExcludeAdd sends S_PacketBox subcode 18 — notify client of new exclude entry.
func sendExcludeAdd(sess *net.Session, name string) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_EVENT)
	w.WriteC(18) // ADD_EXCLUDE
	w.WriteS(name)
	sess.Send(w.Bytes())
}

// sendExcludeRemove sends S_PacketBox subcode 19 — notify client of removed exclude entry.
func sendExcludeRemove(sess *net.Session, name string) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_EVENT)
	w.WriteC(19) // REM_EXCLUDE
	w.WriteS(name)
	sess.Send(w.Bytes())
}
