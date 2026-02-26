package handler

import (
	"fmt"

	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/net/packet"
)

// HandleWho processes C_WHO (opcode 206) — query character info or online count.
// Java: C_Who.java → if name provided, returns character info; otherwise returns online count.
func HandleWho(sess *net.Session, r *packet.Reader, deps *Deps) {
	name := r.ReadS()

	player := deps.World.GetBySession(sess.ID)
	if player == nil {
		return
	}

	// If name is provided, try to look up the character
	if name != "" {
		target := deps.World.GetByName(name)
		if target != nil {
			info := buildWhoInfo(target.Title, target.Name, target.Lawful, target.ClanName)
			sendWhoCharinfo(sess, info)
			return
		}
		// Target not found online — fall through to show online count
	}

	// Show online player count
	count := deps.World.PlayerCount()
	sendWhoAmount(sess, count)
}

// buildWhoInfo formats character info for S_WhoCharinfo.
// Java: S_WhoCharinfo.java → "[title] name (lawfulness) [clanName]"
func buildWhoInfo(title, name string, lawful int32, clanName string) string {
	result := ""
	if title != "" {
		result += title + " "
	}
	result += name + " "

	// Lawful alignment as message code reference
	if lawful >= 500 {
		result += "($1501)" // Good
	} else if lawful >= 0 {
		result += "($1502)" // Neutral
	} else {
		result += "($1503)" // Evil
	}

	if clanName != "" {
		result += " [" + clanName + "]"
	}

	return result
}

// sendWhoCharinfo sends S_MESSAGE_CODE (opcode 71) msgID=166 — character info response.
func sendWhoCharinfo(sess *net.Session, info string) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_MESSAGE_CODE)
	w.WriteH(166)
	w.WriteC(1)
	w.WriteS(info)
	sess.Send(w.Bytes())
}

// sendWhoAmount sends S_MESSAGE_CODE (opcode 71) msgID=81 — online player count.
func sendWhoAmount(sess *net.Session, count int) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_MESSAGE_CODE)
	w.WriteH(81)
	w.WriteC(1)
	w.WriteS(fmt.Sprintf("%d", count))
	sess.Send(w.Bytes())
}
