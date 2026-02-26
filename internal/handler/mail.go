package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/net/packet"
	"github.com/l1jgo/server/internal/persist"
	"github.com/l1jgo/server/internal/world"
	"go.uber.org/zap"
)

const (
	mailTypeNormal  int16 = 0
	mailTypeClan    int16 = 1
	mailTypeStorage int16 = 2
)

// HandleMail processes C_MAIL (opcode 87).
// First byte is the subtype which determines the operation.
func HandleMail(sess *net.Session, r *packet.Reader, deps *Deps) {
	if deps.MailRepo == nil {
		return
	}

	player := deps.World.GetBySession(sess.ID)
	if player == nil {
		return
	}

	mailType := r.ReadC()

	switch mailType {
	case 0x00, 0x02: // Open mailbox: 0=normal, 2=storage (clan=0x01 not implemented)
		handleMailOpen(sess, player, int16(mailType), deps)

	case 0x01: // Clan mail - not implemented
		sendSystemMessage(sess, "血盟信件目前尚未開放使用。")

	case 0x10, 0x12: // Read mail: 0x10=normal, 0x12=storage
		handleMailRead(sess, player, r, int16(mailType-0x10), deps)

	case 0x11: // Clan mail read - not implemented
		sendSystemMessage(sess, "血盟信件目前尚未開放使用。")

	case 0x20: // Send normal mail
		handleMailSend(sess, player, r, deps)

	case 0x21: // Send clan mail - not implemented
		sendSystemMessage(sess, "血盟信件目前尚未開放使用。")

	case 0x30, 0x31, 0x32: // Delete mail
		handleMailDelete(sess, player, r, mailType, deps)

	case 0x40, 0x41: // Move to storage box
		handleMailMoveToStorage(sess, player, r, mailType, deps)

	case 0x60, 0x61, 0x62: // Batch delete
		handleMailBulkDelete(sess, player, r, mailType, deps)
	}
}

// handleMailOpen sends the mail list for the given type.
// Subtype 0x00 (normal) or 0x02 (storage).
func handleMailOpen(sess *net.Session, player *world.PlayerInfo, mailType int16, deps *Deps) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	mails, err := deps.MailRepo.LoadByInbox(ctx, player.CharID, mailType)
	if err != nil {
		deps.Log.Error("讀取信箱失敗", zap.Error(err))
		return
	}

	sendMailList(sess, player, mails, mailType)
}

// handleMailRead reads a specific mail and marks it as read.
// Subtype 0x10 → mailType 0 (normal), 0x12 → mailType 2 (storage).
// Format after type byte: [D mailID]
func handleMailRead(sess *net.Session, player *world.PlayerInfo, r *packet.Reader, mailType int16, deps *Deps) {
	mailID := r.ReadD()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	mail, err := deps.MailRepo.GetByID(ctx, mailID)
	if err != nil {
		deps.Log.Error("讀取信件失敗", zap.Error(err))
		return
	}
	if mail == nil {
		return
	}

	// Only the inbox owner can read
	if mail.InboxID != player.CharID {
		return
	}

	// Mark as read if unread
	if mail.ReadStatus == 0 {
		if err := deps.MailRepo.SetReadStatus(ctx, mailID); err != nil {
			deps.Log.Error("標記信件已讀失敗", zap.Error(err))
		}
	}

	// Send content
	readType := byte(0x10) + byte(mailType)
	sendMailContent(sess, mailID, readType, mail.Content)
}

// handleMailSend processes sending a normal mail (subtype 0x20).
// Format after type byte: [H worldMailCount][S receiverName][...rawBytes]
func handleMailSend(sess *net.Session, player *world.PlayerInfo, r *packet.Reader, deps *Deps) {
	_ = r.ReadH() // worldMailCount (unused)
	receiverName := r.ReadS()
	rawText := r.ReadBytes(r.Remaining())

	// Parse subject and content from raw bytes
	subject, content := parseMailText(rawText)

	// Check adena
	if !consumeAdena(player, int32(deps.Config.Gameplay.MailSendCost)) {
		sendServerMessage(sess, 189) // "金幣不足。"
		return
	}
	sendAdenaUpdate(sess, player)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find receiver - try online first, then DB
	receiver := deps.World.GetByName(receiverName)
	var receiverCharID int32

	if receiver != nil {
		receiverCharID = receiver.CharID
	} else {
		// Offline: look up from DB
		charRow, err := deps.CharRepo.LoadByName(ctx, receiverName)
		if err != nil {
			deps.Log.Error("查詢收件人失敗", zap.Error(err))
			sendMailResult(sess, 0x20, false)
			return
		}
		if charRow == nil {
			sendServerMessage(sess, 109) // "沒有這個人。"
			return
		}
		receiverCharID = charRow.ID
	}

	// Check receiver mailbox limit
	count, err := deps.MailRepo.CountByInbox(ctx, receiverCharID, mailTypeNormal)
	if err != nil {
		deps.Log.Error("查詢收件箱數量失敗", zap.Error(err))
		sendMailResult(sess, 0x20, false)
		return
	}
	if count >= deps.Config.Gameplay.MailMaxPerBox {
		sendMailResult(sess, 0x20, false)
		return
	}

	now := time.Now()

	// Write sender's backup copy
	senderMail := &persist.MailRow{
		Type:       mailTypeNormal,
		Sender:     player.Name,
		Receiver:   receiverName,
		Date:       now,
		ReadStatus: 0,
		InboxID:    player.CharID,
		Subject:    subject,
		Content:    content,
	}
	senderMailID, err := deps.MailRepo.Write(ctx, senderMail)
	if err != nil {
		deps.Log.Error("寫入寄件備份失敗", zap.Error(err))
		sendMailResult(sess, 0x20, false)
		return
	}

	// Write receiver's copy
	receiverMail := &persist.MailRow{
		Type:       mailTypeNormal,
		Sender:     player.Name,
		Receiver:   receiverName,
		Date:       now,
		ReadStatus: 0,
		InboxID:    receiverCharID,
		Subject:    subject,
		Content:    content,
	}
	receiverMailID, err := deps.MailRepo.Write(ctx, receiverMail)
	if err != nil {
		deps.Log.Error("寫入收件信失敗", zap.Error(err))
		sendMailResult(sess, 0x20, false)
		return
	}

	// Notify sender (draft/backup)
	sendMailNotify(sess, player.Name, senderMailID, true, subject)

	// Notify receiver if online
	if receiver != nil {
		sendMailNotify(receiver.Session, player.Name, receiverMailID, false, subject)
		// Sound effect notification (skill sound 1091)
		sendMailSound(receiver.Session, receiver.CharID)
	}

	deps.Log.Info(fmt.Sprintf("信件寄出  寄件=%s  收件=%s  senderID=%d  receiverID=%d",
		player.Name, receiverName, senderMailID, receiverMailID))

	sendMailResult(sess, 0x20, true)
}

// handleMailDelete deletes a single mail.
// Format after type byte: [D mailID]
func handleMailDelete(sess *net.Session, player *world.PlayerInfo, r *packet.Reader, subtype byte, deps *Deps) {
	mailID := r.ReadD()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Verify ownership
	mail, err := deps.MailRepo.GetByID(ctx, mailID)
	if err != nil {
		deps.Log.Error("查詢信件失敗", zap.Error(err))
		return
	}
	if mail == nil || mail.InboxID != player.CharID {
		return
	}

	if err := deps.MailRepo.Delete(ctx, mailID); err != nil {
		deps.Log.Error("刪除信件失敗", zap.Error(err))
		return
	}

	sendMailAck(sess, mailID, subtype)
}

// handleMailMoveToStorage moves a mail to storage box (type → 2).
// Format after type byte: [D mailID]
func handleMailMoveToStorage(sess *net.Session, player *world.PlayerInfo, r *packet.Reader, subtype byte, deps *Deps) {
	mailID := r.ReadD()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Verify ownership
	mail, err := deps.MailRepo.GetByID(ctx, mailID)
	if err != nil {
		deps.Log.Error("查詢信件失敗", zap.Error(err))
		return
	}
	if mail == nil || mail.InboxID != player.CharID {
		return
	}

	// Check storage box limit
	count, err := deps.MailRepo.CountByInbox(ctx, player.CharID, mailTypeStorage)
	if err != nil {
		deps.Log.Error("查詢保管箱數量失敗", zap.Error(err))
		return
	}
	if count >= deps.Config.Gameplay.MailMaxPerBox {
		return
	}

	if err := deps.MailRepo.SetType(ctx, mailID, mailTypeStorage); err != nil {
		deps.Log.Error("移動信件至保管箱失敗", zap.Error(err))
		return
	}

	sendMailAck(sess, mailID, 0x40)
}

// handleMailBulkDelete deletes multiple mails.
// Format after type byte: [D count]{[D mailID] × count}
func handleMailBulkDelete(sess *net.Session, player *world.PlayerInfo, r *packet.Reader, subtype byte, deps *Deps) {
	count := r.ReadD()
	if count <= 0 || int(count) > deps.Config.Gameplay.MailMaxPerBox {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Map subtype to delete ack type: 0x60→0x30, 0x61→0x31, 0x62→0x32
	deleteAckType := subtype - 0x30

	for i := int32(0); i < count; i++ {
		mailID := r.ReadD()

		// Verify ownership
		mail, err := deps.MailRepo.GetByID(ctx, mailID)
		if err != nil {
			deps.Log.Error("批次刪除查詢失敗", zap.Error(err))
			continue
		}
		if mail == nil || mail.InboxID != player.CharID {
			continue
		}

		if err := deps.MailRepo.Delete(ctx, mailID); err != nil {
			deps.Log.Error("批次刪除信件失敗", zap.Error(err))
			continue
		}

		sendMailAck(sess, mailID, deleteAckType)
	}
}

// --- Packet builders ---

// sendMailList sends S_Mail (opcode 186) — mailbox list.
// Format: [C type][H count]{[D mailID][C readStatus][D dateSec][C isSender][S otherName][rawSubject]} × count
func sendMailList(sess *net.Session, player *world.PlayerInfo, mails []persist.MailRow, mailType int16) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_MAIL)
	w.WriteC(byte(mailType))
	w.WriteH(uint16(len(mails)))
	for _, m := range mails {
		w.WriteD(m.ID)
		w.WriteC(byte(m.ReadStatus))
		w.WriteD(int32(m.Date.Unix()))
		// isSender: 1 if this player is the sender, 0 if receiver
		if m.Sender == player.Name {
			w.WriteC(1)
			w.WriteS(m.Receiver)
		} else {
			w.WriteC(0)
			w.WriteS(m.Sender)
		}
		// Subject as raw bytes (Big5, terminated by 0x0000 already included)
		w.WriteBytes(m.Subject)
	}
	sess.Send(w.Bytes())
}

// sendMailContent sends S_Mail (opcode 186) — read mail content.
// Format: [C type][D mailID][rawContent]
func sendMailContent(sess *net.Session, mailID int32, readType byte, content []byte) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_MAIL)
	w.WriteC(readType)
	w.WriteD(mailID)
	if content != nil {
		w.WriteBytes(content)
	}
	sess.Send(w.Bytes())
}

// sendMailResult sends S_Mail (opcode 186) — send result.
// Format: [C type][C success]
func sendMailResult(sess *net.Session, mailType byte, success bool) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_MAIL)
	w.WriteC(mailType)
	if success {
		w.WriteC(1)
	} else {
		w.WriteC(0)
	}
	sess.Send(w.Bytes())
}

// sendMailAck sends S_Mail (opcode 186) — delete/move acknowledgement.
// Format: [C type][D mailID][C 1]
func sendMailAck(sess *net.Session, mailID int32, ackType byte) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_MAIL)
	w.WriteC(ackType)
	w.WriteD(mailID)
	w.WriteC(1)
	sess.Send(w.Bytes())
}

// sendMailNotify sends S_Mail (opcode 186) subtype 0x50 — new mail notification.
// Format: [C 0x50][D mailID][C isDraft][S senderName][rawSubject]
func sendMailNotify(sess *net.Session, senderName string, mailID int32, isDraft bool, subject []byte) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_MAIL)
	w.WriteC(0x50)
	w.WriteD(mailID)
	if isDraft {
		w.WriteC(1)
	} else {
		w.WriteC(0)
	}
	w.WriteS(senderName)
	if subject != nil {
		w.WriteBytes(subject)
	}
	sess.Send(w.Bytes())
}

// sendMailSound sends a skill sound effect (opcode 22) for new mail notification.
// Java uses skill sound ID 1091.
func sendMailSound(sess *net.Session, objID int32) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_SOUND_EFFECT)
	w.WriteD(objID)
	w.WriteH(1091)
	sess.Send(w.Bytes())
}

// sendSystemMessage sends a plain text system message via S_GlobalChat (opcode 243).
// Used for "not yet implemented" messages.
func sendSystemMessage(sess *net.Session, text string) {
	w := packet.NewWriterWithOpcode(packet.S_OPCODE_MESSAGE)
	w.WriteC(9) // system type
	w.WriteS(text)
	sess.Send(w.Bytes())
}

// parseMailText splits raw mail bytes by the 0x0000 double-null separator.
// Client sends: [subject bytes][0x00 0x00][content bytes][0x00 0x00]
// Returns subject (including its trailing 0x0000) and content bytes.
func parseMailText(raw []byte) (subject, content []byte) {
	sp1 := -1
	sp2 := -1

	// Scan for double-null separators at 2-byte boundaries
	for i := 0; i+1 < len(raw); i += 2 {
		if raw[i] == 0 && raw[i+1] == 0 {
			if sp1 < 0 {
				sp1 = i
			} else if sp2 < 0 {
				sp2 = i
				break
			}
		}
	}

	if sp1 < 0 {
		// No separator found: treat entire thing as subject
		return raw, nil
	}

	// Subject includes the separator (sp1 + 2 bytes)
	subject = raw[:sp1+2]

	if sp2 > sp1 {
		content = raw[sp1+2 : sp2]
	} else if sp1+2 < len(raw) {
		content = raw[sp1+2:]
	} else {
		content = []byte{0}
	}

	return subject, content
}
