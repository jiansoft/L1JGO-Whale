package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DollPowerDef describes a single bonus or skill granted by a magic doll.
//
// Power types (Type field):
//
//	"hp","mp","ac","str","dex","con","wis","int","cha"  — flat stat bonus (Value)
//	"hit","dmg","bow_hit","bow_dmg","sp","mr"           — combat bonus (Value)
//	"hpr","mpr"                                         — regen per tick (Value)
//	"hp_regen_tick","mp_regen_tick"                      — periodic regen (Value=amount, Param=interval_sec)
//	"fire_res","water_res","wind_res","earth_res"        — elemental resist (Value)
//	"dodge","weight","exp"                               — utility (Value, exp is %)
//	"dmg_reduction"                                      — damage reduction (Value)
//	"stun_resist","freeze_resist","sleep_resist"         — status resist (Value)
//	"stone_resist","blind_resist"                        — status resist (Value)
//	"skill"                                              — active skill (Value=skillID, Chance=probability%)
//	"speed"                                              — movement speed boost (no Value)
type DollPowerDef struct {
	Type   string // power type identifier
	Value  int    // primary bonus value
	Param  int    // secondary parameter (regen interval, etc.)
	Chance int    // trigger probability % (skill type only)
}

// DollDef holds static data for a magic doll type loaded from YAML.
type DollDef struct {
	ItemID   int32  // etcitem item ID that summons this doll
	GfxID    int32  // sprite/graphics ID
	NameID   string // client string table key or display name
	Name     string // display name
	Duration int    // lifetime in seconds
	Tier     int    // power tier (1-5)
	Powers   []DollPowerDef
}

// DollTable maps etcitem item ID to doll definitions.
type DollTable struct {
	dolls map[int32]*DollDef
}

// Get returns the doll definition for an item ID, or nil if not a doll item.
func (t *DollTable) Get(itemID int32) *DollDef {
	return t.dolls[itemID]
}

// Count returns the number of loaded doll definitions.
func (t *DollTable) Count() int {
	return len(t.dolls)
}

// --- YAML loading ---

type dollPowerEntry struct {
	Type   string `yaml:"type"`
	Value  int    `yaml:"value"`
	Param  int    `yaml:"param"`
	Chance int    `yaml:"chance"`
}

type dollEntry struct {
	ItemID   int32            `yaml:"item_id"`
	GfxID    int32            `yaml:"gfx_id"`
	NameID   string           `yaml:"name_id"`
	Name     string           `yaml:"name"`
	Duration int              `yaml:"duration"`
	Tier     int              `yaml:"tier"`
	Powers   []dollPowerEntry `yaml:"powers"`
}

type dollFile struct {
	Dolls []dollEntry `yaml:"dolls"`
}

// LoadDollTable loads magic doll definitions from a YAML file.
func LoadDollTable(path string) (*DollTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dolls: %w", err)
	}
	var f dollFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse dolls: %w", err)
	}
	t := &DollTable{dolls: make(map[int32]*DollDef, len(f.Dolls))}
	for i := range f.Dolls {
		e := &f.Dolls[i]
		def := &DollDef{
			ItemID:   e.ItemID,
			GfxID:    e.GfxID,
			NameID:   e.NameID,
			Name:     e.Name,
			Duration: e.Duration,
			Tier:     e.Tier,
			Powers:   make([]DollPowerDef, len(e.Powers)),
		}
		for j, p := range e.Powers {
			def.Powers[j] = DollPowerDef{
				Type:   p.Type,
				Value:  p.Value,
				Param:  p.Param,
				Chance: p.Chance,
			}
		}
		t.dolls[e.ItemID] = def
	}
	return t, nil
}
