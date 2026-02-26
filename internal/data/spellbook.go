package data

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SpellbookReqTable holds spellbook level requirements per class and item ID range.
type SpellbookReqTable struct {
	entries []spellbookReqEntry
}

type spellbookReqEntry struct {
	ItemMin int32
	ItemMax int32
	Reqs    map[int16]int // classType → required level
}

// GetLevelReq returns the required character level to use a spellbook/crystal/tablet.
// Returns 0 if the class cannot use items in this range.
func (t *SpellbookReqTable) GetLevelReq(classType int16, itemID int32) int {
	for i := range t.entries {
		e := &t.entries[i]
		if itemID >= e.ItemMin && itemID <= e.ItemMax {
			return e.Reqs[classType] // returns 0 if class not in map
		}
	}
	return 0
}

// Count returns the number of loaded requirement entries.
func (t *SpellbookReqTable) Count() int {
	return len(t.entries)
}

// --- YAML loading ---

type spellbookReqYAML struct {
	ItemMin int32       `yaml:"item_min"`
	ItemMax int32       `yaml:"item_max"`
	Reqs    map[int]int `yaml:"reqs"` // YAML keys are int, converted to int16
}

type spellbookReqFile struct {
	Entries []spellbookReqYAML `yaml:"spellbook_level_reqs"`
}

// LoadSpellbookReqTable loads spellbook level requirements from YAML.
func LoadSpellbookReqTable(path string) (*SpellbookReqTable, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spellbook reqs: %w", err)
	}
	var f spellbookReqFile
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse spellbook reqs: %w", err)
	}
	t := &SpellbookReqTable{
		entries: make([]spellbookReqEntry, 0, len(f.Entries)),
	}
	for _, e := range f.Entries {
		reqs := make(map[int16]int, len(e.Reqs))
		for classType, level := range e.Reqs {
			reqs[int16(classType)] = level
		}
		t.entries = append(t.entries, spellbookReqEntry{
			ItemMin: e.ItemMin,
			ItemMax: e.ItemMax,
			Reqs:    reqs,
		})
	}
	return t, nil
}
