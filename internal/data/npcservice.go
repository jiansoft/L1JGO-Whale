package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// NpcServiceTable holds all NPC service definitions loaded from YAML.
type NpcServiceTable struct {
	healers       map[int32]*HealerDef // npc_id → healer config
	cancel        CancelDef
	haste         HasteDef
	weaponEnchant WeaponEnchantDef
	armorEnchant  ArmorEnchantDef
	polymorph     PolymorphServiceDef
	polyForms     map[string]int32 // action string → poly_id
}

// HealerDef defines a healer NPC's behavior.
type HealerDef struct {
	NpcID    int32
	HealType string // "random" or "full"
	HealMin  int    // only for "random"
	HealMax  int    // only for "random"
	Target   string // "hp" or "hp_mp"
	Cost     int32
	Gfx      int32
	MsgID    uint16
}

// CancelDef defines the cancellation NPC.
type CancelDef struct {
	NpcID    int32
	MaxLevel int16
	Gfx      int32
}

// HasteDef defines the haste buffer NPC.
type HasteDef struct {
	NpcID       int32
	DurationSec int
	Gfx         int32
	MsgID       uint16
}

// WeaponEnchantDef defines weapon enchant NPC parameters.
type WeaponEnchantDef struct {
	DmgBonus    int16
	DurationSec int
	Gfx         int32
}

// ArmorEnchantDef defines armor enchant NPC parameters.
type ArmorEnchantDef struct {
	AcBonus     int16
	DurationSec int
	Gfx         int32
}

// PolymorphServiceDef defines polymorph NPC parameters.
type PolymorphServiceDef struct {
	Cost        int32
	DurationSec int
}

// GetHealer returns healer definition for a NPC ID, or nil if not a healer.
func (t *NpcServiceTable) GetHealer(npcID int32) *HealerDef {
	return t.healers[npcID]
}

// Cancel returns the cancellation NPC definition.
func (t *NpcServiceTable) Cancel() CancelDef { return t.cancel }

// Haste returns the haste NPC definition.
func (t *NpcServiceTable) Haste() HasteDef { return t.haste }

// WeaponEnchant returns weapon enchant parameters.
func (t *NpcServiceTable) WeaponEnchant() WeaponEnchantDef { return t.weaponEnchant }

// ArmorEnchant returns armor enchant parameters.
func (t *NpcServiceTable) ArmorEnchant() ArmorEnchantDef { return t.armorEnchant }

// Polymorph returns polymorph NPC parameters.
func (t *NpcServiceTable) Polymorph() PolymorphServiceDef { return t.polymorph }

// GetPolyForm returns the polymorph GFX ID for an action string, or 0 if not found.
func (t *NpcServiceTable) GetPolyForm(action string) int32 {
	return t.polyForms[action]
}

// Count returns total number of service definitions.
func (t *NpcServiceTable) Count() int {
	return len(t.healers) + len(t.polyForms) + 4 // +4 for cancel/haste/wenchant/aenchant
}

// --- YAML loading ---

type healerYAML struct {
	NpcID    int32  `yaml:"npc_id"`
	HealType string `yaml:"heal_type"`
	HealMin  int    `yaml:"heal_min"`
	HealMax  int    `yaml:"heal_max"`
	Target   string `yaml:"target"`
	Cost     int32  `yaml:"cost"`
	Gfx      int32  `yaml:"gfx"`
	MsgID    int    `yaml:"msg_id"`
}

type cancelYAML struct {
	NpcID    int32 `yaml:"npc_id"`
	MaxLevel int   `yaml:"max_level"`
	Gfx      int32 `yaml:"gfx"`
}

type hasteYAML struct {
	NpcID       int32 `yaml:"npc_id"`
	DurationSec int   `yaml:"duration_sec"`
	Gfx         int32 `yaml:"gfx"`
	MsgID       int   `yaml:"msg_id"`
}

type weaponEnchantYAML struct {
	DmgBonus    int `yaml:"dmg_bonus"`
	DurationSec int `yaml:"duration_sec"`
	Gfx         int32 `yaml:"gfx"`
}

type armorEnchantYAML struct {
	AcBonus     int `yaml:"ac_bonus"`
	DurationSec int `yaml:"duration_sec"`
	Gfx         int32 `yaml:"gfx"`
}

type polyFormYAML struct {
	Action string `yaml:"action"`
	PolyID int32  `yaml:"poly_id"`
}

type polymorphYAML struct {
	Cost        int32          `yaml:"cost"`
	DurationSec int            `yaml:"duration_sec"`
	Forms       []polyFormYAML `yaml:"forms"`
}

type npcServicesYAML struct {
	Healers       []healerYAML      `yaml:"healers"`
	Cancel        cancelYAML        `yaml:"cancel"`
	Haste         hasteYAML         `yaml:"haste"`
	WeaponEnchant weaponEnchantYAML `yaml:"weapon_enchant"`
	ArmorEnchant  armorEnchantYAML  `yaml:"armor_enchant"`
	Polymorph     polymorphYAML     `yaml:"polymorph"`
}

type npcServiceFile struct {
	Services npcServicesYAML `yaml:"npc_services"`
}

// LoadNpcServiceTable loads NPC service definitions from YAML.
func LoadNpcServiceTable(path string) (*NpcServiceTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read npc services: %w", err)
	}
	var f npcServiceFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse npc services: %w", err)
	}
	s := &f.Services

	t := &NpcServiceTable{
		healers:   make(map[int32]*HealerDef, len(s.Healers)),
		polyForms: make(map[string]int32, len(s.Polymorph.Forms)),
	}

	for _, h := range s.Healers {
		t.healers[h.NpcID] = &HealerDef{
			NpcID:    h.NpcID,
			HealType: h.HealType,
			HealMin:  h.HealMin,
			HealMax:  h.HealMax,
			Target:   h.Target,
			Cost:     h.Cost,
			Gfx:      h.Gfx,
			MsgID:    uint16(h.MsgID),
		}
	}

	t.cancel = CancelDef{
		NpcID:    s.Cancel.NpcID,
		MaxLevel: int16(s.Cancel.MaxLevel),
		Gfx:      s.Cancel.Gfx,
	}
	t.haste = HasteDef{
		NpcID:       s.Haste.NpcID,
		DurationSec: s.Haste.DurationSec,
		Gfx:         s.Haste.Gfx,
		MsgID:       uint16(s.Haste.MsgID),
	}
	t.weaponEnchant = WeaponEnchantDef{
		DmgBonus:    int16(s.WeaponEnchant.DmgBonus),
		DurationSec: s.WeaponEnchant.DurationSec,
		Gfx:         s.WeaponEnchant.Gfx,
	}
	t.armorEnchant = ArmorEnchantDef{
		AcBonus:     int16(s.ArmorEnchant.AcBonus),
		DurationSec: s.ArmorEnchant.DurationSec,
		Gfx:         s.ArmorEnchant.Gfx,
	}
	t.polymorph = PolymorphServiceDef{
		Cost:        s.Polymorph.Cost,
		DurationSec: s.Polymorph.DurationSec,
	}
	for _, form := range s.Polymorph.Forms {
		t.polyForms[form.Action] = form.PolyID
	}

	return t, nil
}
