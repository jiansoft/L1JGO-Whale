package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// CraftMaterial represents a required input material for a crafting recipe.
type CraftMaterial struct {
	ItemID int32 `yaml:"item_id"`
	Amount int32 `yaml:"amount"`
}

// CraftOutput represents a produced output item from a crafting recipe.
type CraftOutput struct {
	ItemID int32 `yaml:"item_id"`
	Amount int32 `yaml:"amount"`
}

// CraftRecipe defines a single NPC crafting recipe.
type CraftRecipe struct {
	Action          string          `yaml:"action"`
	NpcID           int32           `yaml:"npc_id"`            // 0 = any NPC
	AmountInputable bool            `yaml:"amount_inputable"`
	Items           []CraftOutput   `yaml:"items"`
	Materials       []CraftMaterial `yaml:"materials"`
}

type itemMakingFile struct {
	Recipes []CraftRecipe `yaml:"recipes"`
}

// ItemMakingTable stores crafting recipes indexed by action string.
type ItemMakingTable struct {
	byAction map[string]*CraftRecipe
}

// LoadItemMakingTable loads crafting recipes from a YAML file.
func LoadItemMakingTable(path string) (*ItemMakingTable, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read item_making_list: %w", err)
	}
	var f itemMakingFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse item_making_list: %w", err)
	}

	t := &ItemMakingTable{
		byAction: make(map[string]*CraftRecipe, len(f.Recipes)),
	}
	for i := range f.Recipes {
		r := &f.Recipes[i]
		t.byAction[r.Action] = r
	}
	return t, nil
}

// Get returns the recipe for the given action string, or nil if not found.
func (t *ItemMakingTable) Get(action string) *CraftRecipe {
	if t == nil {
		return nil
	}
	return t.byAction[action]
}

// Count returns the total number of loaded recipes.
func (t *ItemMakingTable) Count() int {
	if t == nil {
		return 0
	}
	return len(t.byAction)
}
