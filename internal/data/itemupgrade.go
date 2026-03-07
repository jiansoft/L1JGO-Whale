package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ItemUpgrade 物品升級（火神煉化合成）定義。
type ItemUpgrade struct {
	ID            int32  `yaml:"id"`
	NpcID         int32  `yaml:"npc_id"`
	ActionID      string `yaml:"action_id"`
	ActionName    string `yaml:"action_name"`
	UpgradeChance int    `yaml:"upgrade_chance"` // 成功機率 %
	DeleteChance  int    `yaml:"delete_chance"`  // 失敗時刪除機率 %
	NewItemID     int32  `yaml:"new_item_id"`    // 升級後物品 ID
	MainItemID    int32  `yaml:"main_item_id"`   // 主物品 ID
	MainItemCount int32  `yaml:"main_item_count"`
	NeedItemIDs   []int32 `yaml:"need_item_ids"`   // 需要材料 IDs
	NeedCounts    []int32 `yaml:"need_counts"`     // 需要材料數量
	PlusItemIDs   []int32 `yaml:"plus_item_ids"`   // 加成材料 IDs（可選）
	PlusCounts    []int32 `yaml:"plus_counts"`     // 加成材料數量
	PlusAddChance []int   `yaml:"plus_add_chance"` // 每個加成材料的機率加成
	SuccessHTML   string `yaml:"success_html"`
	FailureHTML   string `yaml:"failure_html"`
	DeleteHTML    string `yaml:"delete_html"`
}

type upgradeKey struct {
	NpcID    int32
	ActionID string
}

type itemUpgradeFile struct {
	Upgrades []ItemUpgrade `yaml:"upgrades"`
}

// ItemUpgradeTable 物品升級資料表。
type ItemUpgradeTable struct {
	byAction map[upgradeKey]*ItemUpgrade
}

// Get 根據 NPC ID 和動作 ID 取得升級定義。
func (t *ItemUpgradeTable) Get(npcID int32, actionID string) *ItemUpgrade {
	return t.byAction[upgradeKey{NpcID: npcID, ActionID: actionID}]
}

// Count 回傳升級定義數。
func (t *ItemUpgradeTable) Count() int {
	return len(t.byAction)
}

// LoadItemUpgradeTable 載入物品升級資料。
func LoadItemUpgradeTable(path string) (*ItemUpgradeTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ItemUpgradeTable{byAction: make(map[upgradeKey]*ItemUpgrade)}, nil
		}
		return nil, fmt.Errorf("read item_upgrade: %w", err)
	}
	var f itemUpgradeFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse item_upgrade: %w", err)
	}
	t := &ItemUpgradeTable{byAction: make(map[upgradeKey]*ItemUpgrade, len(f.Upgrades))}
	for i := range f.Upgrades {
		u := &f.Upgrades[i]
		t.byAction[upgradeKey{NpcID: u.NpcID, ActionID: u.ActionID}] = u
	}
	return t, nil
}
