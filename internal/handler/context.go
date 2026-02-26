package handler

import (
	"github.com/l1jgo/server/internal/config"
	"github.com/l1jgo/server/internal/core/event"
	"github.com/l1jgo/server/internal/data"
	"github.com/l1jgo/server/internal/net"
	"github.com/l1jgo/server/internal/net/packet"
	"github.com/l1jgo/server/internal/persist"
	"github.com/l1jgo/server/internal/scripting"
	"github.com/l1jgo/server/internal/world"
	"go.uber.org/zap"
)

// AttackRequest is queued by the handler and processed by CombatSystem in Phase 2.
type AttackRequest struct {
	AttackerSessionID uint64
	TargetID          int32
	IsMelee           bool // true=melee (C_ATTACK), false=ranged (C_FAR_ATTACK)
}

// CombatQueue accepts attack requests from handlers for deferred Phase 2 processing.
type CombatQueue interface {
	QueueAttack(req AttackRequest)
}

// SkillRequest is queued by the handler and processed by SkillSystem in Phase 2.
type SkillRequest struct {
	SessionID uint64
	SkillID   int32
	TargetID  int32
}

// SkillQueue accepts skill requests from handlers for deferred Phase 2 processing.
type SkillQueue interface {
	QueueSkill(req SkillRequest)
}

// NpcKillResult is returned by ProcessMeleeAttack/ProcessRangedAttack when an NPC
// dies. CombatSystem uses it to emit EntityKilled events on the bus.
type NpcKillResult struct {
	KillerSessionID uint64
	KillerCharID    int32
	NpcID           int32 // world NPC object ID
	NpcTemplateID   int32 // NPC template ID from spawn data
	ExpGained       int32
	MapID           int16
	X, Y            int32
}

// Deps holds shared dependencies injected into all packet handlers.
type Deps struct {
	AccountRepo *persist.AccountRepo
	CharRepo    *persist.CharacterRepo
	ItemRepo    *persist.ItemRepo
	Config      *config.Config
	Log         *zap.Logger
	World       *world.State
	Scripting   *scripting.Engine
	NpcActions  *data.NpcActionTable
	Items       *data.ItemTable
	Shops       *data.ShopTable
	Drops       *data.DropTable
	Teleports     *data.TeleportTable
	TeleportHtml  *data.TeleportHtmlTable
	Portals       *data.PortalTable
	Skills        *data.SkillTable
	Npcs          *data.NpcTable
	MobSkills      *data.MobSkillTable
	MapData        *data.MapDataTable
	Polys          *data.PolymorphTable
	ArmorSets      *data.ArmorSetTable
	SprTable       *data.SprTable
	WarehouseRepo  *persist.WarehouseRepo
	WALRepo        *persist.WALRepo
	ClanRepo       *persist.ClanRepo
	BuffRepo       *persist.BuffRepo
	Doors          *data.DoorTable
	ItemMaking     *data.ItemMakingTable
	SpellbookReqs  *data.SpellbookReqTable
	BuffIcons      *data.BuffIconTable
	NpcServices    *data.NpcServiceTable
	BuddyRepo     *persist.BuddyRepo
	ExcludeRepo   *persist.ExcludeRepo
	BoardRepo     *persist.BoardRepo
	MailRepo      *persist.MailRepo
	PetRepo       *persist.PetRepo
	PetTypes      *data.PetTypeTable
	PetItems      *data.PetItemTable
	Dolls         *data.DollTable
	TeleportPages *data.TeleportPageTable
	Combat        CombatQueue // filled after CombatSystem is created
	Skill         SkillQueue  // filled after SkillSystem is created
	Bus           *event.Bus  // event bus for emitting game events (EntityKilled, etc.)
	WeaponSkills  *data.WeaponSkillTable
}

// RegisterAll registers all packet handlers into the registry.
func RegisterAll(reg *packet.Registry, deps *Deps) {
	// Handshake phase
	reg.Register(packet.C_OPCODE_VERSION,
		[]packet.SessionState{packet.StateHandshake},
		func(sess any, r *packet.Reader) {
			HandleVersion(sess.(*net.Session), r, deps)
		},
	)

	// Login phase — BeanFun login (opcode 210) has action byte prefix
	reg.Register(packet.C_OPCODE_SHIFT_SERVER,
		[]packet.SessionState{packet.StateVersionOK},
		func(sess any, r *packet.Reader) {
			HandleAuthBeanFun(sess.(*net.Session), r, deps)
		},
	)
	// Direct login (opcode 119) — no action byte, just account\0 password\0
	reg.Register(packet.C_OPCODE_LOGIN,
		[]packet.SessionState{packet.StateVersionOK},
		func(sess any, r *packet.Reader) {
			HandleAuthDirect(sess.(*net.Session), r, deps)
		},
	)

	// Authenticated phase (character select screen)
	authStates := []packet.SessionState{packet.StateAuthenticated, packet.StateReturningToSelect}

	reg.Register(packet.C_OPCODE_CREATE_CUSTOM_CHARACTER, authStates,
		func(sess any, r *packet.Reader) {
			HandleCreateChar(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_DELETE_CHARACTER, authStates,
		func(sess any, r *packet.Reader) {
			HandleDeleteChar(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_ENTER_WORLD,
		[]packet.SessionState{packet.StateAuthenticated},
		func(sess any, r *packet.Reader) {
			HandleEnterWorld(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_REQUEST_ROLL,
		[]packet.SessionState{packet.StateAuthenticated, packet.StateInWorld, packet.StateReturningToSelect},
		func(sess any, r *packet.Reader) {
			HandleChangeChar(sess.(*net.Session), r, deps)
		},
	)

	// In-world phase
	inWorldStates := []packet.SessionState{packet.StateInWorld}

	reg.Register(packet.C_OPCODE_MOVE, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleMove(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_CHANGE_DIRECTION, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleChangeDirection(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_ATTR, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleAttr(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_DUEL, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleDuel(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_ATTACK, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleAttack(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_FAR_ATTACK, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleFarAttack(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_CHECK_PK, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleCheckPK(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_DIALOG, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleNpcTalk(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_HACTION, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleNpcAction(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_BUY_SELL, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleBuySell(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_CHAT, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleChat(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_SAY, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleSay(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_TELL, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleWhisper(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_USE_ITEM, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleUseItem(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_DESTROY_ITEM, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleDestroyItem(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_DROP, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleDropItem(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_GET, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandlePickupItem(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_FIXABLE_ITEM, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleFixWeaponList(sess.(*net.Session), r, deps)
		},
	)
	// C_FIX (118) = C_FixWeaponList in Java — same handler as C_FIXABLE_ITEM (254).
	// Both opcodes query the damaged weapon list; the client may send either one.
	reg.Register(packet.C_OPCODE_FIX, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleFixWeaponList(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_PERSONAL_SHOP, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleSelectList(sess.(*net.Session), r, deps)
		},
	)
	// Ship transport
	reg.Register(packet.C_OPCODE_ENTER_SHIP, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleEnterShip(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_RESTART, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleRestart(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_ACTION, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleAction(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_BOOKMARK, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleBookmark(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_DELETE_BOOKMARK, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleDeleteBookmark(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_PLATE, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleBoardOrPlate(sess.(*net.Session), r, deps)
		},
	)
	// Board (bulletin board)
	reg.Register(packet.C_OPCODE_BOARD_LIST, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleBoardBack(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_BOARD_READ, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleBoardRead(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_BOARD_WRITE, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleBoardWrite(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_BOARD_DELETE, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleBoardDelete(sess.(*net.Session), r, deps)
		},
	)

	// Mail
	reg.Register(packet.C_OPCODE_MAIL, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleMail(sess.(*net.Session), r, deps)
		},
	)

	reg.Register(packet.C_OPCODE_TELEPORT, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleTeleport(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_ENTER_PORTAL, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleEnterPortal(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_USE_SPELL, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleUseSpell(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_BUY_SPELL, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleBuySpell(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_BUYABLE_SPELL, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleBuyableSpell(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_SAVEIO, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleCharConfig(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_OPEN, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleOpen(sess.(*net.Session), r, deps)
		},
	)

	// Warehouse: all warehouse ops go through C_BUY_SELL (opcode 161) with resultType 2-9.
	// C_DEPOSIT(56) and C_WITHDRAW(44) are castle treasury opcodes, not warehouse.

	// Party
	// C_WHO_PARTY (230) = C_CreateParty in Java — party invite
	reg.Register(packet.C_OPCODE_WHO_PARTY, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleInviteParty(sess.(*net.Session), r, deps)
		},
	)
	// C_INVITE_PARTY_TARGET (43) = C_Party in Java — query party info
	reg.Register(packet.C_OPCODE_INVITE_PARTY_TARGET, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleWhoParty(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_LEAVE_PARTY, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleLeaveParty(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_BANISH_PARTY, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleBanishParty(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_CHAT_PARTY_CONTROL, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandlePartyControl(sess.(*net.Session), r, deps)
		},
	)

	// Clan
	reg.Register(packet.C_OPCODE_CREATE_PLEDGE, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleCreateClan(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_JOIN_PLEDGE, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleJoinClan(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_LEAVE_PLEDGE, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleLeaveClan(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_BAN_MEMBER, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleBanMember(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_WHO_PLEDGE, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleWhoPledge(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_PLEDGE_WATCH, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandlePledgeWatch(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_RANK_CONTROL, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleRankControl(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_TITLE, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleTitle(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_UPLOAD_EMBLEM, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleEmblemUpload(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_ALT_ATTACK, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleEmblemDownload(sess.(*net.Session), r, deps)
		},
	)

	// Polymorph (monlist dialog input)
	reg.Register(packet.C_OPCODE_HYPERTEXT_INPUT_RESULT, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleHypertextInputResult(sess.(*net.Session), r, deps)
		},
	)

	// Trade
	reg.Register(packet.C_OPCODE_ASK_XCHG, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleAskTrade(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_ADD_XCHG, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleAddTrade(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_ACCEPT_XCHG, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleAcceptTrade(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_CANCEL_XCHG, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleCancelTrade(sess.(*net.Session), r, deps)
		},
	)

	// Buddy / Friend list
	reg.Register(packet.C_OPCODE_QUERY_BUDDY, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleQueryBuddy(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_ADD_BUDDY, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleAddBuddy(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_REMOVE_BUDDY, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleRemoveBuddy(sess.(*net.Session), r, deps)
		},
	)

	// Exclude / Block list
	reg.Register(packet.C_OPCODE_EXCLUDE, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleExclude(sess.(*net.Session), r, deps)
		},
	)

	// Who online
	reg.Register(packet.C_OPCODE_WHO, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleWho(sess.(*net.Session), r, deps)
		},
	)

	// Give item to NPC/Pet
	reg.Register(packet.C_OPCODE_GIVE, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleGiveItem(sess.(*net.Session), r, deps)
		},
	)

	// Pet
	reg.Register(packet.C_OPCODE_CHECK_INVENTORY, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandlePetMenu(sess.(*net.Session), r, deps)
		},
	)
	reg.Register(packet.C_OPCODE_NPC_ITEM_CONTROL, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleUsePetItem(sess.(*net.Session), r, deps)
		},
	)

	// Mercenary (stub)
	reg.Register(packet.C_OPCODE_MERCENARYARRANGE, inWorldStates,
		func(sess any, r *packet.Reader) {
			HandleMercenaryArrange(sess.(*net.Session), r, deps)
		},
	)

	// Always allowed (any active state)
	aliveStates := []packet.SessionState{
		packet.StateVersionOK, packet.StateAuthenticated,
		packet.StateInWorld, packet.StateReturningToSelect,
	}
	reg.Register(packet.C_OPCODE_ALIVE, aliveStates,
		func(sess any, r *packet.Reader) {
			// Java: C_KeepALIVE sends S_GameTime to keep client time synced (day/night cycle).
			s := sess.(*net.Session)
			if s.State() == packet.StateInWorld {
				sendGameTime(s, world.GameTimeNow().Seconds())
			}
		},
	)
	reg.Register(packet.C_OPCODE_QUIT, aliveStates,
		func(sess any, r *packet.Reader) {
			HandleQuit(sess.(*net.Session), r, deps)
		},
	)
}
