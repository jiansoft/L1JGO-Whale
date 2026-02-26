package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// BuffIconInfo describes how to send a buff icon to the client.
type BuffIconInfo struct {
	SkillID int32
	Type    string // "shield", "strup", "dexup", "aura", "invis", "wisdom", "blue_potion"
	Param   byte   // icon sub-type (used by shield/strup/dexup)
}

// BuffIconTable maps skill ID to icon display info.
type BuffIconTable struct {
	icons map[int32]*BuffIconInfo
}

// Get returns icon info for a skill ID, or nil if no icon mapping exists.
func (t *BuffIconTable) Get(skillID int32) *BuffIconInfo {
	return t.icons[skillID]
}

// Count returns the number of loaded icon mappings.
func (t *BuffIconTable) Count() int {
	return len(t.icons)
}

// --- YAML loading ---

type buffIconEntry struct {
	SkillID int32  `yaml:"skill_id"`
	Type    string `yaml:"type"`
	Param   int    `yaml:"param"`
}

type buffIconFile struct {
	Icons []buffIconEntry `yaml:"buff_icons"`
}

// LoadBuffIconTable loads buff icon mappings from YAML.
func LoadBuffIconTable(path string) (*BuffIconTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read buff icons: %w", err)
	}
	var f buffIconFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse buff icons: %w", err)
	}
	t := &BuffIconTable{
		icons: make(map[int32]*BuffIconInfo, len(f.Icons)),
	}
	for _, e := range f.Icons {
		t.icons[e.SkillID] = &BuffIconInfo{
			SkillID: e.SkillID,
			Type:    e.Type,
			Param:   byte(e.Param),
		}
	}
	return t, nil
}
