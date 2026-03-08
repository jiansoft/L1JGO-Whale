package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// CastleArea 城堡攻城戰區域範圍。
type CastleArea struct {
	X1  int32 `yaml:"x1"`
	X2  int32 `yaml:"x2"`
	Y1  int32 `yaml:"y1"`
	Y2  int32 `yaml:"y2"`
	Map int16 `yaml:"map"`
}

// CastlePoint 城堡內座標點。
type CastlePoint struct {
	X   int32 `yaml:"x"`
	Y   int32 `yaml:"y"`
	Map int16 `yaml:"map"`
}

// CatapultSpawn 投石車生成點。
type CatapultSpawn struct {
	X    int32  `yaml:"x"`
	Y    int32  `yaml:"y"`
	Map  int16  `yaml:"map"`
	Side string `yaml:"side"` // "attack" 或 "defence"
}

// SubTower 副塔位置（亞丁城專用）。
type SubTower struct {
	X    int32  `yaml:"x"`
	Y    int32  `yaml:"y"`
	Map  int16  `yaml:"map"`
	Name string `yaml:"name"`
}

// CastleConfig 單座城堡的靜態地理配置。
type CastleConfig struct {
	ID           int32           `yaml:"id"`
	Name         string          `yaml:"name"`
	WarArea      CastleArea      `yaml:"war_area"`
	Tower        CastlePoint     `yaml:"tower"`
	InnerMap     int16           `yaml:"inner_map"`
	GetbackTown  int32           `yaml:"getback_town"`
	Catapults    []CatapultSpawn `yaml:"catapults"`
	SubTowers    []SubTower      `yaml:"sub_towers"`
	GateNpcs     []int32         `yaml:"gate_npcs"`
}

// CastleTable 城堡靜態資料索引表。
type CastleTable struct {
	byID         map[int32]*CastleConfig
	townCastleMap map[int32]int32 // town_id → castle_id
}

// LoadCastleTable 從 YAML 載入城堡靜態地理資料。
func LoadCastleTable(path string) (*CastleTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("讀取城堡資料: %w", err)
	}

	var file struct {
		Castles        []CastleConfig   `yaml:"castles"`
		TownCastleMap  map[int32]int32  `yaml:"town_castle_map"`
	}
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("解析城堡資料: %w", err)
	}

	t := &CastleTable{
		byID:          make(map[int32]*CastleConfig, len(file.Castles)),
		townCastleMap: file.TownCastleMap,
	}
	for i := range file.Castles {
		c := &file.Castles[i]
		t.byID[c.ID] = c
	}

	if t.townCastleMap == nil {
		t.townCastleMap = make(map[int32]int32)
	}

	return t, nil
}

// Get 依城堡 ID 取得靜態配置。
func (t *CastleTable) Get(castleID int32) *CastleConfig {
	return t.byID[castleID]
}

// GetCastleIDByTown 依城鎮 ID 取得對應城堡 ID（0=無對應城堡）。
func (t *CastleTable) GetCastleIDByTown(townID int32) int32 {
	return t.townCastleMap[townID]
}

// CheckInWarArea 判斷座標是否在指定城堡的攻城戰區域內。
func (t *CastleTable) CheckInWarArea(castleID int32, x, y int32, mapID int16) bool {
	c := t.byID[castleID]
	if c == nil {
		return false
	}
	a := c.WarArea
	return mapID == a.Map && x >= a.X1 && x <= a.X2 && y >= a.Y1 && y <= a.Y2
}

// CheckInWarAreaOrInner 判斷座標是否在攻城戰區域或內城地圖中。
func (t *CastleTable) CheckInWarAreaOrInner(castleID int32, x, y int32, mapID int16) bool {
	c := t.byID[castleID]
	if c == nil {
		return false
	}
	if c.InnerMap != 0 && mapID == c.InnerMap {
		return true
	}
	a := c.WarArea
	return mapID == a.Map && x >= a.X1 && x <= a.X2 && y >= a.Y1 && y <= a.Y2
}

// GetCastleIDByArea 依座標取得所在城堡 ID（0=不在任何城堡區域）。
func (t *CastleTable) GetCastleIDByArea(x, y int32, mapID int16) int32 {
	for id, c := range t.byID {
		if c.InnerMap != 0 && mapID == c.InnerMap {
			return id
		}
		a := c.WarArea
		if mapID == a.Map && x >= a.X1 && x <= a.X2 && y >= a.Y1 && y <= a.Y2 {
			return id
		}
	}
	return 0
}

// All 回傳所有城堡配置。
func (t *CastleTable) All() []*CastleConfig {
	result := make([]*CastleConfig, 0, len(t.byID))
	for _, c := range t.byID {
		result = append(result, c)
	}
	return result
}

// Count 回傳城堡總數。
func (t *CastleTable) Count() int {
	return len(t.byID)
}
