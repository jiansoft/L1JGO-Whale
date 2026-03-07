package data

import (
	"fmt"
	"math/rand"
	"os"

	"gopkg.in/yaml.v3"
)

// BoxItem 寶箱可能掉落的單一物品。
type BoxItem struct {
	BoxItemID int32 `yaml:"box_item_id"`
	GetItemID int32 `yaml:"get_item_id"`
	RandomInt int32 `yaml:"random_int"` // 機率分母
	Random    int32 `yaml:"random"`     // 機率分子
	MinCount  int32 `yaml:"min_count"`
	MaxCount  int32 `yaml:"max_count"`
	Broadcast bool  `yaml:"broadcast"` // 全服公告
	Bless     int8  `yaml:"bless"`     // -1=不設定, 0=祝福, 1=普通, 2=詛咒
	Enchant   int8  `yaml:"enchant"`   // 強化值
}

// BoxAllItem 全給型寶箱的物品定義。
type BoxAllItem struct {
	BoxItemID int32 `yaml:"box_item_id"`
	GetItemID int32 `yaml:"get_item_id"`
	Count     int32 `yaml:"count"`
	UseType   int   `yaml:"use_type"` // 職業位元限制（0=不限）
	Bless     int8  `yaml:"bless"`
	Enchant   int8  `yaml:"enchant"`
	Broadcast bool  `yaml:"broadcast"`
}

// BoxKeyItem 鑰匙開啟寶箱的物品定義。
type BoxKeyItem struct {
	KeyItemID int32 `yaml:"key_item_id"`
	BoxItemID int32 `yaml:"box_item_id"`
	GetItemID int32 `yaml:"get_item_id"`
	RandomInt int32 `yaml:"random_int"`
	Random    int32 `yaml:"random"`
	MinCount  int32 `yaml:"min_count"`
	MaxCount  int32 `yaml:"max_count"`
	Broadcast bool  `yaml:"broadcast"`
	Bless     int8  `yaml:"bless"`
	Enchant   int8  `yaml:"enchant"`
}

type itemBoxFile struct {
	Items []BoxItem    `yaml:"items"`
	All   []BoxAllItem `yaml:"all_items"`
	Keys  []BoxKeyItem `yaml:"key_items"`
}

// ItemBoxTable 寶箱物品資料表。
type ItemBoxTable struct {
	boxMap    map[int32][]BoxItem    // box_item_id → 隨機抽取列表
	allMap    map[int32][]BoxAllItem // box_item_id → 全給列表
	keyMap    map[int32][]BoxKeyItem // key_item_id → 鑰匙開啟列表
}

// GetBox 取得隨機寶箱的可掉落物品列表。
func (t *ItemBoxTable) GetBox(boxItemID int32) []BoxItem {
	return t.boxMap[boxItemID]
}

// GetBoxAll 取得全給型寶箱的物品列表。
func (t *ItemBoxTable) GetBoxAll(boxItemID int32) []BoxAllItem {
	return t.allMap[boxItemID]
}

// GetBoxKey 取得鑰匙開啟的物品列表。
func (t *ItemBoxTable) GetBoxKey(keyItemID int32) []BoxKeyItem {
	return t.keyMap[keyItemID]
}

// Count 回傳寶箱種類數。
func (t *ItemBoxTable) Count() int {
	return len(t.boxMap) + len(t.allMap) + len(t.keyMap)
}

// RollBox 從隨機寶箱中抽取一個物品。
// Java: BoxRandom.runItem() — 最多嘗試 300 次。
func (t *ItemBoxTable) RollBox(boxItemID int32) *BoxItem {
	items := t.boxMap[boxItemID]
	if len(items) == 0 {
		return nil
	}

	candidates := make([]BoxItem, len(items))
	copy(candidates, items)

	for attempt := 0; attempt < 300 && len(candidates) > 0; attempt++ {
		idx := rand.Intn(len(candidates))
		item := &candidates[idx]

		if item.RandomInt <= 0 {
			return item
		}

		roll := rand.Int31n(item.RandomInt)
		if roll < item.Random {
			return item
		}

		// 低機率物品嘗試多次後移除避免無窮迴圈
		if item.Random <= 10000 && attempt >= 1 {
			candidates = append(candidates[:idx], candidates[idx+1:]...)
		}
	}

	// 300 次都沒中 → 隨機返回一個
	idx := rand.Intn(len(items))
	return &items[idx]
}

// LoadItemBoxTable 載入寶箱資料。
func LoadItemBoxTable(path string) (*ItemBoxTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ItemBoxTable{
				boxMap: make(map[int32][]BoxItem),
				allMap: make(map[int32][]BoxAllItem),
				keyMap: make(map[int32][]BoxKeyItem),
			}, nil
		}
		return nil, fmt.Errorf("read item_box: %w", err)
	}
	var f itemBoxFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse item_box: %w", err)
	}
	t := &ItemBoxTable{
		boxMap: make(map[int32][]BoxItem),
		allMap: make(map[int32][]BoxAllItem),
		keyMap: make(map[int32][]BoxKeyItem),
	}
	for _, item := range f.Items {
		t.boxMap[item.BoxItemID] = append(t.boxMap[item.BoxItemID], item)
	}
	for _, item := range f.All {
		t.allMap[item.BoxItemID] = append(t.allMap[item.BoxItemID], item)
	}
	for _, item := range f.Keys {
		t.keyMap[item.KeyItemID] = append(t.keyMap[item.KeyItemID], item)
	}
	return t, nil
}
