package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DoorGfx holds the visual/geometric template for a door sprite.
type DoorGfx struct {
	GfxID           int32  `yaml:"gfxid"`
	Note            string `yaml:"note"`
	Direction       int    `yaml:"direction"`         // 0 = / (NE-SW), 1 = \ (NW-SE)
	LeftEdgeOffset  int    `yaml:"left_edge_offset"`  // tile offset from center
	RightEdgeOffset int    `yaml:"right_edge_offset"` // tile offset from center
}

// DoorSpawn holds the spawn configuration for a single door instance.
type DoorSpawn struct {
	ID        int32 `yaml:"id"`
	GfxID     int32 `yaml:"gfxid"`
	X         int32 `yaml:"x"`
	Y         int32 `yaml:"y"`
	MapID     int16 `yaml:"map_id"`
	HP        int32 `yaml:"hp"`         // 0 = indestructible
	Keeper    int32 `yaml:"keeper"`     // clan keeper NPC ID, 0 = public
	IsOpening bool  `yaml:"is_opening"` // initial state
}

// DoorTable holds all door GFX templates and spawn data.
type DoorTable struct {
	gfxByID map[int32]*DoorGfx
	spawns  []DoorSpawn
}

// LoadDoorTable loads door GFX and spawn data from YAML files.
func LoadDoorTable(gfxPath, spawnPath string) (*DoorTable, error) {
	t := &DoorTable{
		gfxByID: make(map[int32]*DoorGfx),
	}

	// Load GFX templates
	gfxData, err := os.ReadFile(gfxPath)
	if err != nil {
		return nil, fmt.Errorf("read door gfx: %w", err)
	}
	var gfxFile struct {
		DoorGfxs []DoorGfx `yaml:"door_gfxs"`
	}
	if err := yaml.Unmarshal(gfxData, &gfxFile); err != nil {
		return nil, fmt.Errorf("parse door gfx: %w", err)
	}
	for i := range gfxFile.DoorGfxs {
		g := &gfxFile.DoorGfxs[i]
		t.gfxByID[g.GfxID] = g
	}

	// Load spawns
	spawnData, err := os.ReadFile(spawnPath)
	if err != nil {
		return nil, fmt.Errorf("read door spawn: %w", err)
	}
	var spawnFile struct {
		Doors []DoorSpawn `yaml:"doors"`
	}
	if err := yaml.Unmarshal(spawnData, &spawnFile); err != nil {
		return nil, fmt.Errorf("parse door spawn: %w", err)
	}
	t.spawns = spawnFile.Doors

	return t, nil
}

// GetGfx returns a door GFX template by ID.
func (t *DoorTable) GetGfx(gfxID int32) *DoorGfx {
	return t.gfxByID[gfxID]
}

// Spawns returns all door spawn entries.
func (t *DoorTable) Spawns() []DoorSpawn {
	return t.spawns
}

// Count returns the number of door spawn entries.
func (t *DoorTable) Count() int {
	return len(t.spawns)
}
