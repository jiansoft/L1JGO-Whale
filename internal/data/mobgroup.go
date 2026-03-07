package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// MinionEntry 群體中的隊員定義。
type MinionEntry struct {
	NpcID int32 `yaml:"npc_id"`
	Count int   `yaml:"count"`
}

// MobGroup 怪物群體定義。
// Java: L1MobGroup — 一個 leader + 最多 7 種 minion。
type MobGroup struct {
	ID                     int32         `yaml:"id"`
	Note                   string        `yaml:"note"`
	LeaderID               int32         `yaml:"leader_id"`
	RemoveGroupIfLeaderDie bool          `yaml:"remove_group_if_leader_die"`
	Minions                []MinionEntry `yaml:"minions"`
}

type mobGroupFile struct {
	Groups []MobGroup `yaml:"mobgroups"`
}

// MobGroupTable 怪物群體資料表。
type MobGroupTable struct {
	byID map[int32]*MobGroup
}

// Get 根據群體 ID 取得群體定義。
func (t *MobGroupTable) Get(groupID int32) *MobGroup {
	return t.byID[groupID]
}

// Count 回傳群體定義數。
func (t *MobGroupTable) Count() int {
	return len(t.byID)
}

// LoadMobGroupTable 載入怪物群體資料。
func LoadMobGroupTable(path string) (*MobGroupTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &MobGroupTable{byID: make(map[int32]*MobGroup)}, nil
		}
		return nil, fmt.Errorf("read mobgroup: %w", err)
	}
	var f mobGroupFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse mobgroup: %w", err)
	}
	t := &MobGroupTable{byID: make(map[int32]*MobGroup, len(f.Groups))}
	for i := range f.Groups {
		g := &f.Groups[i]
		t.byID[g.ID] = g
	}
	return t, nil
}
