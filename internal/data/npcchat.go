package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// NPC 聊天觸發時機常數（Java: L1NpcInstance CHAT_TIMING_*）
const (
	ChatTimingAppearance  = 0 // NPC 出現時
	ChatTimingDead        = 1 // NPC 死亡時
	ChatTimingHide        = 2 // NPC 解除隱身時
	ChatTimingGameTime    = 3 // 遊戲時間觸發（保留）
	ChatTimingUnderAttack = 4 // NPC 被攻擊時（首次）
)

// NpcChat NPC 定時聊天定義。
type NpcChat struct {
	NpcID          int32  `yaml:"npc_id"`
	ChatTiming     int    `yaml:"chat_timing"`
	Note           string `yaml:"note"`
	StartDelayTime int    `yaml:"start_delay_time"` // 首次延遲 ms
	ChatID1        string `yaml:"chat_id1"`
	ChatID2        string `yaml:"chat_id2"`
	ChatID3        string `yaml:"chat_id3"`
	ChatID4        string `yaml:"chat_id4"`
	ChatID5        string `yaml:"chat_id5"`
	ChatInterval   int    `yaml:"chat_interval"`   // 對話間隔 ms
	IsShout        bool   `yaml:"is_shout"`        // 大喊（全地圖）
	IsWorldChat    bool   `yaml:"is_world_chat"`   // 世界聊天
	IsRepeat       bool   `yaml:"is_repeat"`       // 是否重複
	RepeatInterval int    `yaml:"repeat_interval"` // 重複間隔 ms
}

// ChatIDs 回傳非空的聊天 ID 列表。
func (c *NpcChat) ChatIDs() []string {
	var ids []string
	for _, id := range []string{c.ChatID1, c.ChatID2, c.ChatID3, c.ChatID4, c.ChatID5} {
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

type npcChatKey struct {
	NpcID      int32
	ChatTiming int
}

type npcChatFile struct {
	Chats []NpcChat `yaml:"chats"`
}

// NpcChatTable NPC 聊天資料表。
type NpcChatTable struct {
	byKey map[npcChatKey]*NpcChat
}

// Get 根據 NPC ID 和觸發時機取得聊天定義。
func (t *NpcChatTable) Get(npcID int32, timing int) *NpcChat {
	return t.byKey[npcChatKey{NpcID: npcID, ChatTiming: timing}]
}

// Count 回傳聊天定義數。
func (t *NpcChatTable) Count() int {
	return len(t.byKey)
}

// LoadNpcChatTable 載入 NPC 聊天資料。
func LoadNpcChatTable(path string) (*NpcChatTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &NpcChatTable{byKey: make(map[npcChatKey]*NpcChat)}, nil
		}
		return nil, fmt.Errorf("read npc_chat: %w", err)
	}
	var f npcChatFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse npc_chat: %w", err)
	}
	t := &NpcChatTable{byKey: make(map[npcChatKey]*NpcChat, len(f.Chats))}
	for i := range f.Chats {
		c := &f.Chats[i]
		t.byKey[npcChatKey{NpcID: c.NpcID, ChatTiming: c.ChatTiming}] = c
	}
	return t, nil
}
