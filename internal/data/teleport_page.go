package data

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// TeleportPageDest is a single destination in a paginated teleport category.
type TeleportPageDest struct {
	Name     string `yaml:"name"`
	X        int32  `yaml:"x"`
	Y        int32  `yaml:"y"`
	MapID    int16  `yaml:"map_id"`
	ItemID   int32  `yaml:"item_id"`
	Price    int32  `yaml:"price"`
	MaxLevel int16  `yaml:"max_level"`
}

type teleportPageFile struct {
	NpcIDs     []int32                          `yaml:"npc_ids"`
	Categories map[string][]TeleportPageDest     `yaml:"categories"`
}

// TeleportPageTable holds paginated teleport data (Npc_Teleport system).
type TeleportPageTable struct {
	npcIDs     map[int32]bool
	categories []string                        // sorted category names
	dests      map[string][]TeleportPageDest   // category key → destinations
}

// LoadTeleportPageTable loads paginated teleport data from a YAML file.
func LoadTeleportPageTable(path string) (*TeleportPageTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read npc_teleport_page: %w", err)
	}
	var f teleportPageFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse npc_teleport_page: %w", err)
	}

	t := &TeleportPageTable{
		npcIDs: make(map[int32]bool, len(f.NpcIDs)),
		dests:  make(map[string][]TeleportPageDest, len(f.Categories)),
	}
	for _, id := range f.NpcIDs {
		t.npcIDs[id] = true
	}
	for cat, entries := range f.Categories {
		t.dests[cat] = entries
		t.categories = append(t.categories, cat)
	}
	sort.Strings(t.categories)
	return t, nil
}

// IsPageTeleportNpc returns true if this NPC uses the paginated teleport system.
func (t *TeleportPageTable) IsPageTeleportNpc(npcID int32) bool {
	return t.npcIDs[npcID]
}

// Categories returns the sorted list of category names.
func (t *TeleportPageTable) Categories() []string {
	return t.categories
}

// GetCategory returns the destinations for a given category, or nil if not found.
func (t *TeleportPageTable) GetCategory(cat string) []TeleportPageDest {
	return t.dests[cat]
}

// Count returns the total number of destinations across all categories.
func (t *TeleportPageTable) Count() int {
	n := 0
	for _, d := range t.dests {
		n += len(d)
	}
	return n
}
