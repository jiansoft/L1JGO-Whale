package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// WarGiftItem 攻城戰禮物物品。
type WarGiftItem struct {
	ItemID int32 `yaml:"item_id"`
	Count  int32 `yaml:"count"`
}

// WarGiftEntry 單座城堡的攻城戰禮物設定。
type WarGiftEntry struct {
	CastleID int32         `yaml:"castle_id"`
	Items    []WarGiftItem `yaml:"items"`
}

// WarGiftTable 攻城戰禮物索引表。
type WarGiftTable struct {
	byCastle map[int32]*WarGiftEntry
}

// LoadWarGiftTable 從 YAML 載入攻城戰禮物資料。
func LoadWarGiftTable(path string) (*WarGiftTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("讀取攻城戰禮物資料: %w", err)
	}

	var file struct {
		Gifts []WarGiftEntry `yaml:"gifts"`
	}
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("解析攻城戰禮物資料: %w", err)
	}

	t := &WarGiftTable{
		byCastle: make(map[int32]*WarGiftEntry, len(file.Gifts)),
	}
	for i := range file.Gifts {
		g := &file.Gifts[i]
		t.byCastle[g.CastleID] = g
	}

	return t, nil
}

// GetByCastle 依城堡 ID 取得禮物設定（nil=無禮物）。
func (t *WarGiftTable) GetByCastle(castleID int32) *WarGiftEntry {
	return t.byCastle[castleID]
}

// Count 回傳設定的城堡數。
func (t *WarGiftTable) Count() int {
	return len(t.byCastle)
}
