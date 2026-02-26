package handler

import (
	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/net/packet"
)

// HandleMercenaryArrange processes C_MERCENARYARRANGE (opcode 129).
// The mercenary system is largely unimplemented in the Java reference as well.
// This stub prevents "unhandled opcode" log warnings.
func HandleMercenaryArrange(sess *net.Session, r *packet.Reader, deps *Deps) {
	// Stub — mercenary system not implemented.
	// Java reference: C_MercenaryArrange.java is essentially empty.
	_ = r
}
