package data

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// PowerShopItem 代表強化物品商店中的一個商品（預設強化/附魔的物品）。
type PowerShopItem struct {
	ItemID       int32 `yaml:"item_id"`
	Price        int32 `yaml:"price"`         // 金幣售價
	EnchantLvl   int16 `yaml:"enchant_level"` // 強化等級
	Bless        int8  `yaml:"bless"`         // 祝福度
	AttrKind     int16 `yaml:"attr_kind"`     // 屬性強化種類
	AttrLevel    int16 `yaml:"attr_level"`    // 屬性強化等級
}

// PowerItemTable 儲存所有 NPC 的強化物品商店資料。
type PowerItemTable struct {
	byNpcID map[int32][]*PowerShopItem
}

type powerItemYAML struct {
	NpcID int32            `yaml:"npc_id"`
	Items []*PowerShopItem `yaml:"items"`
}

// LoadPowerItems 從 YAML 載入強化物品商店資料。
func LoadPowerItems(dataDir string) (*PowerItemTable, error) {
	t := &PowerItemTable{
		byNpcID: make(map[int32][]*PowerShopItem),
	}

	path := filepath.Join(dataDir, "power_items.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return t, nil
		}
		return nil, fmt.Errorf("讀取 power_items.yaml 失敗: %w", err)
	}

	var entries []powerItemYAML
	if err := yaml.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("解析 power_items.yaml 失敗: %w", err)
	}

	for _, e := range entries {
		t.byNpcID[e.NpcID] = e.Items
	}

	return t, nil
}

// Get 取得指定 NPC 的強化物品清單。
func (t *PowerItemTable) Get(npcID int32) []*PowerShopItem {
	if t == nil {
		return nil
	}
	return t.byNpcID[npcID]
}
