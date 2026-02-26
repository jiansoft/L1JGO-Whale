package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PetTypeInfo holds static data for a pet type loaded from YAML.
type PetTypeInfo struct {
	BaseNpcID    int32
	Name         string
	CanEquip     bool
	TamingItemID int32
	HPUpMin      int
	HPUpMax      int
	MPUpMin      int
	MPUpMax      int
	EvolvItemID  int32
	EvolvNpcID   int32
	MsgIDs       [5]int // level-up message IDs for tiers: 12, 24, 36, 48, 50+
	DefyMsgID    int
}

// CanTame returns true if this pet type can be tamed directly.
func (p *PetTypeInfo) CanTame() bool { return p.TamingItemID != 0 }

// CanEvolve returns true if this pet type can evolve further.
func (p *PetTypeInfo) CanEvolve() bool { return p.EvolvNpcID != 0 }

// LevelUpMsgID returns the message ID for the given pet level.
// Returns 0 if no message for this level tier.
func (p *PetTypeInfo) LevelUpMsgID(level int) int {
	switch {
	case level >= 50:
		return p.MsgIDs[4]
	case level >= 48:
		return p.MsgIDs[3]
	case level >= 36:
		return p.MsgIDs[2]
	case level >= 24:
		return p.MsgIDs[1]
	case level >= 12:
		return p.MsgIDs[0]
	default:
		return 0
	}
}

// PetTypeTable holds all pet type definitions indexed by BaseNpcID.
type PetTypeTable struct {
	types map[int32]*PetTypeInfo
}

// Get returns the pet type for the given NPC ID, or nil if not found.
func (t *PetTypeTable) Get(npcID int32) *PetTypeInfo {
	return t.types[npcID]
}

// Count returns the number of loaded pet types.
func (t *PetTypeTable) Count() int {
	return len(t.types)
}

// --- YAML loading ---

type petTypeEntry struct {
	BaseNpcID    int32  `yaml:"base_npc_id"`
	Name         string `yaml:"name"`
	CanEquip     bool   `yaml:"can_equip"`
	TamingItemID int32  `yaml:"taming_item_id"`
	HPUpMin      int    `yaml:"hp_up_min"`
	HPUpMax      int    `yaml:"hp_up_max"`
	MPUpMin      int    `yaml:"mp_up_min"`
	MPUpMax      int    `yaml:"mp_up_max"`
	EvolvItemID  int32  `yaml:"evolv_item_id"`
	EvolvNpcID   int32  `yaml:"evolv_npc_id"`
	MsgIDs       []int  `yaml:"msg_ids"`
	DefyMsgID    int    `yaml:"defy_msg_id"`
}

type petTypeListFile struct {
	PetTypes []petTypeEntry `yaml:"pet_types"`
}

// LoadPetTypeTable loads pet type definitions from a YAML file.
func LoadPetTypeTable(path string) (*PetTypeTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pet_types: %w", err)
	}
	var f petTypeListFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse pet_types: %w", err)
	}
	t := &PetTypeTable{types: make(map[int32]*PetTypeInfo, len(f.PetTypes))}
	for i := range f.PetTypes {
		e := &f.PetTypes[i]
		info := &PetTypeInfo{
			BaseNpcID:    e.BaseNpcID,
			Name:         e.Name,
			CanEquip:     e.CanEquip,
			TamingItemID: e.TamingItemID,
			HPUpMin:      e.HPUpMin,
			HPUpMax:      e.HPUpMax,
			MPUpMin:      e.MPUpMin,
			MPUpMax:      e.MPUpMax,
			EvolvItemID:  e.EvolvItemID,
			EvolvNpcID:   e.EvolvNpcID,
			DefyMsgID:    e.DefyMsgID,
		}
		// Copy up to 5 message IDs
		for j := 0; j < 5 && j < len(e.MsgIDs); j++ {
			info.MsgIDs[j] = e.MsgIDs[j]
		}
		t.types[e.BaseNpcID] = info
	}
	return t, nil
}
