package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PetItemInfo holds static data for a pet equipment item loaded from YAML.
type PetItemInfo struct {
	ItemID   int32
	Name     string
	UseType  int // 0 = armor, 1 = tooth (weapon)
	IsHigher bool
	Hit      int
	Dmg      int
	AC       int
	AddStr   int
	AddCon   int
	AddDex   int
	AddInt   int
	AddWis   int
	AddHP    int
	AddMP    int
	AddSP    int
	MDef     int
}

// IsWeapon returns true if this is a weapon (tooth) item.
func (p *PetItemInfo) IsWeapon() bool { return p.UseType == 1 }

// IsArmor returns true if this is an armor item.
func (p *PetItemInfo) IsArmor() bool { return p.UseType == 0 }

// PetItemTable holds all pet equipment definitions indexed by ItemID.
type PetItemTable struct {
	items map[int32]*PetItemInfo
}

// Get returns the pet item for the given item ID, or nil if not a pet item.
func (t *PetItemTable) Get(itemID int32) *PetItemInfo {
	return t.items[itemID]
}

// Count returns the number of loaded pet items.
func (t *PetItemTable) Count() int {
	return len(t.items)
}

// --- YAML loading ---

type petItemEntry struct {
	ItemID   int32  `yaml:"item_id"`
	Name     string `yaml:"name"`
	UseType  string `yaml:"use_type"` // "tooth" or "armor"
	IsHigher bool   `yaml:"is_higher"`
	Hit      int    `yaml:"hit"`
	Dmg      int    `yaml:"dmg"`
	AC       int    `yaml:"ac"`
	AddStr   int    `yaml:"add_str"`
	AddCon   int    `yaml:"add_con"`
	AddDex   int    `yaml:"add_dex"`
	AddInt   int    `yaml:"add_int"`
	AddWis   int    `yaml:"add_wis"`
	AddHP    int    `yaml:"add_hp"`
	AddMP    int    `yaml:"add_mp"`
	AddSP    int    `yaml:"add_sp"`
	MDef     int    `yaml:"m_def"`
}

type petItemListFile struct {
	PetItems []petItemEntry `yaml:"pet_items"`
}

// LoadPetItemTable loads pet equipment definitions from a YAML file.
func LoadPetItemTable(path string) (*PetItemTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pet_items: %w", err)
	}
	var f petItemListFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse pet_items: %w", err)
	}
	t := &PetItemTable{items: make(map[int32]*PetItemInfo, len(f.PetItems))}
	for i := range f.PetItems {
		e := &f.PetItems[i]
		useType := 0 // armor
		if e.UseType == "tooth" {
			useType = 1 // weapon
		}
		t.items[e.ItemID] = &PetItemInfo{
			ItemID:   e.ItemID,
			Name:     e.Name,
			UseType:  useType,
			IsHigher: e.IsHigher,
			Hit:      e.Hit,
			Dmg:      e.Dmg,
			AC:       e.AC,
			AddStr:   e.AddStr,
			AddCon:   e.AddCon,
			AddDex:   e.AddDex,
			AddInt:   e.AddInt,
			AddWis:   e.AddWis,
			AddHP:    e.AddHP,
			AddMP:    e.AddMP,
			AddSP:    e.AddSP,
			MDef:     e.MDef,
		}
	}
	return t, nil
}
